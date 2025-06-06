package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"talkliketv.com/tltv/internal/interfaces"
	"talkliketv.com/tltv/internal/models"
	"talkliketv.com/tltv/internal/services"
	"talkliketv.com/tltv/internal/services/audiofile"
	"talkliketv.com/tltv/internal/services/tokens"
	"talkliketv.com/tltv/internal/services/translates"
	"talkliketv.com/tltv/internal/testutil"
	"talkliketv.com/tltv/internal/util"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/text/language"
)

func TestAudioFromFile(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	title := testutil.RandomTitle()
	fromVoice := testutil.RandomVoice()
	toVoice := testutil.RandomVoice()
	title.ToVoice = toVoice.Name
	title.FromVoice = fromVoice.Name
	title.TitleLang = "en"

	// create a base path for storing mp3 audio files
	tmpAudioBasePath := testutil.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(tmpAudioBasePath, 0777)
	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)
	require.NoError(t, err)

	audioFromFileName := tmpAudioBasePath + "TestAudioFromFile.txt"
	stringsSlice := []string{"This is the first sentence.", "This is the second sentence."}
	phrase1 := interfaces.Phrase{
		ID:   0,
		Text: "This is the first sentence.",
	}
	phrase2 := interfaces.Phrase{
		ID:   1,
		Text: "This is the second sentence.",
	}
	title.TitlePhrases = []interfaces.Phrase{phrase1, phrase2}

	titleWithTranslates := title
	titleWithTranslates.ToPhrases = []interfaces.Phrase{phrase1, phrase2}

	fiveSecSilenceBasePath := testutil.AudioBasePath + "silence/5SecSilence.mp3"
	fromAudioBasePath := fmt.Sprintf("%s%s/", tmpAudioBasePath, fromVoice.Name)
	toAudioBasePath := fmt.Sprintf("%s%s/", tmpAudioBasePath, toVoice.Name)

	randomToken := testutil.RandomString(32)
	okFormMap := map[string]string{
		"title_name":    title.Name,
		"from_voice_id": title.FromVoice,
		"to_voice_id":   title.ToVoice,
		"pause":         "5",
		"pattern":       "1",
		"token":         randomToken,
	}

	// Define phrases for DetectLanguage
	phraseTexts := []string{phrase1.Text, phrase2.Text}

	testCases := []testCase{
		{
			name: "OK",
			mocks: func(stubs testutil.MockStubs) {
				file, err := os.Create(audioFromFileName)
				require.NoError(t, err)
				defer file.Close()
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.FromVoice).
					Return(fromVoice, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.ToVoice).
					Return(toVoice, nil)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any()).
					Return(stringsSlice, nil)
				// Add this expectation for DetectLanguage
				stubs.TranslateX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Eq(phraseTexts)).
					Return(language.English, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, fromVoice, fromAudioBasePath).
					Return(title.TitlePhrases, nil)
				stubs.TranslateX.EXPECT().
					CreateTTS(gomock.Any(), title, toVoice, toAudioBasePath).
					Return(title.TitlePhrases, nil)
				stubs.AudioFileX.EXPECT().
					BuildAudioInputFiles(titleWithTranslates, fiveSecSilenceBasePath, fromAudioBasePath, toAudioBasePath, gomock.Any()).
					Return(nil)
				stubs.AudioFileX.EXPECT().
					CreateMp3Zip(titleWithTranslates, gomock.Any()).
					Return(file, nil)
				stubs.ModelsX.EXPECT().
					UpdateTokenField(gomock.Any(), true, randomToken, "UploadUsed").
					Return(nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(validSentences)
				return createMultiPartBody(t, data, audioFromFileName, okFormMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
		},
		{
			name: "Pause out of range",
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.ToVoice).
					Return(toVoice, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.FromVoice).
					Return(fromVoice, nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(validSentences)
				formMap := maps.Clone(okFormMap)
				formMap["pause"] = "11"
				return createMultiPartBody(t, data, audioFromFileName, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "pause must be between 3 and 10")
			},
		},
		{
			name: "pattern out of range",
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.ToVoice).
					Return(interfaces.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.FromVoice).
					Return(interfaces.Voice{}, nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(validSentences)
				formMap := maps.Clone(okFormMap)
				formMap["pattern"] = "5"
				return createMultiPartBody(t, data, audioFromFileName, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "pattern must be between 1 and 3")
			},
		},
		{
			name: "Bad Request Body",
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := map[string]string{
					"from_voice_id": title.FromVoice,
					"to_voice_id":   title.ToVoice,
					"pause":         "10",
					"token":         randomToken,
				}
				return createMultiPartBody(t, data, audioFromFileName, formMap)
			},
			mocks: func(stubs testutil.MockStubs) {
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "{\"message\":\"request body has an error: doesn't match schema: Error at \\\"/title_name\\\": property \\\"title_name\\\" is missing\"}")
			},
		},
		{
			name: "File Too Big",
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				tooBigFile := testutil.AudioBasePath + "tooBigFile.txt"
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
				part, err := multiWriter.CreateFormFile("file_path", tooBigFile)
				require.NoError(t, err)
				_, err = io.Copy(part, multiFile)
				require.NoError(t, err)
				//fieldMap := okFormMap
				for field, value := range okFormMap {
					err = multiWriter.WriteField(field, value)
					require.NoError(t, err)
				}
				require.NoError(t, multiWriter.Close())
				return body, multiWriter
			},
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "file too large")
			},
		},
		{
			name: "Too Many Phrases",
			mocks: func(stubs testutil.MockStubs) {
				for i := 0; i < 101; i++ {
					phrase := "This is a sentence that is big enough\n"
					stringsSlice = append(stringsSlice, phrase)
				}

				file, err := os.Create(audioFromFileName)
				require.NoError(t, err)
				defer file.Close()

				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.ToVoice).
					Return(interfaces.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(gomock.Any(), title.FromVoice).
					Return(interfaces.Voice{}, nil)
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any()).
					Return(stringsSlice, nil)
				// CreatePhrasesZip(e echo.Context, chunkedPhrases iter.Seq[[]string], tmpPath string, audioFromFileName string) (*os.File, error)
				stubs.AudioFileX.EXPECT().
					CreatePhrasesZip(gomock.Any(), tmpAudioBasePath, title.Name).
					Return(file, nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(validSentences)
				return createMultiPartBody(t, data, audioFromFileName, okFormMap)
			},
		},
		{
			name: "Used Token",
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(models.ErrUsedToken)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				require.NoError(t, err)
				return createMultiPartBody(t, data, audioFromFileName, okFormMap)
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

func TestAudioFromFile_FileFormatDetection(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	testFileName := testutil.AudioBasePath + "FileFormatDetection.txt"

	title := testutil.RandomTitle()
	randomToken := testutil.RandomString(32)

	formMap := map[string]string{
		"title_name":    title.Name,
		"from_voice_id": title.FromVoice,
		"to_voice_id":   title.ToVoice,
		"token":         randomToken,
		"pause":         "5",
		"pattern":       "1",
	}

	testCases := []testCase{
		{
			name: "Detect Paragraph Format",
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				// Create text that will be detected as paragraph format (longer lines)
				paragraphText := "This is a much longer paragraph that contains multiple sentences. " +
					"It's designed to be detected as a paragraph format rather than one phrase per line. " +
					"The detector should recognize this based on the average line length and structure."
				return createMultiPartBody(t, []byte(paragraphText), testFileName, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "Please parse file before uploading")
			},
		},
		{
			name: "Detect SRT Format",
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				// Create text in SRT format
				srtText := "1\n00:00:01,000 --> 00:00:04,000\nThis is the first subtitle.\n\n" +
					"2\n00:00:05,000 --> 00:00:09,000\nThis is the second subtitle.\n"
				return createMultiPartBody(t, []byte(srtText), testFileName, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "Please parse file before uploading")
			},
		},
		{
			name: "Error Opening File",
			mocks: func(stubs testutil.MockStubs) {
				stubs.ModelsX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				// Create a corrupt multipart form to force file open error
				body := new(bytes.Buffer)
				multiWriter := multipart.NewWriter(body)

				// Create a form file with empty content
				part, err := multiWriter.CreateFormFile("file_path", "corrupt_file.txt")
				require.NoError(t, err)

				// Write a partial/corrupt file content
				_, err = part.Write([]byte{0xFF, 0xD8}) // Incomplete content
				require.NoError(t, err)

				// Add form fields
				if err = multiWriter.WriteField("token", randomToken); err != nil {
					require.NoError(t, err)
				}
				if err = multiWriter.WriteField("title_name", title.Name); err != nil {
					require.NoError(t, err)
				}
				if err = multiWriter.WriteField("from_voice_id", title.FromVoice); err != nil {
					require.NoError(t, err)
				}
				if err = multiWriter.WriteField("to_voice_id", title.ToVoice); err != nil {
					require.NoError(t, err)
				}
				if err = multiWriter.WriteField("pause", "5"); err != nil {
					require.NoError(t, err)
				}
				if err = multiWriter.WriteField("pattern", "1"); err != nil {
					require.NoError(t, err)
				}

				// Close the writer - this will actually make the form valid,
				// so this test will need special handling in the ServerMock
				multiWriter.Close()
				return body, multiWriter
			},
			checkResponse: func(res *http.Response) {
				// Error could be either in opening or format detection depending on the mock
				require.True(t, res.StatusCode == http.StatusBadRequest)
			},
		},
		{
			name: "Missing Form File",
			mocks: func(stubs testutil.MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				// Create a multipart form without the file_path field
				body := new(bytes.Buffer)
				multiWriter := multipart.NewWriter(body)

				// Add other form fields but no file
				if err := multiWriter.WriteField("token", randomToken); err != nil {
					require.NoError(t, err)
				}
				if err := multiWriter.WriteField("title_name", title.Name); err != nil {
					require.NoError(t, err)
				}
				if err := multiWriter.WriteField("from_voice_id", title.FromVoice); err != nil {
					require.NoError(t, err)
				}
				if err := multiWriter.WriteField("to_voice_id", title.ToVoice); err != nil {
					require.NoError(t, err)
				}
				if err := multiWriter.WriteField("pause", "5"); err != nil {
					require.NoError(t, err)
				}
				if err := multiWriter.WriteField("pattern", "1"); err != nil {
					require.NoError(t, err)
				}

				multiWriter.Close()
				return body, multiWriter
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "request body has an error: doesn't match schema: Error at \\\"/file_path\\\":")
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
		})
	}
}

