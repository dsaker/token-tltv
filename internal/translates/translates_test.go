package translates

import (
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"talkliketv.click/tltv/internal/models"
	"testing"

	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"cloud.google.com/go/translate"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/text/language"
	"talkliketv.click/tltv/internal/mock/translates"
	"talkliketv.click/tltv/internal/test"
	"talkliketv.click/tltv/internal/util"
)

type translatesTestCase struct {
	name           string
	buildStubs     func(*mockt.MockTranslateX, *mockt.MockGoogleTranslateClientX, *mockt.MockGoogleTTSClientX)
	checkTranslate func([]models.Phrase, error)
}

func TestGoogleTTS(t *testing.T) {
	if util.Integration {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	title := test.RandomTitle()

	basepath := test.AudioBasePath + title.Name + "/"
	err := os.MkdirAll(basepath, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(basepath)

	voice := test.RandomVoice()
	voice.SsmlGender = "MALE"
	text1 := "This is sentence one."

	testCases := []translatesTestCase{
		{
			name: "No error",
			buildStubs: func(t *mockt.MockTranslateX, tc *mockt.MockGoogleTranslateClientX, tts *mockt.MockGoogleTTSClientX) {
				req := texttospeechpb.SynthesizeSpeechRequest{
					// Set the text input to be synthesized.
					Input: &texttospeechpb.SynthesisInput{
						InputSource: &texttospeechpb.SynthesisInput_Text{Text: text1},
					},
					// Build the voice request, select the language code ("en-US") and the SSML
					// voice gender ("neutral").
					Voice: &texttospeechpb.VoiceSelectionParams{
						LanguageCode: voice.LanguageCodes[0],
						SsmlGender:   texttospeechpb.SsmlVoiceGender_MALE,
						Name:         voice.Name,
					},
					// Select the type of audio file you want returned.
					AudioConfig: &texttospeechpb.AudioConfig{
						AudioEncoding: texttospeechpb.AudioEncoding_MP3,
					},
				}
				resp := texttospeechpb.SynthesizeSpeechResponse{}
				tts.EXPECT().SynthesizeSpeech(gomock.Any(), &req).Return(&resp, nil)
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

			text := mockt.NewMockTranslateX(ctrl)
			gtc := mockt.NewMockGoogleTranslateClientX(ctrl)
			gtts := mockt.NewMockGoogleTTSClientX(ctrl)
			tc.buildStubs(text, gtc, gtts)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/any", nil)
			rec := httptest.NewRecorder()
			newE := e.NewContext(req, rec)

			clients := GoogleClients{
				gtc:  gtc,
				gtts: gtts,
			}
			translates := New(clients, AmazonClients{})
			err = translates.TextToSpeech(newE, []models.Phrase{{ID: 0, Text: text1}}, voice, basepath)
			tc.checkTranslate(nil, err)
		})
	}
}

func TestGoogleTranslate(t *testing.T) {
	if util.Integration {
		t.Skip("skipping unit test")
	}
	t.Parallel()

	modelsLang := models.Language{ID: 0, Code: "es", Name: "Spanish"}
	text1 := "This is sentence one."
	translate1 := models.Phrase{ID: 0, Text: text1}
	translateText := "Esta es la primera oración."
	returnedPhrase := []models.Phrase{{ID: 0, Text: translateText}, translate1}
	translation := translate.Translation{Text: "Esta es la primera oración."}
	testCases := []translatesTestCase{
		{
			name: "No error",
			buildStubs: func(t *mockt.MockTranslateX, tr *mockt.MockGoogleTranslateClientX, ts *mockt.MockGoogleTTSClientX) {
				tr.EXPECT().Translate(gomock.Any(), []string{text1}, language.Spanish, nil).
					Return([]translate.Translation{translation}, nil)
			},
			checkTranslate: func(translates []models.Phrase, err error) {
				require.NoError(t, err)
				test.RequireMatchAnyExcept(t, translates[0], returnedPhrase[0], nil, "", nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			text := mockt.NewMockTranslateX(ctrl)
			trc := mockt.NewMockGoogleTranslateClientX(ctrl)
			tts := mockt.NewMockGoogleTTSClientX(ctrl)
			tc.buildStubs(text, trc, tts)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/titles/translates", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			clients := GoogleClients{
				gtc:  trc,
				gtts: tts,
			}
			translates := New(clients, AmazonClients{})
			translatesRow, err := translates.TranslatePhrases(c, []models.Phrase{translate1}, modelsLang)
			tc.checkTranslate(translatesRow, err)
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

func TestMain(m *testing.M) {
	flag.BoolVar(&util.Integration, "integration", false, "Run integration tests")
	flag.Parse()
	os.Exit(m.Run())
}
