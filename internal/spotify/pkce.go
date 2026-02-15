package spotify

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCE helpers (RFC 7636). Spotify supports PKCE for public clients.
//
// We don't require PKCE for server-side use (client_secret is fine),
// but it's convenient for local bootstrap flows.

type PKCE struct {
	Verifier  string
	Challenge string
}

func NewPKCE() (PKCE, error) {
	verifier, err := randomBase64URL(64)
	if err != nil {
		return PKCE{}, err
	}
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	return PKCE{Verifier: verifier, Challenge: challenge}, nil
}

func randomBase64URL(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
