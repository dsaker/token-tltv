package translates

import (
	"errors"
	"flag"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"talkliketv.click/tltv/internal/models"
	"talkliketv.click/tltv/internal/testflags"
	"talkliketv.click/tltv/internal/testutil"
	"testing"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"cloud.google.com/go/translate"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/text/language"
	"talkliketv.click/tltv/internal/util"
)

var (
	voicesMap map[int]models.Voice
)

type translatesTestCase struct {
	name           string
	buildStubs     func(stubs testutil.MockStubs)
	checkTranslate func([]models.Phrase, error)
}

func TestGoogleTTS(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	title := testutil.RandomTitle(voicesMap)

	basepath := testutil.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(basepath, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(basepath)

	voice := testutil.RandomVoice()
	voice.SsmlGender = models.MALE
	text1 := "This is sentence one."

	testCases := []translatesTestCase{
		{
			name: "No error",
			buildStubs: func(stubs testutil.MockStubs) {
				req := texttospeechpb.SynthesizeSpeechRequest{
					// Set the text input to be synthesized.
					Input: &texttospeechpb.SynthesisInput{
						InputSource: &texttospeechpb.SynthesisInput_Text{Text: text1},
					},
					// Build the voice request, select the language code ("en-US") and the SSML
					Voice: &texttospeechpb.VoiceSelectionParams{
						LanguageCode: voice.LanguageCode,
						SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
						Name:         voice.Name,
					},
					// Select the type of audio file you want returned.
					AudioConfig: &texttospeechpb.AudioConfig{
						AudioEncoding: texttospeechpb.AudioEncoding_MP3,
					},
				}
				resp := texttospeechpb.SynthesizeSpeechResponse{}
				stubs.GoogleTTsClientX.EXPECT().SynthesizeSpeech(gomock.Any(), &req).Return(&resp, nil)
			},
			checkTranslate: func(translates []models.Phrase, err error) {
				require.NoError(t, err)
				isEmpty, err := IsDirectoryEmpty(basepath)
				require.NoError(t, err)
				require.False(t, isEmpty)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			stubs := testutil.NewMockStubs(ctrl)
			tc.buildStubs(stubs)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/any", nil)
			rec := httptest.NewRecorder()
			newE := e.NewContext(req, rec)

			clients := GoogleClients{
				gtc:  stubs.GoogleTranslateClientX,
				gtts: stubs.GoogleTTsClientX,
			}
			translates := New(clients, AmazonClients{}, stubs.ModelsX)
			err = translates.TextToSpeech(newE, []models.Phrase{{ID: 0, Text: text1}}, voice, basepath)
			tc.checkTranslate(nil, err)
		})
	}
}

func TestAmazonTTS(t *testing.T) {
	if util.Test != "amazon" {
		t.Skip("skipping amazon test")
	}
	t.Parallel()

	title := testutil.RandomTitle(voicesMap)

	basepath := testutil.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(basepath, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(basepath)

	voice := testutil.RandomVoice()
	voice.SsmlGender = models.MALE
	text1 := "This is sentence one."

	testCases := []translatesTestCase{
		{
			name: "Nil response",
			buildStubs: func(stubs testutil.MockStubs) {
				ssi := polly.SynthesizeSpeechInput{
					Text:         &text1,
					VoiceId:      types.VoiceId(voice.Name), // voice.Name
					OutputFormat: "mp3",
				}
				resp := polly.SynthesizeSpeechOutput{}
				//SynthesizeSpeech(context.Context, *polly.SynthesizeSpeechInput, ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
				stubs.AmazonTTsClientX.EXPECT().SynthesizeSpeech(gomock.Any(), &ssi).Return(&resp, nil)
			},
			checkTranslate: func(translates []models.Phrase, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "context canceled")
			},
		},
		{
			name: "No error",
			buildStubs: func(stubs testutil.MockStubs) {
				ssi := polly.SynthesizeSpeechInput{
					Text:         &text1,
					VoiceId:      types.VoiceId(voice.Name), // voice.Name
					OutputFormat: "mp3",
				}
				stringReader := strings.NewReader("shiny!")
				stringReadCloser := io.NopCloser(stringReader)
				resp := polly.SynthesizeSpeechOutput{
					AudioStream: stringReadCloser,
				}
				//SynthesizeSpeech(context.Context, *polly.SynthesizeSpeechInput, ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
				stubs.AmazonTTsClientX.EXPECT().SynthesizeSpeech(gomock.Any(), &ssi).Return(&resp, nil)
			},
			checkTranslate: func(translates []models.Phrase, err error) {
				require.NoError(t, err)
				isEmpty, err := IsDirectoryEmpty(basepath)
				require.NoError(t, err)
				require.False(t, isEmpty)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			stubs := testutil.NewMockStubs(ctrl)
			tc.buildStubs(stubs)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/any", nil)
			rec := httptest.NewRecorder()
			newE := e.NewContext(req, rec)

			clients := AmazonClients{
				atc:  stubs.AmazonTranslateClientX,
				atts: stubs.AmazonTTsClientX,
			}
			translates := New(GoogleClients{}, clients, stubs.ModelsX)
			err = translates.TextToSpeech(newE, []models.Phrase{{ID: 0, Text: text1}}, voice, basepath)
			tc.checkTranslate(nil, err)
		})
	}
}

func TestGoogleTranslate(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	modelsLang := models.Language{Code: "es", Name: "Spanish"}
	text1 := "This is sentence one."
	translate1 := models.Phrase{ID: 0, Text: text1}
	translateText := "Esta es la primera oración."
	returnedPhrase := []models.Phrase{{ID: 0, Text: translateText}, translate1}
	translation := translate.Translation{Text: "Esta es la primera oración."}
	title := testutil.RandomTitle(voicesMap)
	title.TitlePhrases = []models.Phrase{{ID: 0, Text: text1}}

	testCases := []translatesTestCase{
		{
			name: "No error",
			buildStubs: func(stubs testutil.MockStubs) {
				stubs.GoogleTranslateClientX.EXPECT().Translate(gomock.Any(), []string{text1}, language.Spanish, nil).
					Return([]translate.Translation{translation}, nil)
			},
			checkTranslate: func(translates []models.Phrase, err error) {
				require.NoError(t, err)
				testutil.RequireMatchAnyExcept(t, translates[0], returnedPhrase[0], nil, "", nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stubs := testutil.NewMockStubs(ctrl)
			tc.buildStubs(stubs)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/titles/translates", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			clients := GoogleClients{
				gtc:  stubs.GoogleTranslateClientX,
				gtts: stubs.GoogleTTsClientX,
			}
			translates := New(clients, AmazonClients{}, stubs.ModelsX)
			translatesRow, err := translates.TranslatePhrases(c, title, modelsLang)
			tc.checkTranslate(translatesRow, err)
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}
	// Define test cases
	testCases := []struct {
		name        string
		phrases     []string
		buildStubs  func(stubs testutil.MockStubs)
		checkResult func(t *testing.T, result language.Tag, err error)
	}{
		{
			name:    "Successful language detection",
			phrases: []string{"Hello world", "This is a test"},
			buildStubs: func(stubs testutil.MockStubs) {
				// Create detection response with English as the language
				detection := translate.Detection{
					Language:   language.English,
					Confidence: 0.95,
				}
				// Build the expected return type: [][]translate.Detection
				detections := [][]translate.Detection{
					{detection}, // First text
					{detection}, // Second text
				}

				stubs.GoogleTranslateClientX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Eq([]string{"Hello world", "This is a test"})).
					Return(detections, nil)
			},
			checkResult: func(t *testing.T, result language.Tag, err error) {
				require.NoError(t, err)
				require.Equal(t, language.English, result)
			},
		},
		{
			name:    "API error",
			phrases: []string{"Hello world"},
			buildStubs: func(stubs testutil.MockStubs) {
				// Simulate an API error
				stubs.GoogleTranslateClientX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Any()).
					Return([][]translate.Detection(nil), errors.New("API error"))
			},
			checkResult: func(t *testing.T, result language.Tag, err error) {
				require.Error(t, err)
				require.Equal(t, language.Und, result)
			},
		},
		{
			name:    "Empty phrases",
			phrases: []string{},
			buildStubs: func(stubs testutil.MockStubs) {
				// Even with empty phrases, we should call the API
				stubs.GoogleTranslateClientX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Eq([]string{})).
					Return([][]translate.Detection{}, nil)
			},
			checkResult: func(t *testing.T, result language.Tag, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no languages detected")
				require.Equal(t, language.Und, result)
			},
		},
		{
			name:    "Non-English language detection",
			phrases: []string{"Hola mundo", "Esta es una prueba"},
			buildStubs: func(stubs testutil.MockStubs) {
				// Create detection for Spanish
				detection := translate.Detection{
					Language:   language.Spanish,
					Confidence: 0.98,
				}
				// Build return structure
				detections := [][]translate.Detection{
					{detection}, // First text
					{detection}, // Second text
				}

				stubs.GoogleTranslateClientX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Any()).
					Return(detections, nil)
			},
			checkResult: func(t *testing.T, result language.Tag, err error) {
				require.NoError(t, err)
				require.Equal(t, language.Spanish, result)
			},
		},
		{
			name:    "Multiple languages with French highest confidence", // Changed name
			phrases: []string{"Hello world", "Bonjour monde", "Hola mundo"},
			buildStubs: func(stubs testutil.MockStubs) {
				// Create multiple detections with different languages
				engDetection := translate.Detection{
					Language:   language.English,
					Confidence: 0.90,
				}
				frDetection := translate.Detection{
					Language:   language.French,
					Confidence: 0.95,
				}
				esDetection := translate.Detection{
					Language:   language.Spanish,
					Confidence: 0.80,
				}

				// Build the return structure where French has highest confidence
				detections := [][]translate.Detection{
					{frDetection},
					{engDetection},
					{esDetection},
				}

				stubs.GoogleTranslateClientX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Any()).
					Return(detections, nil)
			},
			checkResult: func(t *testing.T, result language.Tag, err error) {
				require.NoError(t, err)
				// French should be detected as it has the highest confidence
				require.Equal(t, language.French, result)
			},
		},
		{
			name:    "Multiple languages with expected order",
			phrases: []string{"Hello world", "Bonjour monde", "Hola mundo"},
			buildStubs: func(stubs testutil.MockStubs) {
				// Create multiple detections with different languages
				engDetection := translate.Detection{
					Language:   language.English,
					Confidence: 0.90,
				}
				frDetection := translate.Detection{
					Language:   language.French,
					Confidence: 0.95, // Highest confidence
				}
				esDetection := translate.Detection{
					Language:   language.Spanish,
					Confidence: 0.80,
				}

				// The order of the detections in the outer array should match the order of phrases
				detections := [][]translate.Detection{
					{frDetection},  // For "Hello world"
					{engDetection}, // For "Bonjour monde"
					{esDetection},  // For "Hola mundo"
				}

				stubs.GoogleTranslateClientX.EXPECT().
					DetectLanguage(gomock.Any(), gomock.Any()).
					Return(detections, nil)
			},
			checkResult: func(t *testing.T, result language.Tag, err error) {
				require.NoError(t, err)
				// Use a direct language constant for comparison, not language.French // Explicitly create the French language tag
				require.Equal(t, language.French, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create controller and mock stubs
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			stubs := testutil.NewMockStubs(ctrl)

			// Build stubs
			tc.buildStubs(stubs)

			// Create echo context
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/detect-language", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create the translate service with mock clients
			clients := GoogleClients{
				gtc:  stubs.GoogleTranslateClientX,
				gtts: stubs.GoogleTTsClientX,
			}
			translate2 := New(clients, AmazonClients{}, stubs.ModelsX)

			// Call the function
			result, err := translate2.DetectLanguage(c.Request().Context(), tc.phrases)

			// Check results
			tc.checkResult(t, result, err)
		})
	}
}

// IsDirectoryEmpty returns true if directory is empty and false if not
func IsDirectoryEmpty(dirPath string) (bool, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Read only one entry
	if err == io.EOF {
		return true, nil // Directory is empty
	}
	return false, nil // Directory is not empty
}

// internal/models/models_test.go
func TestMain(m *testing.M) {
	testflags.ParseFlags()
	flag.Parse()

	util.Test = testflags.TestType

	os.Exit(testflags.RunTests(m))
}
