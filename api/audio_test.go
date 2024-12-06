package api

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"talkliketv.click/tltv/internal/models"
	"testing"

	"talkliketv.click/tltv/internal/util"

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

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)
	err := os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"
	stringsSlice := []string{"This is the first sentence.", "This is the second sentence."}
	phrase1 := models.Phrase{
		ID:   0,
		Text: "This is the first sentence.",
	}
	phrase2 := models.Phrase{
		ID:   1,
		Text: "This is the second sentence.",
	}
	title.Phrases = []models.Phrase{phrase1, phrase2}

	titleWithTranslates := title
	titleWithTranslates.Translates = []models.Phrase{phrase1, phrase2}

	fiveSecSilenceBasePath := test.AudioBasePath + "silence/5SecSilence.mp3"
	threeSecSilenceBasePath := test.AudioBasePath + "silence/3SecSilence.mp3"
	fromAudioBasePath := fmt.Sprintf("%s%d/", tmpAudioBasePath, title.FromVoiceId)
	toAudioBasePath := fmt.Sprintf("%s%d/", tmpAudioBasePath, title.ToVoiceId)

	okFormMap := map[string]string{
		"fileLanguageId": strconv.Itoa(title.TitleLangId),
		"titleName":      title.Name,
		"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
		"toVoiceId":      strconv.Itoa(title.ToVoiceId),
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
					CreateTTS(gomock.Any(), title, title.FromVoiceId, fromAudioBasePath).
					Return(title.Phrases, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.ToVoiceId, toAudioBasePath).
					Return(title.Phrases, nil)
				// BuildAudioInputFiles(echo.Context, []int64, db.Title, string, string, string, string) error
				stubs.AudioFileX.EXPECT().
					BuildAudioInputFiles(gomock.Any(), titleWithTranslates, fiveSecSilenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
					Return(nil)
				// CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreateMp3Zip(gomock.Any(), titleWithTranslates, gomock.Any()).
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
		{
			name: "OK with Pause",
			buildStubs: func(stubs MockStubs) {
				file, err := os.Create(filename)
				require.NoError(t, err)
				defer file.Close()
				title.Pause = 3
				titleWithTranslates.Pause = 3
				// GetLines(echo.Context, multipart.File) ([]string, error)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any(), gomock.Any()).
					Return(stringsSlice, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.FromVoiceId, fromAudioBasePath).
					Return(title.Phrases, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.ToVoiceId, toAudioBasePath).
					Return(title.Phrases, nil)
				// BuildAudioInputFiles(echo.Context, []int64, db.Title, string, string, string, string) error
				stubs.AudioFileX.EXPECT().
					BuildAudioInputFiles(gomock.Any(), titleWithTranslates, threeSecSilenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
					Return(nil)
				// CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreateMp3Zip(gomock.Any(), titleWithTranslates, gomock.Any()).
					Return(file, nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")

				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.ToVoiceId),
					"pause":          "3",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
		},
		{
			name: "Pause out of range",
			buildStubs: func(stubs MockStubs) {
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "pause must be between 3 and 10: 11")
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.ToVoiceId),
					"pause":          "11",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
		},
		{
			name: "File LanguageId out of range",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": "1000",
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.ToVoiceId),
					"pause":          "10",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "fileLangId must be between 0 and")
			},
		},
		{
			name: "File LanguageId string",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": "abcdefg",
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.ToVoiceId),
					"pause":          "10",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "error converting fileLanguageId to int: strconv.Atoi: parsing \"abcdefg\": invalid syntax")
			},
		},
		{
			name: "toVoiceId out of range",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      "9999",
					"pause":          "10",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "toVoiceId must be between 0 and 192")
			},
		},
		{
			name: "toVoiceId string",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      "a",
					"pause":          "10",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "error converting toVoiceId to int: strconv.Atoi: parsing \"a\": invalid syntax")
			},
		},
		{
			name: "pause string",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.ToVoiceId),
					"pause":          "a",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "error converting pause to int: strconv.Atoi: parsing \"a\": invalid syntax")
			},
		},
		{
			name: "fromVoiceId out of range",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    "9999",
					"toVoiceId":      strconv.Itoa(title.FromVoiceId),
					"pause":          "10",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "fromVoiceId must be between 0 and 192")
			},
		},
		{
			name: "fromVoiceId string",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    "a",
					"toVoiceId":      strconv.Itoa(title.FromVoiceId),
					"pause":          "10",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "error converting fromVoiceId to int: strconv.Atoi: parsing \"a\": invalid syntax")
			},
		},
		{
			name: "pattern out of range",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.FromVoiceId),
					"pause":          "10",
					"pattern":        "5",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "pattern must be between 1 and 4: 5")
			},
		},
		{
			name: "pattern string",
			buildStubs: func(stubs MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"titleName":      title.Name,
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.FromVoiceId),
					"pause":          "10",
					"pattern":        "a",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "error converting pattern to int: strconv.Atoi: parsing \"a\": invalid syntax")
			},
		},
		{
			name: "Bad Request Body",
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"fileLanguageId": strconv.Itoa(title.TitleLangId),
					"fromVoiceId":    strconv.Itoa(title.FromVoiceId),
					"toVoiceId":      strconv.Itoa(title.ToVoiceId),
					"pause":          "11",
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			buildStubs: func(stubs MockStubs) {
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "{\"message\":\"request body has an error: doesn't match schema: Error at \\\"/titleName\\\": property \\\"titleName\\\" is missing\"}")
			},
		},
		{
			name: "File Too Big",
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				tooBigFile := test.AudioBasePath + "tooBigFile.txt"
				file, err := os.Create(tooBigFile)
				require.NoError(t, err)
				defer file.Close()
				writer := bufio.NewWriter(file)
				for i := 0; i < 64100; i++ {
					// Write random characters to the file
					char := byte('a')
					err = writer.WriteByte(char)
					require.NoError(t, err)
				}
				writer.Flush()

				multiFile, err := os.Open(tooBigFile)
				require.NoError(t, err)
				body := new(bytes.Buffer)
				multiWriter := multipart.NewWriter(body)
				part, err := multiWriter.CreateFormFile("filePath", tooBigFile)
				require.NoError(t, err)
				_, err = io.Copy(part, multiFile)
				require.NoError(t, err)
				fieldMap := okFormMap
				for field, value := range fieldMap {
					err = multiWriter.WriteField(field, value)
					require.NoError(t, err)
				}
				require.NoError(t, multiWriter.Close())
				return body, multiWriter
			},
			buildStubs: func(stubs MockStubs) {
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "file too large")
			},
		},
		{
			name: "Too Many Phrases",
			buildStubs: func(stubs MockStubs) {
				for i := 0; i < 101; i++ {
					phrase := test.RandomString(4) + " " + test.RandomString(4)
					stringsSlice = append(stringsSlice, phrase)
				}

				file, err := os.Create(filename)
				require.NoError(t, err)
				defer file.Close()

				// GetLines(echo.Context, multipart.File) ([]string, error)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any(), gomock.Any()).
					Return(stringsSlice, nil)
				// CreatePhrasesZip(e echo.Context, chunkedPhrases iter.Seq[[]string], tmpPath string, filename string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreatePhrasesZip(gomock.Any(), gomock.Any(), tmpAudioBasePath, title.Name).
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

	tr, af := CreateDependencies()
	e := NewServer(testCfg.Config, tr, af)

	title := test.RandomTitle()

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"

	okFormMap := map[string]string{
		"fileLanguageId": strconv.Itoa(rand.IntN(100)), //nolint:gosec
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ts := httptest.NewServer(e)

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
