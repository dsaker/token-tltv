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

	for _, langCode := range m.languageCodeCache {
		if langCode.Code == code {
			return langCode, nil
		}
	}
	return LanguageCode{}, ErrLanguageCodeInvalid
}

// GetLanguageCodes returns all languages codes sorted by name
func (m *Models) GetLanguageCodes(ctx context.Context) ([]LanguageCode, error) {
	if err := m.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to load languages: %w", err)
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// Return a copy of the cached slice
	result := make([]LanguageCode, len(m.languageCodeCache))
	copy(result, m.languageCodeCache)
	return result, nil
}
