package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type AuthCodeExchangeResult struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type AuthURLOptions struct {
	RedirectURI         string
	Scopes              []string
	ShowDialog          bool
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

func AuthorizationURL(clientID string, opt AuthURLOptions) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", opt.RedirectURI)
	if len(opt.Scopes) > 0 {
		q.Set("scope", strings.Join(opt.Scopes, " "))
	}
	if opt.ShowDialog {
		q.Set("show_dialog", "true")
	}
	if opt.State != "" {
		q.Set("state", opt.State)
	}
	if opt.CodeChallenge != "" {
		q.Set("code_challenge", opt.CodeChallenge)
		m := opt.CodeChallengeMethod
		if m == "" {
			m = "S256"
		}
		q.Set("code_challenge_method", m)
	}
	return "https://accounts.spotify.com/authorize?" + q.Encode()
}

type AuthCodeExchangeOptions struct {
	RedirectURI  string
	CodeVerifier string // optional (PKCE)
}

func ExchangeAuthorizationCode(ctx context.Context, hc *http.Client, creds Credentials, code string, opt AuthCodeExchangeOptions) (AuthCodeExchangeResult, error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", opt.RedirectURI)
	if opt.CodeVerifier != "" {
		form.Set("code_verifier", opt.CodeVerifier)
	}
	if creds.ClientSecret == "" {
		// PKCE / public client
		form.Set("client_id", creds.ClientID)
	}

	endpoint := "https://accounts.spotify.com/api/token"
	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return AuthCodeExchangeResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if creds.ClientSecret != "" {
		req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return AuthCodeExchangeResult{}, err
	}
	defer resp.Body.Close()
	b, _ := ioReadAllLimit(resp.Body, 1<<20)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return AuthCodeExchangeResult{}, fmt.Errorf("spotify token exchange failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var res AuthCodeExchangeResult
	if err := json.Unmarshal(b, &res); err != nil {
		return AuthCodeExchangeResult{}, err
	}
	if res.RefreshToken == "" {
		return AuthCodeExchangeResult{}, errors.New("exchange response missing refresh_token")
	}
	return res, nil
}
