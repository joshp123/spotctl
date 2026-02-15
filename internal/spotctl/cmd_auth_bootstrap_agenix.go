package spotctl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

// bootstrap-agenix:
// - prompts for client id/secret
// - runs browser auth with local callback
// - writes 3 agenix secrets under --secrets-dir (no token copy/paste)
func (c *cli) cmdAuthBootstrapAgenix(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("auth bootstrap-agenix", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	secretsDir := fs.String("secrets-dir", "", "Path to nix-secrets directory (contains secrets.nix)")
	redirectURI := fs.String("redirect-uri", "", "Redirect URI (must match Spotify app settings). Recommended: http://localhost:8899/callback")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if *secretsDir == "" {
		return &exitError{code: 2, err: errors.New("missing --secrets-dir")}
	}
	if *redirectURI == "" {
		return &exitError{code: 2, err: errors.New("missing --redirect-uri")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("auth bootstrap-agenix takes no positional args")}
	}

	if _, err := exec.LookPath("agenix"); err != nil {
		return errors.New("agenix not found on PATH")
	}

	clientID, err := readSecretOrPrompt("SPOTIFY_CLIENT_ID", stderr)
	if err != nil {
		return err
	}
	clientSecret, err := readSecretOrPrompt("SPOTIFY_CLIENT_SECRET", stderr)
	if err != nil {
		return err
	}

	sdir := expandPath(*secretsDir)
	if _, err := os.Stat(filepath.Join(sdir, "secrets.nix")); err != nil {
		return fmt.Errorf("secrets.nix not found in %s", sdir)
	}

	fmt.Fprintln(stderr, "Writing agenix secrets (client id/secret)...")
	if err := writeAgenixSecret(ctx, sdir, "spotify-client-id.age", clientID+"\n"); err != nil {
		return err
	}
	if err := writeAgenixSecret(ctx, sdir, "spotify-client-secret.age", clientSecret+"\n"); err != nil {
		return err
	}

	// Now mint refresh token via login flow.
	state, err := randomState()
	if err != nil {
		return err
	}
	pkce, err := spotify.NewPKCE()
	if err != nil {
		return err
	}

	fmt.Fprintln(stderr, "Starting local HTTPS callback server (browser will warn about self-signed cert)...")
	cb, err := spotify.StartLocalCallbackServer(*redirectURI)
	if err != nil {
		return err
	}
	defer cb.Close()

	authURL := spotify.AuthorizationURL(clientID, spotify.AuthURLOptions{
		RedirectURI:         cb.RedirectURL,
		Scopes:              defaultScopes,
		ShowDialog:          true,
		State:               state,
		CodeChallenge:       pkce.Challenge,
		CodeChallengeMethod: "S256",
	})
	fmt.Fprintln(stderr, authURL)
	_ = openURL(ctx, authURL)

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
		RedirectURI:  cb.RedirectURL,
		CodeVerifier: pkce.Verifier,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(stderr, "Writing agenix secret (refresh token)...")
	if err := writeAgenixSecret(ctx, sdir, "spotify-refresh-token.age", ex.RefreshToken+"\n"); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "OK")
	return nil
}

func writeAgenixSecret(ctx context.Context, dir, file, contents string) error {
	cmd := exec.CommandContext(ctx, "agenix", "-e", file)
	cmd.Dir = dir
	// agenix calls $EDITOR with the target file path appended.
	// Use $0 (not $1) because bash -c sets $0 to the first arg after the script.
	cmd.Env = append(os.Environ(), "EDITOR=bash -c 'cat > \"$0\"'")
	cmd.Stdin = strings.NewReader(contents)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		h, _ := os.UserHomeDir()
		if p == "~" {
			return h
		}
		if strings.HasPrefix(p, "~/") {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}
