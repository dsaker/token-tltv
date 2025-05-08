package models

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrTooManyPhrases    = errors.New("too many phrases")
	ErrVoiceIdInvalid    = errors.New("voice id invalid")
	ErrPauseNotFound     = errors.New("audio pause file not found")
	ErrLanguageIdInvalid = errors.New("language id invalid")
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
	GetLanguages(ctx context.Context) (map[string]Language, error)
	GetVoices(ctx context.Context) (map[string]Voice, error)
	GetVoicesByLanguage(ctx context.Context, languageCode string) (map[string]Voice, error)
	GetVoicesByPlatform(ctx context.Context, platform string) (map[string]Voice, error)
	GetVoicesByPlatformAndLanguage(ctx context.Context, platform, languageCode string) (map[string]Voice, error)
}

// Models implements ModelsX interface using Firestore
type Models struct {
	client          *firestore.Client
	langCollection  string
	voiceCollection string
	languageCache   map[string]Language
	voiceCache      map[string]Voice
	cacheExpiration time.Time
	cacheDuration   time.Duration
	cacheMutex      sync.RWMutex
}

// NewModels creates a new Models instance
func NewModels(client *firestore.Client, langCollection, voiceCollection string) *Models {
	return &Models{
		client:          client,
		langCollection:  langCollection,
		voiceCollection: voiceCollection,
		cacheDuration:   15 * time.Minute,
		languageCache:   make(map[string]Language),
		voiceCache:      make(map[string]Voice),
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

	// Load voices
	voiceDocs, err := m.client.Collection(m.voiceCollection).Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("failed to get voices: %w", err)
	}

	newVoiceCache := make(map[string]Voice)
	for _, doc := range voiceDocs {
		var voice Voice
		if err := doc.DataTo(&voice); err != nil {
			return fmt.Errorf("error parsing voice data: %w", err)
		}
		newVoiceCache[doc.Ref.ID] = voice
	}

	// Update cache
	m.languageCache = newLangCache
	m.voiceCache = newVoiceCache
	m.cacheExpiration = time.Now().Add(m.cacheDuration)

	return nil
}
