package api

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/util"
)

func homeView(e echo.Context) error {
	return e.Render(http.StatusOK, "home.gohtml", nil)
}

// Update the handler so it displays the signup page.
func audioView(e echo.Context) error {
	return e.Render(http.StatusOK, "audio.gohtml", newTemplateData(""))
}

// ParseFile takes a file and parses it into the phrases that will be used to
// create the audio mp3 files.
// This allows the user to check to make sure the phrases are parsed correctly
// before uploading the file.
func (s *Server) ParseFile(e echo.Context) error {
	stringsSlice, err := s.parseFile(e)
	if err != nil {
		return e.String(http.StatusBadRequest, err.Error())
	}

	// Get file handler for filename, size and headers
	fh, err := e.FormFile("filePath")
	if err != nil {
		e.Logger().Error(err)
		return e.String(http.StatusBadRequest, err.Error())
	}

	zippedFile, err := s.zipStringsSlice(e, stringsSlice, fh.Filename)
	if err != nil {
		return e.String(http.StatusBadRequest, err.Error())
	}
	return e.Attachment(zippedFile.Name(), zippedFile.Name()+".zip")
}

// AudioFromFile accepts a file in srt, phrase per line, or paragraph form and
// sends a zip file of mp3 audio tracks for learning a language that you choose
func (s *Server) AudioFromFile(e echo.Context) error {

	token := e.FormValue("token")
	// check token
	if err := models.CheckToken(token); err != nil {
		return e.Render(http.StatusForbidden, "audio.gohtml", newTemplateData(err.Error()))
	}

	title, err := validateAudioRequest(e)
	if err != nil {
		return e.Render(http.StatusBadRequest, "audio.gohtml", newTemplateData(err.Error()))
	}

	// TODO put limit on characters
	phrases, phraseZipFile, err := s.processFile(e, title.Name)
	if err != nil {
		if errors.Is(err, util.ErrTooManyPhrases) {
			return e.Attachment(phraseZipFile.Name(), "TooManyPhrasesUseTheseFiles.zip")
		}
		if strings.Contains(err.Error(), "unable to parse file") {
			return e.Render(http.StatusBadRequest, "audio.gohtml", newTemplateData(err.Error()))

		}
		return e.Render(http.StatusInternalServerError, "audio.gohtml", newTemplateData(err.Error()))
	}

	// TODO add to and from languages to titleName
	titleName := title.Name + "." + strconv.Itoa(title.FromVoiceId) + "-" + strconv.Itoa(title.ToVoiceId) + ".zip"

	title.TitlePhrases = phrases

	zipFile, err := s.createAudioFromTitle(e, *title)
	if err != nil {
		return e.Render(http.StatusInternalServerError, "audio.gohtml", newTemplateData(err.Error()))
	}

	titleName = titleName + "." + strconv.Itoa(title.FromVoiceId) + "-" + strconv.Itoa(title.ToVoiceId) + ".zip"

	// change token status to Used
	err = models.SetTokenStatus(token, models.Used)
	if err != nil {
		return e.Render(http.StatusInternalServerError, "audio.gohtml", newTemplateData(err.Error()))
	}
	return e.Attachment(zipFile.Name(), titleName)
}

func (s *Server) parseFile(e echo.Context) ([]string, error) {
	// Get file handler for filename, size and headers
	fh, err := e.FormFile("file_path")
	if err != nil {
		e.Logger().Error(err)
		return nil, util.ErrUnableToParseFile(err)
	}

	// Check if file size is too large 64000 == 8KB ~ approximately 4 pages of text
	if fh.Size > s.config.FileUploadLimit {
		rString := fmt.Sprintf("file too large (%d > %d)", fh.Size, s.config.FileUploadLimit)
		return nil, util.ErrUnableToParseFile(errors.New(rString))
	}
	src, err := fh.Open()
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	defer src.Close()

	// get an array of all the phrases from the uploaded file
	stringsSlice, err := s.af.GetLines(e, src)
	if err != nil {
		return nil, util.ErrUnableToParseFile(err)
	}

	return stringsSlice, nil
}

func (s *Server) zipStringsSlice(e echo.Context, slice []string, name string) (*os.File, error) {
	chunkedPhrases := slices.Chunk(slice, s.config.MaxNumPhrases)
	phrasesBasePath := s.config.TTSBasePath + name + "/"
	// create zip of phrases files of maxNumPhrases for user to use instead of uploaded file
	zipFile, err := s.af.CreatePhrasesZip(e, chunkedPhrases, phrasesBasePath, name)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}
	return zipFile, nil
}

func (s *Server) processFile(e echo.Context, titleName string) ([]models.Phrase, *os.File, error) {
	stringsSlice, err := s.parseFile(e)
	if err != nil {
		return nil, nil, err
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

	// make an array of phrases with id so we can match all the translates and text-to-speech
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
	// TODO if you don't want these files to persist then you need to defer removing them from calling function
	audioBasePath := s.config.TTSBasePath + title.Name

	fromAudioBasePath := fmt.Sprintf("%s/%d/", audioBasePath, title.FromVoiceId)
	toAudioBasePath := fmt.Sprintf("%s/%d/", audioBasePath, title.ToVoiceId)

	_, err := s.translate.CreateTTS(e, title, title.FromVoiceId, fromAudioBasePath)
	if err != nil {
		e.Logger().Error(err)
		// if error remove all the text-to-speech created up to that point
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}

	toPhrases, err := s.translate.CreateTTS(e, title, title.ToVoiceId, toAudioBasePath)
	if err != nil {
		e.Logger().Error(err)
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}
	title.ToPhrases = toPhrases

	// get pause path string to build the full pause file path
	pausePath, ok := audiofile.AudioPauseFilePath[title.Pause]
	if !ok {
		e.Logger().Error(util.ErrPauseNotFound)
		return nil, util.ErrPauseNotFound
	}
	fullPausePath := s.config.TTSBasePath + pausePath

	// create a temporary directory for building all the files
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

func validateAudioRequest(e echo.Context) (*models.Title, error) {
	languagesCount := models.GetLanguagesLength() - 1
	voicesCount := models.GetVoicesLength() - 1
	// get values from multipart form
	titleName := e.FormValue("title_name")
	// convert strings from multipart form to int
	fileLangId, err := strconv.Atoi(e.FormValue("file_language_id"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting file_language_id to int: %s", err.Error())
	}
	// validate fileLangId
	if fileLangId < 0 || fileLangId > languagesCount {
		e.Logger().Error(util.ErrLanguageIdInvalid)
		return nil, fmt.Errorf("file_language_id must be between 0 and %d", languagesCount)
	}
	toVoiceId, err := strconv.Atoi(e.FormValue("to_voice_id"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting to_voice_id to int: %s", err.Error())
	}
	// validate voiceId
	if toVoiceId < 0 || toVoiceId > voicesCount {
		e.Logger().Error(util.ErrVoiceIdInvalid)
		return nil, fmt.Errorf("to_voice_id must be between 0 and %d", voicesCount)
	}
	fromVoiceId, err := strconv.Atoi(e.FormValue("from_voice_id"))
	if err != nil {
		e.Logger().Error(err)
		return nil, fmt.Errorf("error converting from_voice_id to int: %s", err.Error())
	}
	// validate voiceId
	if fromVoiceId < 0 || fromVoiceId > voicesCount {
		voiceIdError := errors.New(fmt.Sprintf("from_voice_id must be between 0 and %d", voicesCount))
		e.Logger().Error(voiceIdError)
		return nil, voiceIdError
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

	// pattern is the pattern used to build the audio files at /internal/pattern
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