func TestParseFile(t *testing.T) {
	if util.Test != "unit" && !testing.Short() {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	parseFileName := testutil.ParseBasePath + "TestParseFile.txt"
	err := os.MkdirAll(testutil.ParseBasePath, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(testutil.ParseBasePath)

	testCases := []testCase{
		{
			name: "OK",
			mocks: func(stubs testutil.MockStubs) {
				file, err := os.Create(parseFileName)
				require.NoError(t, err)
				defer file.Close()
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any()).
					Return([]string{"This is the first sentence.", "This is the second sentence."}, nil)
				stubs.AudioFileX.EXPECT().
					CreatePhrasesZip(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(file, nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(testutil.FiveSentences)
				return createMultiPartBody(t, data, parseFileName, nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
				require.Equal(t, "attachment; filename=\"TestParseFile.txt_parsed.zip\"",
					res.Header.Get("Content-Disposition"))
			},
		},
		{
			name: "File Too Large",
			mocks: func(stubs testutil.MockStubs) {
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any()).
					Return(nil, services.NewFileTooLargeError(65000, 64000))
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is a test file that is too large.\n")
				return createMultiPartBody(t, data, parseFileName, nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusInternalServerError, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "file too large")
			},
		},
		{
			name: "Error Getting Form File",
			mocks: func(stubs testutil.MockStubs) {
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				// Create a multipart form without the file_path field
				body := new(bytes.Buffer)
				multiWriter := multipart.NewWriter(body)
				err := multiWriter.WriteField("other_field", "value")
				require.NoError(t, err)
				require.NoError(t, multiWriter.Close())
				return body, multiWriter
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "request body has an error: failed to decode request body: part other_field: undefined")
			},
		},
		{
			name: "Error Zipping File",
			mocks: func(stubs testutil.MockStubs) {
				stubs.AudioFileX.EXPECT().
					GetLines(gomock.Any()).
					Return([]string{"This is a test sentence."}, nil)
				stubs.AudioFileX.EXPECT().
					CreatePhrasesZip(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("error creating zip file"))
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is a test sentence.\n")
				return createMultiPartBody(t, data, parseFileName, nil)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusInternalServerError, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "error zipping file")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ts := setupServerTest(ctrl, tc)
			multiBody, multiWriter := tc.multipartBody(t)
			req, err := http.NewRequest(http.MethodPost, ts.URL+parseBasePath, multiBody)
			require.NoError(t, err)

			req.Header.Set("Content-Type", multiWriter.FormDataContentType())
			res, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			tc.checkResponse(res)
		})
	}
}

// TestGoogleIntegration tests the audio from file endpoint with the google tts client
// Program arguments: -test=integration -project-id=token-tltv-test
func TestGoogleIntegration(t *testing.T) {
	if util.Test != "integration" && !testing.Short() {
		t.Skip("skipping integration test")
	}

	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})

	// Use the application default credentials
	ctx := context.Background()
	fClient, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer fClient.Close()

	// Initialize Firestore interfaces
	mods, err := models.NewModels(fClient, "test", models.LangCollString, models.LangCodeCollString, models.VoiceCollString, models.TokenCollString)
	require.NoError(t, err)

	// generate new token and add it to the collection
	plaintext, hash := addTokenFirestore(ctx, t, mods)

	// defer deleting the collection
	defer func(ctx context.Context, hash string) {
		err = mods.DeleteToken(ctx, hash)
		require.NoError(t, err)
	}(ctx, hash)

	// Create Google TTS client using the interface
	googleTTSClient, err := translates.NewGoogleTTSClient(ctx)
	require.NoError(t, err)

	// Create the translate service with the client interface
	tr := translates.New(googleTTSClient, mods)

	srv := NewServer(testCfg.Config, tr, af, mods)
	e := srv.NewEcho(nil)
	title := testutil.RandomTitle()

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := testutil.AudioBasePath + title.Name + "/"
	err = os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"

	okFormMap := map[string]string{
		"title_name":    title.Name,
		"from_voice_id": "af-ZA-Standard-A",
		"to_voice_id":   "am-ET-Standard-A",
		"token":         plaintext,
		"pause":         "4",
		"pattern":       "1",
	}

	testCases := []testCase{
		{
			name: "OK",
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(validSentences)
				return createMultiPartBody(t, data, filename, okFormMap)
			},
		},
		{
			name: "Used Token",
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusForbidden, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "token already used")
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte(validSentences)
				// generate new token
				token2, plaintext2, err := tokens.GenerateToken()
				require.NoError(t, err)
				token2.UploadUsed = true
				err = mods.AddToken(ctx, *token2)
				require.NoError(t, err)
				okFormMap2 := map[string]string{
					"title_name":    title.Name,
					"from_voice_id": title.FromVoice,
					"to_voice_id":   title.ToVoice,
					"token":         plaintext2,
					"pause":         "4",
					"pattern":       "1",
				}
				return createMultiPartBody(t, data, filename, okFormMap2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
