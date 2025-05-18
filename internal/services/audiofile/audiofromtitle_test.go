package audiofile

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"os"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"
)

func TestAudioFromTitle(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	// Setup common test data
	fromVoice := testutil.RandomVoice()
	toVoice := testutil.RandomVoice()
	title := testutil.RandomTitle()

	// Define AudioPauseFilePath for tests
	audioPauseFilePath := AudioPauseFilePath[title.Pause]

	// Create a temporary directory for tests
	tempDir, err := os.MkdirTemp("/tmp/", "audio-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Ensure tempDir ends with a slash
	if !os.IsPathSeparator(tempDir[len(tempDir)-1]) {
		tempDir = tempDir + string(os.PathSeparator)
	}

	// Create a temporary zip file to return from CreateMp3Zip
	zipFile, err := os.CreateTemp("/tmp", "test-zip-*.zip")
	require.NoError(t, err)
	defer os.Remove(zipFile.Name())

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Create mocks
		mocks := testutil.NewMockStubs(ctrl)
		// Set expected paths
		audioBasePath := tempDir + title.Name
		fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)
		toAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, toVoice.Name)
		pausePath := tempDir + audioPauseFilePath

		// Create result phrases for second TTS call
		toPhrases := []interfaces.Phrase{
			{ID: 1, Text: "Test phrase 1"},
			{ID: 2, Text: "Test phrase 2"},
		}

		titleWithPhrases := title
		titleWithPhrases.ToPhrases = toPhrases

		// Setup mock expectations
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, fromVoice, fromAudioBasePath).Return(nil, nil)
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, toVoice, toAudioBasePath).Return(toPhrases, nil)
		mocks.AudioFileX.EXPECT().BuildAudioInputFiles(titleWithPhrases, pausePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).Return(nil)
		mocks.AudioFileX.EXPECT().CreateMp3Zip(titleWithPhrases, gomock.Any()).Return(zipFile, nil)

		ctx := context.Background()
		// Call the function under test
		result, err := AudioFromTitle(ctx, mocks.TranslateX, mocks.AudioFileX, fromVoice, toVoice, title, tempDir)

		// Assert expectations
		require.NoError(t, err)
		assert.Equal(t, zipFile, result)
	})

	t.Run("First CreateTTS fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Create mocks
		mocks := testutil.NewMockStubs(ctrl)

		// Set expected paths
		audioBasePath := tempDir + title.Name
		fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)

		// Setup mock expectations
		expectedErr := errors.New("tts creation failed")
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, fromVoice, fromAudioBasePath).Return(nil, expectedErr)

		ctx := context.Background()
		// Call the function under test
		result, err := AudioFromTitle(ctx, mocks.TranslateX, mocks.AudioFileX, fromVoice, toVoice, title, tempDir)

		// Assert expectations
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("Second CreateTTS fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Create mocks
		mocks := testutil.NewMockStubs(ctrl)

		// Set expected paths
		audioBasePath := tempDir + title.Name
		fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)
		toAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, toVoice.Name)

		// Setup mock expectations
		expectedErr := errors.New("second tts creation failed")
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, fromVoice, fromAudioBasePath).Return(nil, nil)
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, toVoice, toAudioBasePath).Return(nil, expectedErr)

		ctx := context.Background()
		// Call the function under test
		result, err := AudioFromTitle(ctx, mocks.TranslateX, mocks.AudioFileX, fromVoice, toVoice, title, tempDir)

		// Assert expectations
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("Invalid pause", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Create mocks
		mocks := testutil.NewMockStubs(ctrl)

		// Set expected paths
		audioBasePath := tempDir + title.Name
		fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)
		toAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, toVoice.Name)

		// Create title with invalid pause
		invalidTitle := title
		invalidTitle.Pause = 999

		// Setup mock expectations
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), invalidTitle, fromVoice, fromAudioBasePath).Return(nil, nil)
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), invalidTitle, toVoice, toAudioBasePath).Return(nil, nil)

		// Call the function under test
		result, err := AudioFromTitle(context.Background(), mocks.TranslateX, mocks.AudioFileX, fromVoice, toVoice, invalidTitle, tempDir)

		// Assert expectations
		assert.Error(t, err)
		assert.Equal(t, interfaces.ErrPauseNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("BuildAudioInputFiles fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Create mocks
		mocks := testutil.NewMockStubs(ctrl)

		// Set expected paths
		audioBasePath := tempDir + title.Name
		fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)
		toAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, toVoice.Name)
		pausePath := tempDir + audioPauseFilePath

		// Create result phrases for second TTS call
		toPhrases := []interfaces.Phrase{
			{ID: 1, Text: "Test phrase 1"},
			{ID: 2, Text: "Test phrase 2"},
		}

		titleWithPhrases := title
		titleWithPhrases.ToPhrases = toPhrases

		// Setup mock expectations
		expectedErr := errors.New("build audio files failed")
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, fromVoice, fromAudioBasePath).Return(nil, nil)
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, toVoice, toAudioBasePath).Return(toPhrases, nil)
		mocks.AudioFileX.EXPECT().BuildAudioInputFiles(titleWithPhrases, pausePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).Return(expectedErr)

		// Call the function under test
		result, err := AudioFromTitle(context.Background(), mocks.TranslateX, mocks.AudioFileX, fromVoice, toVoice, title, tempDir)

		// Assert expectations
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})

	t.Run("CreateMp3Zip fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		// Create mocks
		mocks := testutil.NewMockStubs(ctrl)

		// Set expected paths
		audioBasePath := tempDir + title.Name
		fromAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, fromVoice.Name)
		toAudioBasePath := fmt.Sprintf("%s/%s/", audioBasePath, toVoice.Name)
		pausePath := tempDir + audioPauseFilePath

		// Create result phrases for second TTS call
		toPhrases := []interfaces.Phrase{
			{ID: 1, Text: "Test phrase 1"},
			{ID: 2, Text: "Test phrase 2"},
		}

		titleWithPhrases := title
		titleWithPhrases.ToPhrases = toPhrases

		// Setup mock expectations
		expectedErr := errors.New("create mp3 zip failed")
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, fromVoice, fromAudioBasePath).Return(nil, nil)
		mocks.TranslateX.EXPECT().CreateTTS(gomock.Any(), title, toVoice, toAudioBasePath).Return(toPhrases, nil)
		mocks.AudioFileX.EXPECT().BuildAudioInputFiles(titleWithPhrases, pausePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).Return(nil)
		mocks.AudioFileX.EXPECT().CreateMp3Zip(titleWithPhrases, gomock.Any()).Return(nil, expectedErr)

		// Call the function under test
		result, err := AudioFromTitle(context.Background(), mocks.TranslateX, mocks.AudioFileX, fromVoice, toVoice, title, tempDir)

		// Assert expectations
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}
