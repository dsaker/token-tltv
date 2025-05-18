package models

import (
	"context"
	"fmt"
	"talkliketv.com/tltv/internal/interfaces"
)

// GetLanguage retrieves a language by code
func (m *Models) GetLanguage(ctx context.Context, code string) (interfaces.Language, error) {
	// Try to refresh cache if needed
	if err := m.refreshCache(ctx); err != nil {
		// If cache refresh fails, try direct document lookup
		docRef := m.langCollection.Doc(code)
		doc, err := docRef.Get(ctx)
		if err != nil {
			return interfaces.Language{}, fmt.Errorf("language not found: %w", err)
		}

		var lang interfaces.Language
		if err := doc.DataTo(&lang); err != nil {
			return interfaces.Language{}, fmt.Errorf("error parsing language: %w", err)
		}
		return lang, nil
	}

	// Look up in cache
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	lang, ok := m.languageCache[code]
	if !ok {
		return interfaces.Language{}, ErrLanguageIdInvalid
	}
	return lang, nil
}
