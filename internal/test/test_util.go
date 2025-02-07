package test

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"talkliketv.click/tltv/internal/mock"
	"talkliketv.click/tltv/internal/models"
	"testing"
)

var (
	AudioBasePath     = "/tmp/test/audio/"
	GcpTestProject    = "token-tltv-test"
	FirestoreTestColl = "token-tltv-test"
)

func RequireMatchAnyExcept(t *testing.T, model any, response any, skip []string, except string, shouldEqual any) {
	v := reflect.ValueOf(response)
	u := reflect.ValueOf(model)

	for i := 0; i < v.NumField(); i++ {
		// Check if field name is the one that should be different
		if v.Type().Field(i).Name == except {
			// Check if type is int32 or int64
			if v.Field(i).CanInt() {
				// check if equal as int64
				require.Equal(t, shouldEqual, v.Field(i).Int())
			} else {
				// if not check if equal as string
				require.Equal(t, shouldEqual, v.Field(i).String())
			}
		} else if slices.Contains(skip, v.Type().Field(i).Name) {
			continue
		} else {
			if v.Field(i).CanInt() {
				require.Equal(t, u.Field(i).Int(), v.Field(i).Int())
			} else {
				require.Equal(t, u.Field(i).String(), v.Field(i).String())
			}
		}
	}
}

const (
	DefaultPause            = 5
	DefaultPattern          = 1
	MaxLanguages            = 75
	MaxVoices               = 95
	ValidLangId             = 16
	alphabet                = "abcdefghijklmnopqrstuvwxyz"
	FirestoreTestCollection = "token-tltv-test"
	GcPTestProject          = "token-tltv-test"
)

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)] //nolint:gosec
		sb.WriteByte(c)
	}

	return sb.String()
}

func RandomPhrase() models.Phrase {
	return models.Phrase{
		ID:   rand.Intn(100), //nolint:gosec
		Text: RandomString(20),
	}
}

// RandomVoice creates a random Voice for testing
func RandomVoice() models.Voice {
	return models.Voice{
		LangId:                 rand.Intn(MaxLanguages), //nolint:gosec
		LanguageCodes:          []string{RandomString(8), RandomString(8)},
		Gender:                 2,
		VoiceName:              RandomString(8),
		NaturalSampleRateHertz: 24000,
	}
}

func RandomGoogleTitle() (title models.Title) {
	return models.Title{
		Name:        RandomString(8),
		TitleLangId: ValidLangId,
		ToVoiceId:   models.Voices[rand.Intn(MaxVoices)].ID, //nolint:gosec
		FromVoiceId: models.Voices[rand.Intn(MaxVoices)].ID, //nolint:gosec
		Pause:       DefaultPause,
		Pattern:     DefaultPattern,
	}
}

type MockStubs struct {
	TranslateX             *mock.MockTranslateX
	GoogleTranslateClientX *mock.MockGoogleTranslateClientX
	GoogleTTsClientX       *mock.MockGoogleTTSClientX
	AmazonTranslateClientX *mock.MockAmazonTranslateClientX
	AmazonTTsClientX       *mock.MockAmazonTTSClientX
	AudioFileX             *mock.MockAudioFileX
	TokensX                *mock.MockTokensX
	ModelsX                *mock.MockModelsX
}

// NewMockStubs creates instantiates new instances of all the mock interfaces for testing
func NewMockStubs(ctrl *gomock.Controller) MockStubs {
	return MockStubs{
		TranslateX:             mock.NewMockTranslateX(ctrl),
		GoogleTranslateClientX: mock.NewMockGoogleTranslateClientX(ctrl),
		GoogleTTsClientX:       mock.NewMockGoogleTTSClientX(ctrl),
		AudioFileX:             mock.NewMockAudioFileX(ctrl),
		AmazonTranslateClientX: mock.NewMockAmazonTranslateClientX(ctrl),
		AmazonTTsClientX:       mock.NewMockAmazonTTSClientX(ctrl),
		TokensX:                mock.NewMockTokensX(ctrl),
		ModelsX:                mock.NewMockModelsX(ctrl),
	}
}
