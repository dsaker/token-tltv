package audiofile

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"os"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/services"
	"talkliketv.click/tltv/internal/test"
)

// AudioFromTitle is a helper function that performs the tasks shared by
// AudioFromFile and AudioFromTitle
func AudioFromTitle(e echo.Context, t services.TranslateX, af AudioFileX, title models.Title, path string) (*os.File, error) {
	// TODO if you don't want these files to persist then you need to defer removing them from calling function
	audioBasePath := path + title.Name

	fromAudioBasePath := fmt.Sprintf("%s/%d/", audioBasePath, title.FromVoiceId)
	toAudioBasePath := fmt.Sprintf("%s/%d/", audioBasePath, title.ToVoiceId)

	_, err := t.CreateTTS(e, title, title.FromVoiceId, fromAudioBasePath)
	if err != nil {
		e.Logger().Error(err)
		// if error remove all the text-to-speech created up to that point
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			e.Logger().Error(osErr)
		}
		return nil, err
	}

	toPhrases, err := t.CreateTTS(e, title, title.ToVoiceId, toAudioBasePath)
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
	pausePath, ok := AudioPauseFilePath[title.Pause]
	if !ok {
		e.Logger().Error(models.ErrPauseNotFound)
		return nil, models.ErrPauseNotFound
	}
	fullPausePath := path + pausePath

	// create a temporary directory for building all the files
	tmpDirPath := fmt.Sprintf("%s%s-%s/", path, title.Name, test.RandomString(4))
	err = os.MkdirAll(tmpDirPath, 0777)
	if err != nil {
		e.Logger().Error(err)
		return nil, err
	}

	if err = af.BuildAudioInputFiles(e, title, fullPausePath, fromAudioBasePath, toAudioBasePath, tmpDirPath); err != nil {
		return nil, err
	}

	return af.CreateMp3Zip(e, title, tmpDirPath)
}
