package audiofile

import (
	"context"
	"fmt"
	"log"
	"os"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/services/translates"
	"talkliketv.com/tltv/internal/testutil"
)

// AudioFromTitle is a helper function that performs the tasks shared by
// AudioFromFile and AudioFromTitle
func AudioFromTitle(c context.Context, t translates.TranslateX, af AudioFileX, fromVoice interfaces.Voice, toVoice interfaces.Voice, title interfaces.Title, path string) (*os.File, error) {
	// TODO if you don't want these files to persist then you need to defer removing them from calling function
	audioBasePath := path + title.Name

	fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)
	toAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, toVoice.Name)

	_, err := t.CreateTTS(c, title, fromVoice, fromAudioBasePath)
	if err != nil {
		// if error remove all the text-to-speech created up to that point
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			log.Printf("error removing audioBasePath: %v", osErr)
		}
		return nil, err
	}

	toPhrases, err := t.CreateTTS(c, title, toVoice, toAudioBasePath)
	if err != nil {
		osErr := os.RemoveAll(audioBasePath)
		if osErr != nil {
			log.Printf("error removing audioBasePath: %v", osErr)
		}
		return nil, err
	}
	title.ToPhrases = toPhrases

	// get pause path string to build the full pause file path
	pausePath, ok := AudioPauseFilePath[title.Pause]
	if !ok {
		return nil, interfaces.ErrPauseNotFound
	}
	fullPausePath := path + pausePath

	// create a temporary directory for building all the files
	tmpDirPath := fmt.Sprintf("%s%s-%s/", path, title.Name, testutil.RandomString(4))
	err = os.MkdirAll(tmpDirPath, 0777)
	if err != nil {
		return nil, err
	}

	if err = af.BuildAudioInputFiles(title, fullPausePath, fromAudioBasePath, toAudioBasePath, tmpDirPath); err != nil {
		return nil, err
	}

	// TODO save audioBasePath to storage bucket
	return af.CreateMp3Zip(title, tmpDirPath)
}
