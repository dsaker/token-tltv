package api

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"strings"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/models"
)

// ParseFile takes a file and parses it into the phrases that will be used to
// create the audio mp3 files.
// This allows the user to check to make sure the phrases are parsed correctly
// before uploading the file.
func (s *Server) ParseFile(e echo.Context) error {
	stringsSlice, err := audiofile.FileParse(e, s.af, s.config.FileUploadLimit)
	if err != nil {
		return e.String(http.StatusInternalServerError, "error parsing file"+err.Error())
	}

	// Get file handler for filename, size and headers
	fh, err := e.FormFile("file_path")
	if err != nil {
		return e.String(http.StatusBadRequest, "error getting form file"+err.Error())
	}

	zippedFile, err := audiofile.ZipStringsSlice(e, s.af, stringsSlice, s.config.MaxNumPhrases, s.config.TTSBasePath, fh.Filename)
	if err != nil {
		return e.String(http.StatusInternalServerError, "error zipping file"+err.Error())
	}
	return e.Attachment(zippedFile.Name(), fh.Filename+"_parsed.zip")
}

// AudioFromFile accepts a file in srt, phrase per line, or paragraph form and
// sends a zip file of mp3 audio tracks for learning a language that you choose
func (s *Server) AudioFromFile(e echo.Context) error {
	token := e.FormValue("token")
	// check token
	if err := s.tokens.CheckToken(e.Request().Context(), token); err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusForbidden, "invalid token: "+err.Error())
	}

	title, err := s.validateAudioRequest(e)
	if err != nil {
		return e.String(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	// TODO put limit on characters
	phrases, phraseZipFile, err := audiofile.ProcessFile(e, s.af, s.config, title.Name)
	if err != nil {
		if errors.Is(err, models.ErrTooManyPhrases) {
			return e.Attachment(phraseZipFile.Name(), "TooManyPhrasesUseTheseFiles")
		}
		e.Logger().Error(err)
		if strings.Contains(err.Error(), "unable to parsefile file") {
			return e.String(http.StatusBadRequest, "unable to parsefile file: "+err.Error())
		}
		return e.String(http.StatusInternalServerError, "unable to process file: "+err.Error())
	}

	title.TitlePhrases = phrases

	zipFile, err := audiofile.AudioFromTitle(e, s.translate, s.af, *title, s.config.TTSBasePath)
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusInternalServerError, "unable to create audio file: "+err.Error())
	}

	// change token status to Used
	err = s.tokens.UpdateField(e.Request().Context(), true, token, "UploadUsed")
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusInternalServerError, "unable to update token: "+err.Error())
	}

	// TODO change Id's to language codes
	titleName := title.Name + "." + strconv.Itoa(title.FromVoiceId) + "-" + strconv.Itoa(title.ToVoiceId) + ".zip"
	return e.Attachment(zipFile.Name(), titleName)
}

func (s *Server) validateAudioRequest(e echo.Context) (*models.Title, error) {
	// get values from multipart form
	titleName := e.FormValue("title_name")
	// convert strings from multipart form to int
	fileLangId, err := strconv.Atoi(e.FormValue("file_language_id"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting file_language_id to int: %s", err.Error())
	}

	// validate fileLangId
	_, err = s.m.GetLanguage(fileLangId)
	if err != nil {
		e.Logger().Error(models.ErrLanguageIdInvalid)
		return nil, models.ErrLanguageIdInvalid
	}

	toVoiceId, err := strconv.Atoi(e.FormValue("to_voice_id"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting to_voice_id to int: %s", err.Error())
	}
	// validate toVoiceId
	_, err = s.m.GetVoice(toVoiceId)
	if err != nil {
		e.Logger().Error(models.ErrVoiceIdInvalid)
		return nil, models.ErrVoiceIdInvalid
	}
	fromVoiceId, err := strconv.Atoi(e.FormValue("from_voice_id"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting from_voice_id to int: %s", err.Error())
	}
	// valid fromVoiceId
	_, err = s.m.GetVoice(fromVoiceId)
	if err != nil {
		e.Logger().Error(models.ErrVoiceIdInvalid)
		return nil, models.ErrVoiceIdInvalid
	}

	pause, err := strconv.Atoi(e.FormValue("pause"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting pause to int: %s", err.Error())
	}
	// validate pause
	if pause < 3 || pause > 10 {
		pauseError := errors.New("pause must be between 3 and 10")
		e.Logger().Error(pauseError)
		return nil, pauseError
	}

	// title-input
	titleInput := e.FormValue("title_name")
	// validate title name
	if len(titleInput) < 5 || len(titleInput) > 32 {
		titleInputError := errors.New("title_name must be between 5 and 32")
		e.Logger().Error(titleInputError)
		return nil, titleInputError
	}

	//
	pattern, err := strconv.Atoi(e.FormValue("pattern"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting pattern to int: %s", err.Error())
	}
	// validate pause
	if pattern < 1 || pattern > 3 {
		patternError := errors.New("pattern must be between 1 and 3")
		e.Logger().Error(patternError)
		return nil, patternError
	}

	return &models.Title{
		Name:         titleName,
		TitleLangId:  fileLangId,
		ToVoiceId:    toVoiceId,
		FromVoiceId:  fromVoiceId,
		Pause:        pause,
		TitlePhrases: nil,
		ToPhrases:    nil,
		Pattern:      pattern,
	}, nil
}
