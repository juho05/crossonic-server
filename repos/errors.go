package repos

import "fmt"

type ErrorType string

func (e ErrorType) Error() string {
	return string(e)
}

const (
	ErrNotFound          ErrorType = "not found"
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
	if t, ok := target.(Error); ok {
		return e.Type == t.Type
	}
	if t, ok := target.(ErrorType); ok {
		return e.Type == t
	}
	return false
}

func (e Error) Unwrap() error {
	return e.DBErr
}
