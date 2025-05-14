package models

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
)

var (
	ErrTooManyPhrases      = errors.New("too many phrases")
	ErrVoiceIdInvalid      = errors.New("voice id invalid")
	ErrPauseNotFound       = errors.New("audio pause file not found")
	ErrLanguageIdInvalid   = errors.New("language id invalid")
	ErrVoiceNotFound       = errors.New("voice not found")
	ErrLanguageCodeInvalid = errors.New("language code invalid")
)

type Title struct {
	Name         string
	TitleLang    string
	ToVoice      string
	FromVoice    string
	Pause        int
	TitlePhrases []Phrase
	ToPhrases    []Phrase
	Pattern      int
}

type Phrase struct {
	ID   int
	Text string
}

type Status int

const (
	New Status = iota
	Used
)

type ModelsX interface {
	GetLanguage(ctx context.Context, code string) (Language, error)
	GetVoice(ctx context.Context, name string) (Voice, error)
	GetVoices(ctx context.Context) ([]Voice, error)
	GetVoicesByLanguage(ctx context.Context, languageCode string) ([]Voice, error)
	GetVoicesByPlatform(ctx context.Context, platform string) ([]Voice, error)
	GetVoicesByPlatformAndLanguage(ctx context.Context, platform, languageCode string) ([]Voice, error)
	GetLanguageCodes(ctx context.Context) ([]LanguageCode, error)
}

// Models implements ModelsX interface using Firestore
type Models struct {
	client             *firestore.Client
	langCollection     string
	voiceCollection    string
	langCodeCollection string
	languageCache      map[string]Language
	languageCodeCache  []LanguageCode
	voiceCache         []Voice
	cacheExpiration    time.Time
	cacheDuration      time.Duration
	cacheMutex         sync.RWMutex
}

// NewModels creates a new Models instance
func NewModels(client *firestore.Client, langCollection, voiceCollection, langCodeCollection string) *Models {
	return &Models{
		client:             client,
		langCollection:     langCollection,
		langCodeCollection: langCodeCollection,
		voiceCollection:    voiceCollection,
		cacheDuration:      60 * time.Minute,
		languageCache:      make(map[string]Language),
		languageCodeCache:  make([]LanguageCode, 0),
		voiceCache:         make([]Voice, 0),
	}
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
	langDocs, err := m.client.Collection(m.langCollection).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get languages: %w", err)
	}

	newLangCache := make(map[string]Language)
	for _, doc := range langDocs {
		var lang Language
		if err := doc.DataTo(&lang); err != nil {
			return fmt.Errorf("error parsing language data: %w", err)
		}
		newLangCache[doc.Ref.ID] = lang
	}

	// Load language codes
	langCodeDocs, err := m.client.Collection(m.langCodeCollection).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get language codes: %w", err)
	}

	newLangCodeCache := make([]LanguageCode, 0, len(langCodeDocs))
	for _, doc := range langCodeDocs {
		var langCode LanguageCode
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
	voiceDocs, err := m.client.Collection(m.voiceCollection).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get voices: %w", err)
	}

	newVoiceCache := make([]Voice, 0, len(voiceDocs))
	for _, doc := range voiceDocs {
		var voice Voice
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
