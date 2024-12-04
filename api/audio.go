package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"talkliketv.click/tltv/internal/models"

	"github.com/labstack/echo/v4"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/util"
)

// AudioFromFile accepts a file in srt, phrase per line, or paragraph form and
// sends a zip file of mp3 audio tracks for learning a language that you choose
func (s *Server) AudioFromFile(e echo.Context) error {
	// TODO add more oapi validation
	// get values from multipart form
	titleName := e.FormValue("titleName")
	// convert strings from multipart form to int16's
	fileLangId, err := strconv.Atoi(e.FormValue("fileLanguageId"))
	if err != nil {
		return e.String(http.StatusBadRequest, fmt.Sprintf("error converting fileLanguageId to int16: %s", err.Error()))
	}
	toVoiceId, err := strconv.Atoi(e.FormValue("toVoiceId"))
	if err != nil {
		return e.String(http.StatusBadRequest, fmt.Sprintf("error converting toVoiceId to int16: %s", err.Error()))
	}
	fromVoiceId, err := strconv.Atoi(e.FormValue("fromVoiceId"))
	if err != nil {
		return e.String(http.StatusBadRequest, fmt.Sprintf("error converting fromVoiceId to int16: %s", err.Error()))
	}

	pause := s.config.PhrasePause
	// check if user sent 'pause' in the request and update config if they did
	pauseForm := e.FormValue("pause")
	if pauseForm != "" {
		pauseInt, err := strconv.Atoi(pauseForm)
		if err != nil {
			return e.String(http.StatusBadRequest, fmt.Sprintf("error converting fromVoiceId to int: %s", err.Error()))
		}
		if pauseInt > 10 || pauseInt < 3 {
			return e.String(http.StatusBadRequest, fmt.Sprintf("pause must be between 3 and 10: %d", pauseInt))
		}
		pause = pauseInt
	}

	pattern := s.config.AudioPattern
	patternForm := e.FormValue("pattern")
	if patternForm != "" {
		patternInt, err := strconv.Atoi(patternForm)
		if err != nil {
			return e.String(http.StatusBadRequest, fmt.Sprintf("error converting pattern to int: %s", err.Error()))
		}
		if patternInt > 3 || patternInt < 1 {
			return e.String(http.StatusBadRequest, fmt.Sprintf("pattern must be between 1 and 3: %d", patternInt))
		}
		pattern = patternInt
	}

	phrases, phraseZipFile, err := s.processFile(e, titleName)
	if err != nil {
		if errors.Is(err, util.ErrTooManyPhrases) {
			return e.Attachment(phraseZipFile.Name(), "TooManyPhrasesUseTheseFiles.zip")
		}
		if strings.Contains(err.Error(), "unable to parse file") {
			return e.String(http.StatusBadRequest, err.Error())
		}
		return e.String(http.StatusInternalServerError, err.Error())
	}

	title := models.Title{
		Name:        titleName,
		TitleLangId: fileLangId,
		ToVoiceId:   toVoiceId,
		FromVoiceId: fromVoiceId,
		Pause:       pause,
		Phrases:     phrases,
		Pattern:     pattern,
	}
	zipFile, err := s.createAudioFromTitle(e, title)
	if err != nil {
		if errors.Is(err, util.ErrVoiceLangIdNoMatch) {
			return e.String(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, util.ErrVoiceIdInvalid) {
			return e.String(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, util.ErrOneFile) {
			return e.Attachment(zipFile.Name(), titleName+".mp3")
		}
		return e.String(http.StatusInternalServerError, err.Error())
	}
	return e.Attachment(zipFile.Name(), titleName+".zip")
}

func (s *Server) processFile(e echo.Context, titleName string) ([]models.Phrase, *os.File, error) {
	// Get file handler for filename, size and headers
	fh, err := e.FormFile("filePath")
	if err != nil {
		e.Logger().Error(err)
		return nil, nil, util.ErrUnableToParseFile(err)
	}

	// Check if file size is too large 64000 == 8KB ~ approximately 4 pages of text
	if fh.Size > s.config.FileUploadLimit {
		rString := fmt.Sprintf("file too large (%d > %d)", fh.Size, s.config.FileUploadLimit)
		return nil, nil, util.ErrUnableToParseFile(errors.New(rString))
	}
	src, err := fh.Open()
	if err != nil {
		e.Logger().Error(err)
		return nil, nil, err
	}
	defer src.Close()

	// get an array of all the phrases from the uploaded file
	stringsSlice, err := s.af.GetLines(e, src)
	if err != nil {
		return nil, nil, util.ErrUnableToParseFile(err)
	}
	// send back zip of split files of phrase that requester can use if too big
	if len(stringsSlice) > s.config.MaxNumPhrases {
		chunkedPhrases := slices.Chunk(stringsSlice, s.config.MaxNumPhrases)
		phrasesBasePath := s.config.TTSBasePath + titleName + "/"
		// create zip of phrases files of maxNumPhrases for user to use instead of uploaded file
		zipFile, err := s.af.CreatePhrasesZip(e, chunkedPhrases, phrasesBasePath, titleName)
		if err != nil {
			e.Logger().Error(err)
			return nil, nil, err
		}
		return nil, zipFile, util.ErrTooManyPhrases
	}

	phrases := make([]models.Phrase, len(stringsSlice))
	for i := range stringsSlice {
		phrases[i] = models.Phrase{
			ID:   i,
			Text: stringsSlice[i],
		}
	}
	return phrases, nil, nil
}

// createAudioFromTitle is a helper function that performs the tasks shared by
// AudioFromFile and AudioFromTitle
func (s *Server) createAudioFromTitle(e echo.Context, title models.Title) (*os.File, error) {
	fromVoice, ok := models.Voices[title.FromVoiceId]
	if !ok {
		return nil, util.ErrVoiceIdInvalid
	}
	toVoice, ok := models.Voices[title.ToVoiceId]
	if !ok {
		return nil, util.ErrVoiceIdInvalid
	}
	// TODO if you don't want these files to persist then you need to defer removing them from calling function
	audioBasePath := s.config.TTSBasePath + title.Name

	fromAudioBasePath := fmt.Sprintf("%s%d/", audioBasePath, fromVoice.LangId)
	toAudioBasePath := fmt.Sprintf("%s%d/", audioBasePath, toVoice.LangId)

	if _, err := s.translates.CreateTTS(e, title, title.FromVoiceId, fromAudioBasePath); err != nil {
		e.Logger().Error(err)
		// if error remove all the text-to-speech created up to that point
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}

	tr, err := s.translates.CreateTTS(e, title, title.ToVoiceId, toAudioBasePath)
	if err != nil {
		e.Logger().Error(err)
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}

	title.Translates = tr
	pausePath, ok := audiofile.AudioPauseFilePath[title.Pause]
	if !ok {
		e.Logger().Error(errors.New("audio pause file not found"))
		return nil, err
	}
	fullPausePath := s.config.TTSBasePath + pausePath

	tmpDirPath := fmt.Sprintf("%s%s-%s/", s.config.TTSBasePath, title.Name, test.RandomString(4))
	err = os.MkdirAll(tmpDirPath, 0777)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	if err = s.af.BuildAudioInputFiles(e, title, fullPausePath, fromAudioBasePath, toAudioBasePath, tmpDirPath); err != nil {
		return nil, err
	}

	return s.af.CreateMp3Zip(e, title, tmpDirPath)
}
