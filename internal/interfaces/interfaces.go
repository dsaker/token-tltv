package interfaces

import (
	"context"
	"errors"
)

type ModelsStore interface {
	GetLanguage(ctx context.Context, code string) (Language, error)
	GetVoice(ctx context.Context, name string) (Voice, error)
	GetVoices(ctx context.Context) ([]Voice, error)
	GetLanguageCodes(ctx context.Context) ([]LanguageCode, error)
	GetLanguageCode(ctx context.Context, code string) (LanguageCode, error)
	CheckToken(c context.Context, token string) error
	AddToken(ctx context.Context, token Token) error
	UpdateTokenField(c context.Context, value any, token, path string) error
	DeleteToken(ctx context.Context, hash string) error
}

var (
	ErrTooManyPhrases = errors.New("too many phrases")
	ErrPauseNotFound  = errors.New("audio pause file not found")
)
