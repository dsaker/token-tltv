package models

import (
	"context"
	"fmt"
)

type LanguageCode struct {
	Code     string `firestore:"code"`
	Name     string `firestore:"name"`
	Language string `firestore:"language"`
	Country  string `firestore:"country"`
	Platform string `firestore:"platform"`
}

// GetLanguageCode retrieves a languageCode by code
func (m *Models) GetLanguageCode(ctx context.Context, code string) (LanguageCode, error) {
	// Try to refresh cache if needed
	if err := m.refreshCache(ctx); err != nil {
		// If cache refresh fails, try direct document lookup
		docRef := m.client.Collection(m.langCodeCollection).Doc(code)
		doc, err := docRef.Get(ctx)
		if err != nil {
			return LanguageCode{}, fmt.Errorf("language code not found: %w", err)
		}

		var langCode LanguageCode
		if err := doc.DataTo(&langCode); err != nil {
			return LanguageCode{}, fmt.Errorf("error parsing language code: %w", err)
		}
		return langCode, nil
	}

	// Look up in cache
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	langCode, ok := m.languageCodeCache[code]
	if !ok {
		return LanguageCode{}, ErrLanguageCodeInvalid
	}
	return langCode, nil
}

// GetLanguageCodes returns all languages codes
func (m *Models) GetLanguageCodes(ctx context.Context) (map[string]LanguageCode, error) {
	if err := m.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to load languages: %w", err)
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// Create a copy of the map to avoid concurrent access issues
	result := make(map[string]LanguageCode, len(m.languageCodeCache))
	for k, v := range m.languageCodeCache {
		result[k] = v
	}

	return result, nil
}
