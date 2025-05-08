package api

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/services"
	"talkliketv.click/tltv/internal/services/audiofile"
)

func (s *Server) ParseFile(e echo.Context) error {
	fh, err := e.FormFile("file_path")
	if err != nil {
		return e.String(http.StatusBadRequest, "error getting form file: "+err.Error())
	}
	stringsSlice, err := audiofile.FileParse(e, s.af, s.config.FileUploadLimit)
	if err != nil {
		if services.IsFileTooLargeError(err) {
			return e.String(http.StatusBadRequest, "error parsing file: "+err.Error())
		}
		return e.String(http.StatusInternalServerError, "error parsing file: "+err.Error())
	}

	zippedFile, err := audiofile.ZipStringsSlice(e, s.af, stringsSlice, s.config.MaxNumPhrases, s.config.TTSBasePath, fh.Filename)
	if err != nil {
		return e.String(http.StatusInternalServerError, "error zipping file: "+err.Error())
	}
	return e.Attachment(zippedFile.Name(), fh.Filename+"_parsed.zip")
}

func (s *Server) AudioFromFile(e echo.Context) error {
	if err := s.tokens.CheckToken(e.Request().Context(), e.FormValue("token")); err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusForbidden, "invalid token: "+err.Error())
	}

	fh, err := e.FormFile("file_path")
	if err != nil {
		return e.String(http.StatusBadRequest, "error getting form file: "+err.Error())
	}

	if fh.Size > s.config.FileUploadLimit {
		return e.String(http.StatusBadRequest, "file too large")
	}

	src, err := fh.Open()
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusBadRequest, "error opening file: "+err.Error())
	}
	// make sure the file has been parsed before continuing
	filetype, err := audiofile.DetectTextFormat(src)
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusBadRequest, "error detecting file type: "+err.Error())
	}
	if filetype != audiofile.OnePhrasePerLine {
		return e.String(http.StatusBadRequest, "Please parse file before uploading")
	}
	src.Close()

	title, fromVoice, toVoice, err := services.ValidateAudioRequest(e, s.m)
	if err != nil {
		return e.String(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	phrases, phraseZipFile, err := audiofile.ProcessFile(e, s.af, s.config, title.Name)
	if err != nil {
		if errors.Is(err, models.ErrTooManyPhrases) {
			return e.Attachment(phraseZipFile.Name(), "TooManyPhrasesUseTheseFiles")
		}
		if services.IsFileTooLargeError(err) {
			return e.String(http.StatusBadRequest, err.Error())
		}
		e.Logger().Error(err)
		return e.String(http.StatusInternalServerError, "unable to process file: "+err.Error())
	}

	var phraseTexts []string
	for i := 0; i < len(phrases) && i < 3; i++ {
		phraseTexts = append(phraseTexts, phrases[i].Text)
	}
	detectedFileLanguage, err := s.translate.DetectLanguage(e.Request().Context(), phraseTexts)
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusInternalServerError, "unable to detect language: "+err.Error())
	}

	title.TitleLang = detectedFileLanguage.String()
	title.TitlePhrases = phrases
	zipFile, err := audiofile.AudioFromTitle(e, s.translate, s.af, *fromVoice, *toVoice, *title, s.config.TTSBasePath)
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusInternalServerError, "unable to create audio file: "+err.Error())
	}

	if err := s.tokens.UpdateField(e.Request().Context(), true, e.FormValue("token"), "UploadUsed"); err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusInternalServerError, "unable to update token: "+err.Error())
	}

	titleName := fmt.Sprintf("%s.%s-%s.zip", title.Name, title.TitleLang, title.ToVoice)
	return e.Attachment(zipFile.Name(), titleName)
}
