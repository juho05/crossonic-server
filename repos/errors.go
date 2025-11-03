package repos

import (
	"errors"
	"fmt"
)

type ErrorType string

func (e ErrorType) Error() string {
	return string(e)
}

const (
	ErrNotFound          ErrorType = "not found"
	ErrExists            ErrorType = "already exists"
	ErrTooMany           ErrorType = "got multiple rows, expected one"
	ErrNestedTransaction ErrorType = "nested transactions not allowed"
	ErrGeneral           ErrorType = "general"
	ErrInvalidParams     ErrorType = "invalid parameters"
)

func NewError(message string, errType ErrorType, innerErr error) Error {
	return Error{
		Message: message,
		Type:    errType,
		DBErr:   innerErr,
	}
}

type Error struct {
	Message string
	Type    ErrorType
	DBErr   error
}

func (e Error) Error() string {
	if e.DBErr == nil {
		return e.Message
	}
	if e.Message == "" {
		return e.DBErr.Error()
	}
	return fmt.Sprintf("%s: %s", e.Message, e.DBErr)
}

func (e Error) Is(target error) bool {
	var t1 Error
	if errors.As(target, &t1) {
		return e.Type == t1.Type
	}
	var t2 ErrorType
	if errors.As(target, &t2) {
		return e.Type == t2
	}
	return false
}

func (e Error) Unwrap() error {
	return e.DBErr
}

var (
	ErrInvalidDate = errors.New("invalid date")
)
