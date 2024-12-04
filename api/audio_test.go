package api

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"talkliketv.click/tltv/internal/models"
	"testing"

	"talkliketv.click/tltv/internal/util"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"talkliketv.click/tltv/internal/test"
)

func TestAudioFromFile(t *testing.T) {
	if util.Integration {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	// TODO add test for pattern
	title := test.RandomTitle()
	toVoice := test.RandomVoice()
	fromVoice := test.RandomVoice()

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)
	err := os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"
	stringsSlice := []string{"This is the first sentence.", "This is the second sentence."}

	silenceBasePath := test.AudioBasePath + "silence/4SecSilence.mp3"
	fromAudioBasePath := fmt.Sprintf("%s%d/", tmpAudioBasePath, fromVoice.LangId)
	toAudioBasePath := fmt.Sprintf("%s%d/", tmpAudioBasePath, toVoice.LangId)

	okFormMap := map[string]string{
		"fileLanguageId": strconv.Itoa(title.TitleLangId),
		"titleName":      title.Name,
		"fromVoiceId":    strconv.Itoa(rand.IntN(100)),
		"toVoiceId":      strconv.Itoa(rand.IntN(100)),
	}

	testCases := []testCase{
		{
			name: "OK",
			buildStubs: func(stubs MockStubs) {
				file, err := os.Create(filename)
				require.NoError(t, err)
				defer file.Close()
				// GetLines(echo.Context, multipart.File) ([]string, error)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any(), gomock.Any()).
					Return(stringsSlice, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, fromVoice.LangId, fromAudioBasePath).
					Return([]models.Phrase{}, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, toVoice.LangId, toAudioBasePath).
					Return([]models.Phrase{}, nil)
				// BuildAudioInputFiles(echo.Context, []int64, db.Title, string, string, string, string) error
				stubs.AudioFileX.EXPECT().
					BuildAudioInputFiles(gomock.Any(), title, silenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
					Return(nil)
				// CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreateMp3Zip(gomock.Any(), title, gomock.Any()).
					Return(file, nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				return createMultiPartBody(t, data, filename, okFormMap)
			},
		},
		//{
		//	name: "OK with pause",
		//	user: user,
		//	buildStubs: func(stubs MockStubs) {
		//		file, err := os.Create(filename)
		//		require.NoError(t, err)
		//		defer file.Close()
		//		silenceBasePath = test.AudioBasePath + "silence/3SecSilence.mp3"
		//		// GetLines(echo.Context, multipart.File) ([]string, error)
		//		stubs.AudioFileX.EXPECT().
		//			GetLines(gomock.Any(), gomock.Any()).
		//			Return(stringsSlice, nil)
		//		stubs.MockQuerier.EXPECT().
		//			InsertTitle(gomock.Any(), insertTitle).
		//			Times(1).Return(title, nil)
		//		stubs.TranslateX.EXPECT().
		//			InsertNewPhrases(gomock.Any(), title, stubs.MockQuerier, stringsSlice).
		//			Times(1).Return(dbTranslates, nil)
		//		// SelectVoiceById(ctx context.Context, id int16) (Voice, error)
		//		stubs.MockQuerier.EXPECT().
		//			SelectVoiceById(gomock.Any(), fromVoice.ID).
		//			Return(fromVoice, nil)
		//		stubs.MockQuerier.EXPECT().
		//			SelectVoiceById(gomock.Any(), toVoice.ID).
		//			Return(toVoice, nil)
		//		stubs.TranslateX.EXPECT().
		//			CreateTTS(gomock.Any(), stubs.MockQuerier, title, fromVoice.ID, fromAudioBasePath).
		//			Return(nil)
		//		stubs.TranslateX.EXPECT().
		//			CreateTTS(gomock.Any(), stubs.MockQuerier, title, toVoice.ID, toAudioBasePath).
		//			Return(nil)
		//		// SelectPhraseIdsByTitleId(ctx context.Context, titleID int64) ([]int64, error)
		//		stubs.MockQuerier.EXPECT().
		//			SelectPhraseIdsByTitleId(gomock.Any(), title.ID).
		//			Return(phraseIDs, nil)
		//		// BuildAudioInputFiles(echo.Context, []int64, db.Title, string, string, string, string) error
		//		stubs.AudioFileX.EXPECT().
		//			BuildAudioInputFiles(gomock.Any(), phraseIDs, title, silenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
		//			Return(nil)
		//		stubs.MockQuerier.EXPECT().
		//			SelectTranslatesByTitleIdLangId(gomock.Any(), selectTranslatesByTitleIdLangIdParams).
		//			Return(dbTranslates, nil)
		//		// CreateMp3Zip(echo.Context, db.Title, string) (*os.File, error)
		//		stubs.AudioFileX.EXPECT().
		//			CreateMp3Zip(gomock.Any(), dbTranslates, title, gomock.Any()).
		//			Return(file, nil)
		//	},
		//	checkResponse: func(res *http.Response) {
		//		require.Equal(t, http.StatusOK, res.StatusCode)
		//	},
		//	multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
		//		data := []byte("This is the first sentence.\nThis is the second sentence.\n")
		//
		//		formMap := map[string]string{
		//			"fileLanguageId": strconv.Itoa(int(title.OgLanguageID)),
		//			"titleName":      title.Title,
		//			"fromVoiceId":    strconv.Itoa(int(fromVoice.ID)),
		//			"toVoiceId":      strconv.Itoa(int(toVoice.ID)),
		//			"pause":          "3",
		//		}
		//		return createMultiPartBody(t, data, filename, formMap)
		//	},
		//	permissions: []string{db.WriteTitlesCode},
		//},
		//{
		//	name: "Pause out of range",
		//	multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
		//		data := []byte("This is the first sentence.\nThis is the second sentence.\n")
		//
		//		formMap := map[string]string{
		//			"fileLanguageId": strconv.Itoa(int(title.OgLanguageID)),
		//			"titleName":      title.Title,
		//			"fromVoiceId":    strconv.Itoa(int(fromVoice.ID)),
		//			"toVoiceId":      strconv.Itoa(int(toVoice.ID)),
		//			"pause":          "11",
		//		}
		//		return createMultiPartBody(t, data, filename, formMap)
		//	},
		//	user: user,
		//	buildStubs: func(stubs MockStubs) {
		//	},
		//	checkResponse: func(res *http.Response) {
		//		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		//		resBody := readBody(t, res)
		//		require.Contains(t, resBody, "pause must be between 3 and 10: 11")
		//	},
		//	permissions: []string{db.WriteTitlesCode},
		//},
		//{
		//	name: "Bad Request Body",
		//	user: user,
		//	multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
		//		data := []byte("This is the first sentence.\nThis is the second sentence.\n")
		//		formMap := map[string]string{
		//			"fileLanguageId": strconv.Itoa(int(title.OgLanguageID)),
		//			"toVoiceId":      strconv.Itoa(int(toVoice.ID)),
		//			"fromVoiceId":    strconv.Itoa(int(fromVoice.ID)),
		//		}
		//		return createMultiPartBody(t, data, filename, formMap)
		//	},
		//	buildStubs: func(stubs MockStubs) {
		//	},
		//	checkResponse: func(res *http.Response) {
		//		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		//		resBody := readBody(t, res)
		//		require.Contains(t, resBody, "{\"message\":\"request body has an error: doesn't match schema: Error at \\\"/titleName\\\": property \\\"titleName\\\" is missing\"}")
		//	},
		//	permissions: []string{db.WriteTitlesCode},
		//},
		//{
		//	name: "File Too Big",
		//	user: user,
		//	multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
		//		tooBigFile := test.AudioBasePath + "tooBigFile.txt"
		//		file, err := os.Create(tooBigFile)
		//		require.NoError(t, err)
		//		defer file.Close()
		//		writer := bufio.NewWriter(file)
		//		for i := 0; i < 64100; i++ {
		//			// Write random characters to the file
		//			char := byte('a')
		//			err = writer.WriteByte(char)
		//			require.NoError(t, err)
		//		}
		//		writer.Flush()
		//
		//		multiFile, err := os.Open(tooBigFile)
		//		require.NoError(t, err)
		//		body := new(bytes.Buffer)
		//		multiWriter := multipart.NewWriter(body)
		//		part, err := multiWriter.CreateFormFile("filePath", tooBigFile)
		//		require.NoError(t, err)
		//		_, err = io.Copy(part, multiFile)
		//		require.NoError(t, err)
		//		fieldMap := okFormMap
		//		for field, value := range fieldMap {
		//			err = multiWriter.WriteField(field, value)
		//			require.NoError(t, err)
		//		}
		//		require.NoError(t, multiWriter.Close())
		//		return body, multiWriter
		//	},
		//	buildStubs: func(stubs MockStubs) {
		//	},
		//	checkResponse: func(res *http.Response) {
		//		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		//		resBody := readBody(t, res)
		//		require.Contains(t, resBody, "file too large")
		//	},
		//	permissions: []string{db.WriteTitlesCode},
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ts := setupServerTest(t, ctrl, tc)
			multiBody, multiWriter := tc.multipartBody(t)
			req, err := http.NewRequest(http.MethodPost, ts.URL+audioBasePath, multiBody)
			require.NoError(t, err)

			req.Header.Set("Content-Type", multiWriter.FormDataContentType())
			res, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			tc.checkResponse(res)
			require.NoError(t, err)
		})
	}
}

