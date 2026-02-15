package spotify

import "fmt"

// APIError is a structured error returned by Spotify Web API.
//
// Most endpoints respond with:
//
//	{"error":{"status":403,"message":"..."}}
//
// We keep the original HTTP status code plus the parsed message.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Body
	}
	if msg == "" {
		msg = "spotify api error"
	}
	return fmt.Sprintf("spotify api error (%d): %s", e.StatusCode, msg)
}
