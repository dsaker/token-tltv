package services

import (
	"errors"
	"fmt"
)

var (
	ErrOneFile           = errors.New("no need to zip one file")
	ErrUnableToParseFile = func(err error) error {

		return fmt.Errorf("unable to parsefile file: %s", err)
	}
)

type FileTooLargeError struct {
	Size  int64
	Limit int64
}

func (f FileTooLargeError) Error() string {
	return fmt.Sprintf("file too large (%d > %d)", f.Size, f.Limit)
}

func NewFileTooLargeError(size, limit int64) error {
	return FileTooLargeError{Size: size, Limit: limit}
}

func ErrFileTooLarge(size int64, limit int64) error {
	return NewFileTooLargeError(size, limit)
}

func IsFileTooLargeError(err error) bool {
	var fileTooLargeError FileTooLargeError
	ok := errors.As(err, &fileTooLargeError)
	return ok
}
