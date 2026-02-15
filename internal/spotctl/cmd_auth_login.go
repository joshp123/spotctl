package spotctl

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

func (c *cli) cmdAuthLogin(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	redirectURI := fs.String("redirect-uri", "", "Redirect URI (must match Spotify app settings). Recommended: http://localhost:8899/callback")
	scopes := fs.String("scopes", strings.Join(defaultScopes, " "), "Space-separated scopes")
	showDialog := fs.Bool("show-dialog", true, "Force auth dialog")
	noOpen := fs.Bool("no-open", false, "Do not auto-open browser; print URL only")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if *redirectURI == "" {
		return &exitError{code: 2, err: errors.New("missing --redirect-uri")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("auth login takes no positional args")}
	}

	clientID, err := readSecretOrPrompt("SPOTIFY_CLIENT_ID", stderr)
	if err != nil {
		return err
	}
	clientSecret, err := readSecretOrPrompt("SPOTIFY_CLIENT_SECRET", stderr)
	if err != nil {
		return err
	}

	state, err := randomState()
	if err != nil {
		return err
	}

	// Start the callback server before opening the browser.
	fmt.Fprintln(stderr, "Starting local HTTPS callback server (browser will warn about self-signed cert)...")
	cb, err := spotify.StartLocalCallbackServer(*redirectURI)
	if err != nil {
		return err
	}
	defer cb.Close()

	authURL := spotify.AuthorizationURL(clientID, spotify.AuthURLOptions{
		RedirectURI: cb.RedirectURL,
		Scopes:      strings.Fields(*scopes),
		ShowDialog:  *showDialog,
		State:       state,
	})

	fmt.Fprintln(stderr, "Open this URL if your browser didn't open automatically:")
	fmt.Fprintln(stderr, authURL)
	if !*noOpen {
		_ = openURL(ctx, authURL)
	}

	res, err := cb.Wait(ctx)
	if err != nil {
		return err
	}
	if res.Error != "" {
		return fmt.Errorf("spotify auth error: %s", res.Error)
	}
	if res.Code == "" {
		return errors.New("spotify callback missing code")
	}
	if res.State != "" && res.State != state {
		return errors.New("spotify callback state mismatch")
	}

	ex, err := spotify.ExchangeAuthorizationCode(ctx, c.hc, spotify.Credentials{ClientID: clientID, ClientSecret: clientSecret}, res.Code, spotify.AuthCodeExchangeOptions{
		RedirectURI: cb.RedirectURL,
	})
	if err != nil {
		return err
	}

	// stdout only: refresh token
	fmt.Fprintln(stdout, ex.RefreshToken)
	fmt.Fprintln(stderr, "OK. Use this as SPOTIFY_REFRESH_TOKEN (value or file).")
	return nil
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func readSecretOrPrompt(key string, stderr io.Writer) (string, error) {
	// If the env isn't set at all, prompt.
	if os.Getenv(key) == "" && os.Getenv(key+"_FILE") == "" {
		fmt.Fprintf(stderr, "%s: ", key)
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		v := strings.TrimSpace(line)
		if v == "" {
			return "", fmt.Errorf("missing %s", key)
		}
		return v, nil
	}
	return spotify.ReadSecretEnvOrFile(key)
}
