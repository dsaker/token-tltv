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
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/translates"
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

	title := test.RandomTitle()

	// create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(tmpAudioBasePath, 0777)
	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)
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
	title.TitlePhrases = []models.Phrase{phrase1, phrase2}

	titleWithTranslates := title
	titleWithTranslates.ToPhrases = []models.Phrase{phrase1, phrase2}

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
			buildStubs: func(stubs test.MockStubs) {
				file, err := os.Create(filename)
				require.NoError(t, err)
				defer file.Close()
				// GetLines(echo.Context, multipart.File) ([]string, error)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any(), gomock.Any()).
					Return(stringsSlice, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.FromVoiceId, fromAudioBasePath).
					Return(title.TitlePhrases, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.ToVoiceId, toAudioBasePath).
					Return(title.TitlePhrases, nil)
				// BuildAudioInputFiles(echo.Context, []int64, db.Title, string, string, string, string) error
				stubs.AudioFileX.EXPECT().
					BuildAudioInputFiles(gomock.Any(), titleWithTranslates, fiveSecSilenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
					Return(nil)
				// CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreateMp3Zip(gomock.Any(), titleWithTranslates, gomock.Any()).
					Return(file, nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				okFormMap["token"] = token
				return createMultiPartBody(t, data, filename, okFormMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
		},
		{
			name: "OK with Pause",
			buildStubs: func(stubs test.MockStubs) {
				file, err := os.Create(filename)
				require.NoError(t, err)
				defer file.Close()
				// change pause
				title.Pause = 3
				titleWithTranslates.Pause = 3
				// GetLines(echo.Context, multipart.File) ([]string, error)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any(), gomock.Any()).
					Return(stringsSlice, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.FromVoiceId, fromAudioBasePath).
					Return(title.TitlePhrases, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, title.ToVoiceId, toAudioBasePath).
					Return(title.TitlePhrases, nil)
				// BuildAudioInputFiles(echo.Context, []int64, db.Title, string, string, string, string) error
				stubs.AudioFileX.EXPECT().
					BuildAudioInputFiles(gomock.Any(), titleWithTranslates, threeSecSilenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
					Return(nil)
				// CreateMp3Zip(e echo.Context, t models.Title, tmpDir string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreateMp3Zip(gomock.Any(), titleWithTranslates, gomock.Any()).
					Return(file, nil)
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
		},
		{
			name: "Pause out of range",
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "pause must be between 3 and 10: 11")
			},
		},
		{
			name: "File LanguageId out of range",
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "toVoiceId must be between 0 and "+strconv.Itoa(models.GetVoicesLength()-1))
			},
		},
		{
			name: "toVoiceId string",
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)

				require.Contains(t, resBody, "fromVoiceId must be between 0 and "+strconv.Itoa(models.GetVoicesLength()-1))
			},
		},
		{
			name: "fromVoiceId string",
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
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
					"pause":          "10",
				}
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
				return createMultiPartBody(t, data, filename, formMap)
			},
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				okFormMap["token"] = token
				fieldMap := okFormMap
				for field, value := range fieldMap {
					err = multiWriter.WriteField(field, value)
					require.NoError(t, err)
				}
				require.NoError(t, multiWriter.Close())
				return body, multiWriter
			},
			buildStubs: func(stubs test.MockStubs) {
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "file too large")
			},
		},
		{
			name: "Too Many Phrases",
			buildStubs: func(stubs test.MockStubs) {
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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				okFormMap["token"] = token
				return createMultiPartBody(t, data, filename, okFormMap)
			},
		},
		{
			name: "Used Token",
			buildStubs: func(stubs test.MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				okFormMap["token"] = token
				err = models.SetTokenStatus(token, models.Used)
				require.NoError(t, err)
				return createMultiPartBody(t, data, filename, okFormMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusForbidden, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "token already used")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ts := setupServerTest(ctrl, tc)
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

func TestGoogleIntegration(t *testing.T) {
	if !util.Integration {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})
	// create translates with google or amazon clients depending on the flag set in conifg
	// I also set a global platform since this will not be changed during execution
	tr := translates.New(*translates.NewGoogleClients(), translates.AmazonClients{}, &models.Models{})
	if translates.GlobalPlatform == translates.Amazon {
		tr = translates.New(translates.GoogleClients{}, *translates.NewAmazonClients(), &models.Models{})
	}

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
				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				formMap["token"] = token
				return createMultiPartBody(t, data, filename, formMap)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//ctrl := gomock.NewController(t)
			//defer ctrl.Finish()

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

func TestAmazonIntegration(t *testing.T) {
	if !util.Integration {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	translates.GlobalPlatform = translates.Amazon
	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})
	// create translates with google or amazon clients depending on the flag set in conifg
	// I also set a global platform since this will not be changed during execution
	tr := translates.New(*translates.NewGoogleClients(), translates.AmazonClients{}, &models.Models{})
	if translates.GlobalPlatform == translates.Amazon {
		tr = translates.New(translates.GoogleClients{}, *translates.NewAmazonClients(), &models.Models{})
	}

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
		"fileLanguageId": strconv.Itoa(rand.IntN(test.MaxLanguages)), //nolint:gosec
		"titleName":      title.Name,
		"fromVoiceId":    strconv.Itoa(rand.IntN(test.MaxVoices)), //nolint:gosec
		"toVoiceId":      strconv.Itoa(rand.IntN(test.MaxVoices)), //nolint:gosec
	}

	testCases := []testCase{

		{
			name: "OK",
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")

				mu.Lock()
				token := tokenStrings[tokenCount]
				tokenCount++
				mu.Unlock()
				okFormMap["token"] = token
				return createMultiPartBody(t, data, filename, okFormMap)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//ctrl := gomock.NewController(t)
			//defer ctrl.Finish()

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
