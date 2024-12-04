package util

import (
	"errors"
	"fmt"
)

var (
	ErrVoiceLangIdNoMatch = errors.New("voice id does not match chosen language id")
	ErrOneFile            = errors.New("no need to zip one file")
	ErrUnableToParseFile  = func(err error) error {
		return fmt.Errorf("unable to parse file: %s", err)
	}
	ErrTooManyPhrases = errors.New("too many phrases")
	ErrIntConversion  = errors.New("conversion failed: The interface does not hold an int")
	ErrVoiceIdInvalid = errors.New("voice id invalid")
)
