package spotify

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTokenManagerRefresh(t *testing.T) {
	var gotAuth string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(500)
			return
		}
		if r.URL.Path != "/api/token" {
			w.WriteHeader(404)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		b, _ := ioReadAllLimit(r.Body, 1<<20)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at","token_type":"Bearer","expires_in":3600}`)
	}))
	defer srv.Close()

	creds := Credentials{ClientID: "cid", ClientSecret: "sec", RefreshToken: "rt"}
	m, err := NewTokenManager(creds, TokenManagerOptions{HTTP: srv.Client(), AccountsBase: srv.URL, Now: func() time.Time { return time.Unix(0, 0) }})
	if err != nil {
		t.Fatal(err)
	}

	tok, err := m.AccessToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "at" {
		t.Fatalf("tok=%q", tok)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("cid:sec"))
	if gotAuth != wantAuth {
		t.Fatalf("auth=%q want=%q", gotAuth, wantAuth)
	}

	vals, _ := url.ParseQuery(strings.TrimSpace(gotBody))
	if vals.Get("grant_type") != "refresh_token" {
		t.Fatalf("grant_type=%q", vals.Get("grant_type"))
	}
	if vals.Get("refresh_token") != "rt" {
		t.Fatalf("refresh_token=%q", vals.Get("refresh_token"))
	}
}
