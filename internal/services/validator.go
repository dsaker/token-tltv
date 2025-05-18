package services

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/mail"
	"strconv"
	"strings"
	"talkliketv.com/tltv/internal/interfaces"
	"unicode/utf8"
)

// Validator Add a new NonFieldErrors []string field to the struct, which we will use to
// hold any validation errors which are not related to a specific form field.
type Validator struct {
	NonFieldErrors []string
	FieldErrors    map[string]string
}

// Valid Update the Valid() method to also check that the NonFieldErrors slice is
// empty.
func (v *Validator) Valid() bool {
	return len(v.FieldErrors) == 0 && len(v.NonFieldErrors) == 0
}

// In returns true if a specific value is in a list of strings.
func In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

// CheckField adds an error message to the FieldErrors map only if a
// validation check is not 'ok'.
func (v *Validator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

// Matches returns true if a value matches a provided compiled regular
// expression pattern.
//func Matches(value string, rx *regexp.Regexp) bool {
//	return rx.MatchString(value)
//}

func (v *Validator) IsEmail(email string) bool {
	emailAddress, err := mail.ParseAddress(email)
	return err == nil && emailAddress.Address == email
}

// AddNonFieldError Create an AddNonFieldError() helper for adding error messages to the new
// NonFieldErrors slice.
func (v *Validator) AddNonFieldError(message string) {
	v.NonFieldErrors = append(v.NonFieldErrors, message)
}

// AddFieldError adds an error message to the FieldErrors map (so long as no
// entry already exists for the given key).
func (v *Validator) AddFieldError(key, message string) {
	// Note: We need to initialize the map first, if it isn't already
	// initialized.
	if v.FieldErrors == nil {
		v.FieldErrors = make(map[string]string)
	}

	if _, exists := v.FieldErrors[key]; !exists {
		v.FieldErrors[key] = message
	}
}

// NotBlank returns true if a value is not an empty string.
func (v *Validator) NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// NotNil returns true if a value is not nill.
func (v *Validator) NotNil(value any) bool {
	return value != nil
}

// MinChars returns true if a value contains at least n characters.
func (v *Validator) MinChars(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// MaxChars returns true if a value contains less than n characters.
func (v *Validator) MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// In returns true if a specific value is in a list of strings.
func (v *Validator) In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

func ValidateAudioRequest(e echo.Context, m interfaces.ModelsStore) (*interfaces.Title, *interfaces.Voice, *interfaces.Voice, error) {
	// Extract form values
	titleName := e.FormValue("title_name")
	fromVoiceID := e.FormValue("from_voice_id")
	toVoiceID := e.FormValue("to_voice_id")

	fromVoice, err := m.GetVoice(e.Request().Context(), fromVoiceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid from_voice_id: %s", fromVoiceID)
	}

	toVoice, err := m.GetVoice(e.Request().Context(), toVoiceID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid to_voice_id: %s", toVoiceID)
	}

	// Parse and validate numeric parameters
	pause, err := strconv.Atoi(e.FormValue("pause"))
	if err != nil || pause < 3 || pause > 10 {
		return nil, nil, nil, errors.New("pause must be between 3 and 10")
	}

	pattern, err := strconv.Atoi(e.FormValue("pattern"))
	if err != nil || pattern < 1 || pattern > 3 {
		return nil, nil, nil, errors.New("pattern must be between 1 and 3")
	}

	// Validate title name length
	if len(titleName) < 5 || len(titleName) > 32 {
		return nil, nil, nil, errors.New("title_name must be between 5 and 32")
	}

	// Create title object
	title := &interfaces.Title{
		Name:      titleName,
		TitleLang: "",
		FromVoice: fromVoiceID,
		ToVoice:   toVoiceID,
		Pause:     pause,
		Pattern:   pattern,
	}

	return title, &fromVoice, &toVoice, nil
}
