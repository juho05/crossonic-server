package listenbrainz

import "errors"

var (
	ErrUnexpectedResponseCode = errors.New("unexpected response code")
	ErrUnexpectedResponseBody = errors.New("unexpected response body")
	ErrUnauthenticated        = errors.New("unauthenticated")
	ErrDisabled               = errors.New("feature disabled")
)
