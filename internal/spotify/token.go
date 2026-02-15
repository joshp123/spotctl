package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type TokenManagerOptions struct {
	HTTP         *http.Client
	AccountsBase string
	CachePath    string // optional; empty => no cache
	Now          func() time.Time
}

type TokenManager struct {
	creds Credentials
	hc    *http.Client
	base  string
	now   func() time.Time

	cachePath string

	mu   sync.Mutex
	cur  Token
	have bool
}

func NewTokenManager(creds Credentials, opt TokenManagerOptions) (*TokenManager, error) {
	base := strings.TrimRight(opt.AccountsBase, "/")
	if base == "" {
		base = "https://accounts.spotify.com"
	}
	m := &TokenManager{
		creds:     creds,
		hc:        opt.HTTP,
		base:      base,
		cachePath: opt.CachePath,
		now:       opt.Now,
	}
	if m.hc == nil {
		m.hc = http.DefaultClient
	}
	if m.now == nil {
		m.now = time.Now
	}
	if m.cachePath != "" {
		_ = m.loadCache()
	}
	return m, nil
}

func (m *TokenManager) AccessToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.have && m.cur.AccessToken != "" {
		// Refresh a bit early.
		if m.now().Add(30 * time.Second).Before(m.cur.ExpiresAt) {
			return m.cur.AccessToken, nil
		}
	}

	tok, err := m.refresh(ctx)
	if err != nil {
		return "", err
	}
	m.cur = tok
	m.have = true
	if m.cachePath != "" {
		_ = m.saveCache(tok)
	}
	return tok.AccessToken, nil
}

func (m *TokenManager) ForceRefresh(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, err := m.refresh(ctx)
	if err != nil {
		return "", err
	}
	m.cur = tok
	m.have = true
	if m.cachePath != "" {
		_ = m.saveCache(tok)
	}
	return tok.AccessToken, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope,omitempty"`
	ExpiresIn   int    `json:"expires_in"`
}

func (m *TokenManager) refresh(ctx context.Context) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", m.creds.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.base+"/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(m.creds.ClientID, m.creds.ClientSecret)

	resp, err := m.hc.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()

	b, _ := ioReadAllLimit(resp.Body, 1<<20)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if resp.StatusCode == 400 {
			// Spotify often returns JSON {error, error_description}.
			var e struct {
				Error            string `json:"error"`
				ErrorDescription string `json:"error_description"`
			}
			_ = json.Unmarshal(b, &e)
			if e.ErrorDescription != "" {
				return Token{}, fmt.Errorf("spotify token refresh failed: %s", e.ErrorDescription)
			}
		}
		return Token{}, fmt.Errorf("spotify token refresh failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var tr tokenResponse
	if err := json.Unmarshal(b, &tr); err != nil {
		return Token{}, fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return Token{}, errors.New("token response missing access_token")
	}

	exp := m.now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return Token{AccessToken: tr.AccessToken, TokenType: tr.TokenType, Scope: tr.Scope, ExpiresAt: exp}, nil
}

func (m *TokenManager) loadCache() error {
	b, err := os.ReadFile(m.cachePath)
	if err != nil {
		return err
	}
	var tok Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return err
	}
	if tok.AccessToken == "" || tok.ExpiresAt.IsZero() {
		return errors.New("invalid token cache")
	}
	m.cur = tok
	m.have = true
	return nil
}

func (m *TokenManager) saveCache(tok Token) error {
	if m.cachePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.cachePath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.cachePath, b, 0o600)
}
