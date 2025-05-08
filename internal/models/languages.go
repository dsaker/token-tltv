package models

import (
	"context"
	"fmt"
)

type Language struct {
	Name     string `firestore:"name"`
	Code     string `firestore:"code"`
	Platform string `firestore:"platform"`
}

// GetLanguage retrieves a language by code
func (m *Models) GetLanguage(ctx context.Context, code string) (Language, error) {
	// Try to refresh cache if needed
	if err := m.refreshCache(ctx); err != nil {
		// If cache refresh fails, try direct document lookup
		docRef := m.client.Collection(m.langCollection).Doc(code)
		doc, err := docRef.Get(ctx)
		if err != nil {
			return Language{}, fmt.Errorf("language not found: %w", err)
		}

		var lang Language
		if err := doc.DataTo(&lang); err != nil {
			return Language{}, fmt.Errorf("error parsing language: %w", err)
		}
		return lang, nil
	}

	// Look up in cache
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	lang, ok := m.languageCache[code]
	if !ok {
		return Language{}, ErrLanguageIdInvalid
	}
	return lang, nil
}

// GetLanguages returns all languages
func (m *Models) GetLanguages(ctx context.Context) (map[string]Language, error) {
	if err := m.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to load languages: %w", err)
	}

	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// Create a copy of the map to avoid concurrent access issues
	result := make(map[string]Language, len(m.languageCache))
	for k, v := range m.languageCache {
		result[k] = v
	}

	return result, nil
}
