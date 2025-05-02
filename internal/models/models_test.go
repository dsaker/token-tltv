package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"talkliketv.click/tltv/internal/util"
)

func TestModelsGetLanguage(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	languages := map[int]Language{
		1: {ID: 1, Code: "en", Name: "English"},
		2: {ID: 2, Code: "es", Name: "Spanish"},
	}

	models := &Models{
		Languages: languages,
		Voices:    make(map[int]Voice),
	}

	testCases := []struct {
		name          string
		languageID    int
		expectedLang  Language
		expectedError error
	}{
		{
			name:          "Valid language ID",
			languageID:    1,
			expectedLang:  Language{ID: 1, Code: "en", Name: "English"},
			expectedError: nil,
		},
		{
			name:          "Another valid language ID",
			languageID:    2,
			expectedLang:  Language{ID: 2, Code: "es", Name: "Spanish"},
			expectedError: nil,
		},
		{
			name:          "Invalid language ID",
			languageID:    999,
			expectedLang:  Language{},
			expectedError: ErrLanguageIdInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lang, err := models.GetLanguage(tc.languageID)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedLang, lang)
			}
		})
	}
}

func TestModelsGetVoice(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	voices := map[int]Voice{
		1: {ID: 1, VoiceName: "en-US-Standard-A", Gender: MALE},
		2: {ID: 2, VoiceName: "es-ES-Standard-A", Gender: FEMALE},
	}

	models := &Models{
		Languages: make(map[int]Language),
		Voices:    voices,
	}

	testCases := []struct {
		name          string
		voiceID       int
		expectedVoice Voice
		expectedError error
	}{
		{
			name:          "Valid voice ID",
			voiceID:       1,
			expectedVoice: Voice{ID: 1, VoiceName: "en-US-Standard-A", Gender: MALE},
			expectedError: nil,
		},
		{
			name:          "Another valid voice ID",
			voiceID:       2,
			expectedVoice: Voice{ID: 2, VoiceName: "es-ES-Standard-A", Gender: FEMALE},
			expectedError: nil,
		},
		{
			name:          "Invalid voice ID",
			voiceID:       999,
			expectedVoice: Voice{},
			expectedError: ErrVoiceIdInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			voice, err := models.GetVoice(tc.voiceID)

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedVoice, voice)
			}
		})
	}
}

func TestModelsGetLanguages(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	languages := map[int]Language{
		1: {ID: 1, Code: "en", Name: "English"},
		2: {ID: 2, Code: "es", Name: "Spanish"},
	}

	models := &Models{
		Languages: languages,
		Voices:    make(map[int]Voice),
	}

	result := models.GetLanguages()
	assert.Equal(t, languages, result)
}

func TestModelsGetVoices(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	voices := map[int]Voice{
		1: {ID: 1, VoiceName: "en-US-Standard-A", Gender: MALE},
		2: {ID: 2, VoiceName: "es-ES-Standard-A", Gender: FEMALE},
	}

	models := &Models{
		Languages: make(map[int]Language),
		Voices:    voices,
	}

	result := models.GetVoices()
	assert.Equal(t, voices, result)
}

func TestErrorConstants(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	// Test that the error constants are defined correctly
	assert.Equal(t, "too many phrases", ErrTooManyPhrases.Error())
	assert.Equal(t, "voice id invalid", ErrVoiceIdInvalid.Error())
	assert.Equal(t, "audio pause file not found", ErrPauseNotFound.Error())
	assert.Equal(t, "language id invalid", ErrLanguageIdInvalid.Error())
}

func TestGenderConstants(t *testing.T) {
	if util.Test != "unit" {
		t.Skip("skipping unit test")
	}

	t.Parallel()

	// Test that Gender enum values are defined correctly
	assert.Equal(t, Gender(1), MALE)
	assert.Equal(t, Gender(2), FEMALE)
	assert.Equal(t, Gender(3), NEUTRAL)
}
