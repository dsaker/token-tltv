package test

import (
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"talkliketv.click/tltv/internal/models"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	AudioBasePath = "/tmp/test/audio/"
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
	DefaultPause   = 5
	DefaultPattern = 2
	alphabet       = "abcdefghijklmnopqrstuvwxyz"
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
		ID:   rand.Intn(100),
		Text: RandomString(20),
	}
}

// RandomVoice creates a random Voice for testing
func RandomVoice() models.Voice {
	return models.Voice{
		LangId:                 rand.Intn(100),
		LanguageCodes:          []string{RandomString(8), RandomString(8)},
		SsmlGender:             "FEMALE",
		Name:                   RandomString(8),
		NaturalSampleRateHertz: 24000,
	}
}

func RandomTitle() (title models.Title) {
	return models.Title{
		Name:        RandomString(8),
		TitleLangId: rand.Intn(100),
		ToVoiceId:   rand.Intn(100),
		FromVoiceId: rand.Intn(100),
		Pause:       DefaultPause,
		Phrases:     nil,
		Translates:  nil,
		Pattern:     DefaultPattern,
	}
}
