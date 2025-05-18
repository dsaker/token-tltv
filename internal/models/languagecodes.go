package models

import (
	"context"
	"fmt"
	"talkliketv.com/tltv/internal/interfaces"
)

// GetLanguageCode retrieves a languageCode by code
func (m *Models) GetLanguageCode(ctx context.Context, code string) (interfaces.LanguageCode, error) {
	// Try to refresh cache if needed
	if err := m.refreshCache(ctx); err != nil {
		// If cache refresh fails, try direct document lookup
		docRef := m.langCodeCollection.Doc(code)
		doc, err := docRef.Get(ctx)
		if err != nil {
			return interfaces.LanguageCode{}, fmt.Errorf("language code not found: %w", err)
		}

		var langCode interfaces.LanguageCode
		if err := doc.DataTo(&langCode); err != nil {
			return interfaces.LanguageCode{}, fmt.Errorf("error parsing language code: %w", err)
		}
		return langCode, nil
	}

	// Look up in cache
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	for _, langCode := range m.languageCodeCache {
		if langCode.Code == code {
			return langCode, nil
		}
	}
	return interfaces.LanguageCode{}, ErrLanguageCodeInvalid
}

// GetLanguageCodes returns all languages codes sorted by name
func (m *Models) GetLanguageCodes(ctx context.Context) ([]interfaces.LanguageCode, error) {
	if err := m.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to load languages: %w", err)
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// Return a copy of the cached slice
	result := make([]interfaces.LanguageCode, len(m.languageCodeCache))
	copy(result, m.languageCodeCache)
	return result, nil
}