func TestAudioFromFileIntegration(t *testing.T) {
	if !util.Integration {
		t.Skip("skipping integration test")
	}

	t.Parallel()
	e := echo.New()

	tr, af := CreateDependencies(e)
	_ = NewServer(e, testCfg.Config, tr, af)

	title := test.RandomTitle()

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"

	okFormMap := map[string]string{
		"fileLanguageId": strconv.Itoa(rand.IntN(100)),
		"titleName":      title.Name,
		"fromVoiceId":    strconv.Itoa(80),
		"toVoiceId":      strconv.Itoa(182),
	}

	testCases := []testCase{

		{
			name: "OK",
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")

				formMap := okFormMap
				return createMultiPartBody(t, data, filename, formMap)
			},
		},
		//{
		//	name: "Invalid Voice Id",
		//	user: user,
		//	checkResponse: func(res *http.Response) {
		//		require.Equal(t, http.StatusBadRequest, res.StatusCode)
		//		resBody := readBody(t, res)
		//		require.Contains(t, resBody, "voice id invalid")
		//	},
		//	multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
		//		data := []byte("This is the first sentence.\nThis is the second sentence.\n")
		//		badFormMap := map[string]string{
		//			"fileLanguageId": strconv.Itoa(int(title.OgLanguageID)),
		//			"titleName":      title.Title,
		//			"fromVoiceId":    strconv.Itoa(80),
		//			"toVoiceId":      strconv.Itoa(1000),
		//		}
		//		return createMultiPartBody(t, data, filename, badFormMap)
		//	},
		//	permissions: []string{db.WriteTitlesCode},
		//},
		//{
		//	name: "OK with Pause",
		//	user: user,
		//	checkResponse: func(res *http.Response) {
		//		require.Equal(t, http.StatusOK, res.StatusCode)
		//	},
		//	multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
		//		data := []byte("This is the first sentence.\nThis is the second sentence.\n")
		//
		//		pauseFormMap := map[string]string{
		//			"fileLanguageId": strconv.Itoa(int(title.OgLanguageID)),
		//			"titleName":      title.Title,
		//			"fromVoiceId":    strconv.Itoa(80),
		//			"toVoiceId":      strconv.Itoa(182),
		//			"pause":          "6",
		//		}
		//		return createMultiPartBody(t, data, filename, pauseFormMap)
		//	},
		//	permissions: []string{db.WriteTitlesCode},
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ts := httptest.NewServer(e)

			multiBody, multiWriter := tc.multipartBody(t)
			url := ts.URL + audioBasePath + filename
			req, err := http.NewRequest(http.MethodPost, url, multiBody)
			require.NoError(t, err)

			req.Header.Set("Content-Type", multiWriter.FormDataContentType())
			res, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			tc.checkResponse(res)
			require.NoError(t, err)
		})
	}
}
