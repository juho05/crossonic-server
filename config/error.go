package config

import "fmt"

type Error struct {
	Key     string
	Message string
}

func newError(key, message string) Error {
	return Error{
		Key:     key,
		Message: message,
	}
}

func (c Error) Error() string {
	return fmt.Sprintf("config: %s: %s", c.Key, c.Message)
}
