package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ClientOptions struct {
	HTTP      *http.Client
	APIBase   string
	UserAgent string
}

type Client struct {
	tok *TokenManager
	hc  *http.Client

	apiBase   string
	userAgent string
}

func NewClient(tok *TokenManager, opt ClientOptions) *Client {
	base := strings.TrimRight(opt.APIBase, "/")
	if base == "" {
		base = "https://api.spotify.com"
	}
	ua := opt.UserAgent
	if ua == "" {
		ua = "spotctl/0.1"
	}
	hc := opt.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{tok: tok, hc: hc, apiBase: base, userAgent: ua}
}

type DefaultHTTPClientOptions struct {
	Timeout time.Duration
}

func DefaultHTTPClient(opt DefaultHTTPClientOptions) *http.Client {
	to := opt.Timeout
	if to == 0 {
		to = 15 * time.Second
	}
	return &http.Client{Timeout: to}
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Me(ctx)
	return err
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, body any, out any, expectedStatus ...int) error {
	if len(expectedStatus) == 0 {
		expectedStatus = []int{200}
	}

	u := c.apiBase + path
	if q != nil && len(q) > 0 {
		u += "?" + q.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		bb, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = bb
	}

	debug := os.Getenv("SPOTCTL_DEBUG") != "" || os.Getenv("SPOTCTL_DEBUG_HTTP") != ""
	try := func(forceRefresh bool) (*http.Response, []byte, error) {
		var token string
		var err error
		if forceRefresh {
			token, err = c.tok.ForceRefresh(ctx)
		} else {
			token, err = c.tok.AccessToken(ctx)
		}
		if err != nil {
			return nil, nil, err
		}

		var r io.Reader
		if bodyBytes != nil {
			r = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, r)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", c.userAgent)
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			return nil, nil, err
		}
		bb, _ := ioReadAllLimit(resp.Body, 2<<20)
		_ = resp.Body.Close()
		if debug {
			log.Printf("spotctl http: %s %s -> %d", method, u, resp.StatusCode)
		}
		return resp, bb, nil
	}

	// Retry policy:
	// - 401 -> refresh and retry once
	// - 429 -> wait Retry-After and retry once
	// - 5xx -> small backoff and retry twice
	resp, bb, err := try(false)
	if err != nil {
		return err
	}
	if resp.StatusCode == 401 {
		resp, bb, err = try(true)
		if err != nil {
			return err
		}
	}
	if resp.StatusCode == 429 {
		ra := resp.Header.Get("Retry-After")
		if ra != "" {
			if secs, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil && secs > 0 {
				maxWait := 15
				if s := strings.TrimSpace(os.Getenv("SPOTCTL_MAX_RETRY_AFTER_SECS")); s != "" {
					if v, err := strconv.Atoi(s); err == nil && v > 0 {
						maxWait = v
					}
				}
				if secs > maxWait {
					return &APIError{StatusCode: 429, Message: fmt.Sprintf("rate limited (Retry-After=%ds); wait then retry", secs), Body: strings.TrimSpace(string(bb))}
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(secs) * time.Second):
				}
				resp, bb, err = try(false)
				if err != nil {
					return err
				}
			}
		}
	}
	for attempt := 0; attempt < 2 && resp.StatusCode >= 500 && resp.StatusCode <= 599; attempt++ {
		d := time.Duration(attempt+1) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
		resp, bb, err = try(false)
		if err != nil {
			return err
		}
	}

	ok := false
	for _, s := range expectedStatus {
		if resp.StatusCode == s {
			ok = true
			break
		}
	}
	if !ok {
		return decodeAPIError(resp.StatusCode, resp.Status, bb)
	}

	if out != nil {
		// Some endpoints legitimately return 204 No Content.
		// Callers can include 204 in expectedStatus and then interpret an empty body.
		if len(bb) == 0 {
			return nil
		}
		if err := json.Unmarshal(bb, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func decodeAPIError(code int, status string, body []byte) error {
	// Spotify: {"error":{"status":401,"message":"..."}}
	var e struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &e); err == nil {
		if e.Error.Message != "" {
			return &APIError{StatusCode: code, Message: e.Error.Message, Body: strings.TrimSpace(string(body))}
		}
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = status
	}
	return &APIError{StatusCode: code, Message: msg, Body: msg}
}

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	lr := &io.LimitedReader{R: r, N: limit + 1}
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("response too large")
	}
	return b, nil
}
