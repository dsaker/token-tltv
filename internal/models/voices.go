package models

import (
	"context"
	"fmt"
	"strings"
)

type Gender int

const (
	MALE Gender = iota + 1
	FEMALE
	NEUTRAL
)

type Voice struct {
	Name                   string `firestore:"name"`
	Language               string `firestore:"language"`
	LanguageCode           string `firestore:"languageCodes"`
	SsmlGender             Gender `firestore:"ssmlGender"`
	NaturalSampleRateHertz int32  `firestore:"naturalSampleRateHertz"`
	Platform               string `firestore:"platform"`            // "google" or "amazon"
	SampleURL              string `firestore:"sampleUrl,omitempty"` // URL to sample audio
}

// GetVoice retrieves a voice by name
func (m *Models) GetVoice(ctx context.Context, name string) (Voice, error) {
	// Try to refresh cache if needed
	if err := m.refreshCache(ctx); err != nil {
		// If cache refresh fails, try direct document lookup
		docRef := m.client.Collection(m.voiceCollection).Doc(name)
		doc, err := docRef.Get(ctx)
		if err != nil {
			return Voice{}, fmt.Errorf("voice not found: %w", err)
		}

		var voice Voice
		if err := doc.DataTo(&voice); err != nil {
			return Voice{}, fmt.Errorf("error parsing voice: %w", err)
		}
		return voice, nil
	}

	// Look up in cache
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	for _, voice := range m.voiceCache {
		if voice.Name == name {
			return voice, nil
		}
	}
	return Voice{}, ErrLanguageCodeInvalid
}

// GetVoices returns all voices
func (m *Models) GetVoices(ctx context.Context) ([]Voice, error) {
	if err := m.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to load voices: %w", err)
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// Convert map to slice
	result := make([]Voice, 0, len(m.voiceCache))
	for _, v := range m.voiceCache {
		result = append(result, v)
	}

	return result, nil
}

// GetVoicesByLanguage returns all voices for a specific language code
func (m *Models) GetVoicesByLanguage(ctx context.Context, languageCode string) ([]Voice, error) {
	// First get all voices
	voices, err := m.GetVoices(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by language code
	result := make([]Voice, 0)
	for _, voice := range voices {
		if strings.HasPrefix(voice.LanguageCode, languageCode) {
			result = append(result, voice)
		}
	}

	return result, nil
}

// GetVoicesByPlatform returns all voices for a specific platform
func (m *Models) GetVoicesByPlatform(ctx context.Context, platform string) ([]Voice, error) {
	// First get all voices
	voices, err := m.GetVoices(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by platform
	result := make([]Voice, 0)
	for _, voice := range voices {
		if voice.Platform == platform {
			result = append(result, voice)
		}
	}

	return result, nil
}

// GetVoicesByPlatformAndLanguage returns all voices for a specific platform and language code
func (m *Models) GetVoicesByPlatformAndLanguage(ctx context.Context, platform, languageCode string) ([]Voice, error) {
	// First get all voices
	voices, err := m.GetVoices(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by both platform and language code
	result := make([]Voice, 0)
	for _, voice := range voices {
		if voice.Platform != platform {
			continue
		}

		if strings.HasPrefix(voice.LanguageCode, languageCode) {
			result = append(result, voice)
		}
	}

	return result, nil
}
