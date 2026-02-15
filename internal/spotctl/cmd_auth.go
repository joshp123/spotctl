package spotctl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

var defaultScopes = []string{
	"user-read-playback-state",
	"user-modify-playback-state",
	"user-read-currently-playing",
	"playlist-read-private",
	"playlist-modify-private",
	"playlist-modify-public",
	"user-read-private",
}

func (c *cli) cmdAuth(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &exitError{code: 2, err: errors.New("missing subcommand for auth")}
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "url":
		return c.cmdAuthURL(ctx, args, stdout, stderr)
	case "exchange":
		return c.cmdAuthExchange(ctx, args, stdout, stderr)
	case "login":
		return c.cmdAuthLogin(ctx, args, stdout, stderr)
	case "bootstrap-agenix":
		return c.cmdAuthBootstrapAgenix(ctx, args, stdout, stderr)
	default:
		return &exitError{code: 2, err: fmt.Errorf("unknown auth subcommand: %s", sub)}
	}
}

func (c *cli) cmdAuthURL(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("auth url", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	redirectURI := fs.String("redirect-uri", "", "Redirect URI (must match Spotify app settings)")
	scopes := fs.String("scopes", strings.Join(defaultScopes, " "), "Space-separated scopes")
	showDialog := fs.Bool("show-dialog", true, "Force auth dialog")
	if err := fs.Parse(args); err != nil {
		return &exitError{code: 2, err: err}
	}
	if *redirectURI == "" {
		return &exitError{code: 2, err: errors.New("missing --redirect-uri")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("auth url takes no positional args")}
	}

	clientID, err := spotify.ReadSecretEnvOrFile("SPOTIFY_CLIENT_ID")
	if err != nil {
		return err
	}

	u := spotify.AuthorizationURL(clientID, spotify.AuthURLOptions{
		RedirectURI: *redirectURI,
		Scopes:      strings.Fields(*scopes),
		ShowDialog:  *showDialog,
	})
	fmt.Fprintln(stdout, u)
	return nil
}

func (c *cli) cmdAuthExchange(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("auth exchange", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	redirectURI := fs.String("redirect-uri", "", "Redirect URI")
	code := fs.String("code", "", "Authorization code")
	redirectURL := fs.String("redirect-url", "", "Full redirect URL (paste from browser address bar)")
	if err := fs.Parse(args); err != nil {
		return &exitError{code: 2, err: err}
	}
	if *redirectURI == "" {
		return &exitError{code: 2, err: errors.New("missing --redirect-uri")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("auth exchange takes no positional args")}
	}

	finalCode := strings.TrimSpace(*code)
	if finalCode == "" {
		if *redirectURL == "" {
			return &exitError{code: 2, err: errors.New("must supply --code or --redirect-url")}
		}
		u, err := url.Parse(*redirectURL)
		if err != nil {
			return &exitError{code: 2, err: fmt.Errorf("invalid --redirect-url: %w", err)}
		}
		finalCode = u.Query().Get("code")
		if finalCode == "" {
			return &exitError{code: 2, err: errors.New("redirect URL missing code=...")}
		}
	}

	clientID, err := spotify.ReadSecretEnvOrFile("SPOTIFY_CLIENT_ID")
	if err != nil {
		return err
	}
	clientSecret, err := spotify.ReadSecretEnvOrFile("SPOTIFY_CLIENT_SECRET")
	if err != nil {
		return err
	}

	res, err := spotify.ExchangeAuthorizationCode(ctx, c.hc, spotify.Credentials{ClientID: clientID, ClientSecret: clientSecret}, finalCode, spotify.AuthCodeExchangeOptions{RedirectURI: *redirectURI})
	if err != nil {
		return err
	}

	// stdout: refresh token only (easy copy/paste).
	fmt.Fprintln(stdout, res.RefreshToken)
	fmt.Fprintln(stderr, "Store this as SPOTIFY_REFRESH_TOKEN (value or file).")
	return nil
}
