package models

import (
	"context"
	"fmt"
	"talkliketv.com/tltv/internal/interfaces"
)

// GetVoice retrieves a voice by name
func (m *Models) GetVoice(ctx context.Context, name string) (interfaces.Voice, error) {
	// Try to refresh cache if needed
	if err := m.refreshCache(ctx); err != nil {
		// If cache refresh fails, try direct document lookup
		docRef := m.voiceCollection.Doc(name)
		doc, err := docRef.Get(ctx)
		if err != nil {
			return interfaces.Voice{}, fmt.Errorf("voice not found: %w", err)
		}

		var voice interfaces.Voice
		if err := doc.DataTo(&voice); err != nil {
			return interfaces.Voice{}, fmt.Errorf("error parsing voice: %w", err)
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
	return interfaces.Voice{}, ErrVoiceIdInvalid
}

func (m *Models) GetVoices(ctx context.Context) ([]interfaces.Voice, error) {
	if err := m.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to load voices: %w", err)
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	result := make([]interfaces.Voice, 0, len(m.voiceCache))
	result = append(result, m.voiceCache...)

	return result, nil
}
