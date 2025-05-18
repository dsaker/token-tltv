package models

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"talkliketv.com/tltv/internal/interfaces"
	"time"
)

var (
	ErrLanguageIdInvalid   = errors.New("language id invalid")
	ErrLanguageCodeInvalid = errors.New("language code invalid")
	ErrVoiceIdInvalid      = errors.New("voice id invalid")
	ErrUsedToken           = errors.New("token already used")
)

const (
	LangCollString     = "languages"
	VoiceCollString    = "voices"
	LangCodeCollString = "languageCodes"
	TokenCollString    = "tokens"
)

// Models implements ModelsX interface using Firestore
type Models struct {
	tokenCollection    *firestore.CollectionRef // Add this field
	langCollection     *firestore.CollectionRef
	voiceCollection    *firestore.CollectionRef
	langCodeCollection *firestore.CollectionRef
	languageCache      map[string]interfaces.Language
	languageCodeCache  []interfaces.LanguageCode
	voiceCache         []interfaces.Voice
	cacheExpiration    time.Time
	cacheDuration      time.Duration
	cacheMutex         sync.RWMutex
}

// FirestoreClient interface defines the methods we use from firestore.Client
type FirestoreClient interface {
	Collection(path string) *firestore.CollectionRef
}

// NewModels creates a new Models instance
func NewModels(client FirestoreClient, env, lang, langCode, voice, tok string) (*Models, error) {
	// Validate collection paths
	collections := []string{lang, langCode, voice, tok}
	for _, coll := range collections {
		if err := validateCollectionPath(coll); err != nil {
			return nil, fmt.Errorf("warning: Invalid Firestore collection path: %s, error: %v", coll, err)
		}
	}

	models := &Models{
		langCollection:     client.Collection(lang),
		langCodeCollection: client.Collection(langCode),
		voiceCollection:    client.Collection(voice),
		tokenCollection:    client.Collection(tok),
		cacheExpiration:    time.Time{},
		cacheDuration:      60 * time.Minute,
		languageCache:      make(map[string]interfaces.Language),
		languageCodeCache:  make([]interfaces.LanguageCode, 0),
		voiceCache:         make([]interfaces.Voice, 0),
	}

	if env == "prod" {
		err := models.refreshCache(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to refresh cache: %w", err)
		}
	}

	return models, nil
}

// validateCollectionPath checks if a collection path follows Firestore rules
func validateCollectionPath(path string) error {
	if path == "" {
		return fmt.Errorf("collection path cannot be empty")
	}

	// Check for invalid characters in the path
	// Firestore collection names can't contain: '/', '.', '..', '/', '//'
	invalidPatterns := []string{"/", ".", "..", "//"}
	for _, pattern := range invalidPatterns {
		if strings.Contains(path, pattern) {
			return fmt.Errorf("collection path cannot contain '%s'", pattern)
		}
	}

	// Check that path is not too long (Firestore has a limit)
	if len(path) > 1500 {
		return fmt.Errorf("collection path too long, maximum is 1500 characters")
	}

	return nil
}

// refreshCache loads all languages and voices from Firestore
func (m *Models) refreshCache(ctx context.Context) error {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	// Check if cache is still valid
	if !m.cacheExpiration.IsZero() && time.Now().Before(m.cacheExpiration) {
		return nil
	}

	// Load languages
	langDocs, err := m.langCollection.Documents(ctx).GetAll()
	if err != nil || len(langDocs) == 0 {
		return fmt.Errorf("failed to get languages: %v", err)
	}

	newLangCache := make(map[string]interfaces.Language)
	for _, doc := range langDocs {
		var lang interfaces.Language
		if err := doc.DataTo(&lang); err != nil {
			return fmt.Errorf("error parsing language data: %w", err)
		}
		newLangCache[doc.Ref.ID] = lang
	}

	// Load language codes
	langCodeDocs, err := m.langCodeCollection.Documents(ctx).GetAll()
	if err != nil || len(langCodeDocs) == 0 {
		return fmt.Errorf("failed to get language codes: %w", err)
	}

	newLangCodeCache := make([]interfaces.LanguageCode, 0, len(langCodeDocs))
	for _, doc := range langCodeDocs {
		var langCode interfaces.LanguageCode
		if err := doc.DataTo(&langCode); err != nil {
			return fmt.Errorf("error parsing language code data: %w", err)
		}
		newLangCodeCache = append(newLangCodeCache, langCode)
	}

	// Sort language codes by name
	sort.Slice(newLangCodeCache, func(i, j int) bool {
		return newLangCodeCache[i].Name < newLangCodeCache[j].Name
	})

	// Load voices
	voiceDocs, err := m.voiceCollection.Documents(ctx).GetAll()
	if err != nil || len(voiceDocs) == 0 {
		return fmt.Errorf("failed to get voices: %w", err)
	}

	newVoiceCache := make([]interfaces.Voice, 0, len(voiceDocs))
	for _, doc := range voiceDocs {
		var voice interfaces.Voice
		if err := doc.DataTo(&voice); err != nil {
			return fmt.Errorf("error parsing voice data: %w", err)
		}
		newVoiceCache = append(newVoiceCache, voice)
	}

	// Sort voices by name
	sort.Slice(newVoiceCache, func(i, j int) bool {
		return newVoiceCache[i].Name < newVoiceCache[j].Name
	})

	// Update cache
	m.languageCache = newLangCache
	m.languageCodeCache = newLangCodeCache
	m.voiceCache = newVoiceCache
	m.cacheExpiration = time.Now().Add(m.cacheDuration)

	return nil
}
