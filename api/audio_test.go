package api

import (
	"bufio"
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"io"
	"maps"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"talkliketv.click/tltv/internal/audio/audiofile"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/translates"
	"talkliketv.click/tltv/internal/util"
	"testing"
)

func TestAudioFromFile(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	title := test.RandomTitle(voicesMap)

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
	fromAudioBasePath := fmt.Sprintf("%s%d/", tmpAudioBasePath, title.FromVoiceId)
	toAudioBasePath := fmt.Sprintf("%s%d/", tmpAudioBasePath, title.ToVoiceId)

	randomToken := test.RandomString(32)
	okFormMap := map[string]string{
		"file_language_id": strconv.Itoa(title.TitleLangId),
		"title_name":       title.Name,
		"from_voice_id":    strconv.Itoa(title.FromVoiceId),
		"to_voice_id":      strconv.Itoa(title.ToVoiceId),
		"pause":            "5",
		"pattern":          "1",
		"token":            randomToken,
	}

	testCases := []testCase{
		{
			name: "OK",
			buildStubs: func(stubs test.MockStubs) {
				file, err := os.Create(filename)
				require.NoError(t, err)
				defer file.Close()
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(title.TitleLangId).
					Return(models.Language{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.ToVoiceId).
					Return(models.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.FromVoiceId).
					Return(models.Voice{}, nil)
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
				stubs.TokensX.EXPECT().
					UpdateField(gomock.Any(), true, randomToken, "UploadUsed").
					Return(nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				return createMultiPartBody(t, data, filename, okFormMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
		},
		{
			name: "Pause out of range",
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(title.TitleLangId).
					Return(models.Language{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.ToVoiceId).
					Return(models.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.FromVoiceId).
					Return(models.Voice{}, nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := maps.Clone(okFormMap)
				formMap["pause"] = "11"
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "pause must be between 3 and 10")
			},
		},
		{
			name: "file_language_id out of range",
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(9999).
					Return(models.Language{}, models.ErrLanguageIdInvalid)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := maps.Clone(okFormMap)
				formMap["file_language_id"] = "9999"
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "language id invalid")
			},
		},
		{
			name: "file_langauge_id string",
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := maps.Clone(okFormMap)
				formMap["file_language_id"] = "abcd"
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, ": invalid syntax")
			},
		},
		{
			name: "to_voice_id out of range",
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(title.TitleLangId).
					Return(models.Language{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(9999).
					Return(models.Voice{}, models.ErrVoiceIdInvalid)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := maps.Clone(okFormMap)
				formMap["to_voice_id"] = "9999"
				return createMultiPartBody(t, data, filename, formMap)
			},
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusBadRequest, res.StatusCode)
				resBody := readBody(t, res)
				require.Contains(t, resBody, "voice id invalid")
			},
		},
		{
			name: "pattern out of range",
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(title.TitleLangId).
					Return(models.Language{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.ToVoiceId).
					Return(models.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.FromVoiceId).
					Return(models.Voice{}, nil)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				formMap := maps.Clone(okFormMap)
				formMap["pattern"] = "5"
				return createMultiPartBody(t, data, filename, formMap)
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
					"file_language_id": strconv.Itoa(title.TitleLangId),
					"from_voice_id":    strconv.Itoa(title.FromVoiceId),
					"to_voice_id":      strconv.Itoa(title.ToVoiceId),
					"pause":            "10",
					"token":            randomToken,
				}
				return createMultiPartBody(t, data, filename, formMap)
			},
			buildStubs: func(stubs test.MockStubs) {
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
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(title.TitleLangId).
					Return(models.Language{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.ToVoiceId).
					Return(models.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.FromVoiceId).
					Return(models.Voice{}, nil)
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

				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(nil)
				stubs.ModelsX.EXPECT().
					GetLanguage(title.TitleLangId).
					Return(models.Language{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.ToVoiceId).
					Return(models.Voice{}, nil)
				stubs.ModelsX.EXPECT().
					GetVoice(title.FromVoiceId).
					Return(models.Voice{}, nil)
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
		{
			name: "Used Token",
			buildStubs: func(stubs test.MockStubs) {
				stubs.TokensX.EXPECT().
					CheckToken(gomock.Any(), randomToken).
					Return(models.ErrUsedToken)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
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

// TestGoogleIntegration tests the audio from file endpoint with the google tts client
// Program arguments: -test=integration -project-id=token-tltv-test
func TestGoogleIntegration(t *testing.T) {
	if util.Test != "integration" {
		t.Skip("skipping integration test")
	}

	mods := models.Models{Languages: langsMap, Voices: voicesMap}
	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})
	// create translates with google or amazon clients depending on the flag set in conifg
	tr := translates.New(*translates.NewGoogleClients(), translates.AmazonClients{}, &mods, translates.Google)

	// Use the application default credentials
	ctx := context.Background()
	client, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer client.Close()

	// generate new token and add it to the collection
	plaintext, tokens := addTokenFirestore(t, client, ctx)

	// defer deleting the collection
	defer func(ctx context.Context, client *firestore.Client, coll *firestore.CollectionRef) {
		err = util.DeleteFirestoreCollection(ctx, client, coll)
		require.NoError(t, err)
	}(ctx, client, tokens.Coll)

	srv := NewServer(testCfg.Config, tr, af, &tokens, &mods)
	e := srv.NewEcho(nil)
	title := test.RandomTitle(voicesMap)

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	err = os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"

	okFormMap := map[string]string{
		"file_language_id": strconv.Itoa(title.TitleLangId),
		"title_name":       title.Name,
		"from_voice_id":    strconv.Itoa(title.FromVoiceId),
		"to_voice_id":      strconv.Itoa(title.ToVoiceId),
		"token":            *plaintext,
		"pause":            "4",
		"pattern":          "1",
	}

	testCases := []testCase{

		{
			name: "OK",
			checkResponse: func(res *http.Response) {
				require.Equal(t, http.StatusOK, res.StatusCode)
			},
			multipartBody: func(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")

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
				data := []byte("This is the first sentence.\nThis is the second sentence.\n")
				// generate new token
				token2, plaintext2, err := models.GenerateToken()
				require.NoError(t, err)
				token2.UploadUsed = true
				err = tokens.AddToken(ctx, *token2)
				require.NoError(t, err)
				okFormMap2 := map[string]string{
					"file_language_id": strconv.Itoa(title.TitleLangId),
					"title_name":       title.Name,
					"from_voice_id":    strconv.Itoa(title.FromVoiceId),
					"to_voice_id":      strconv.Itoa(title.ToVoiceId),
					"token":            plaintext2,
					"pause":            "4",
					"pattern":          "1",
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

// TestAmazonIntegration tests the audio from file endpoint with the amazon tts client
// Program arguments: -test=integration -project-id=token-tltv-test
func TestAmazonIntegration(t *testing.T) {
	if util.Test != "integration" {
		t.Skip("skipping amazon integration test")
	}

	ctx := context.Background()

	langs, voices := models.MakeAmazonMaps()

	//initialize audiofile with the real command runner
	af := audiofile.New(&audiofile.RealCmdRunner{})
	model := models.Models{
		Languages: langs,
		Voices:    voices,
	}
	tr := translates.New(translates.GoogleClients{}, *translates.NewAmazonClients(), &model, translates.Amazon)

	// Use the application default credentials
	client, err := testCfg.FirestoreClient()
	require.NoError(t, err)
	defer client.Close()

	// generate new token and add it to the collection
	plaintext, tokens := addTokenFirestore(t, client, ctx)

	// defer deleting the collection
	defer func(ctx context.Context, client *firestore.Client, coll *firestore.CollectionRef) {
		err = util.DeleteFirestoreCollection(ctx, client, coll)
		require.NoError(t, err)
	}(ctx, client, tokens.Coll)

	testCfg.Platform = "amazon"
	srv := NewServer(testCfg.Config, tr, af, &tokens, &model)

	e := srv.NewEcho(nil)
	title := test.RandomTitle(voicesMap)

	//create a base path for storing mp3 audio files
	tmpAudioBasePath := test.AudioBasePath + title.Name + "/"
	err = os.MkdirAll(tmpAudioBasePath, 0777)
	require.NoError(t, err)

	// remove directory after tests run
	defer os.RemoveAll(tmpAudioBasePath)

	filename := tmpAudioBasePath + "TestAudioFromFile.txt"

	numVoices := len(voices)
	okFormMap := map[string]string{
		"file_language_id": strconv.Itoa(title.TitleLangId),
		"title_name":       title.Name,
		"from_voice_id":    strconv.Itoa(voices[rand.Intn(numVoices)].ID), //nolint:gosec
		"to_voice_id":      strconv.Itoa(voices[rand.Intn(numVoices)].ID), //nolint:gosec
		"token":            *plaintext,
		"pause":            "4",
		"pattern":          "1",
	}

	testCases := []testCase{

		{
			name: "OK",
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
