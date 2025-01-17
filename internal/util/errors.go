package util

import (
	"errors"
	"fmt"
)

var (
	ErrOneFile           = errors.New("no need to zip one file")
	ErrUnableToParseFile = func(err error) error {
		return fmt.Errorf("unable to parse file: %s", err)
	}
	ErrTooManyPhrases    = errors.New("too many phrases")
	ErrVoiceIdInvalid    = errors.New("voice id invalid")
	ErrPauseNotFound     = errors.New("audio pause file not found")
	ErrLanguageIdInvalid = errors.New("language id invalid")
	ErrPauseInvalid      = errors.New("pause out of range (must be between 3 and 10")
)
