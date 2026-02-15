package spotctl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *exitError) Unwrap() error { return e.err }

func Main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if err := run(ctx, args, stdout, stderr); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			if ee.err != nil {
				fmt.Fprintln(stderr, humanizeError(ee.err).Error())
			}
			return ee.code
		}
		fmt.Fprintln(stderr, humanizeError(err).Error())
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return &exitError{code: 2, err: errors.New("missing command")}
	}

	cmd := args[0]
	args = args[1:]

	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		printUsage(stdout)
		return nil
	}

	cli := newCLI()

	switch cmd {
	case "device":
		return cli.cmdDevice(ctx, args, stdout, stderr)
	case "status":
		return cli.cmdStatus(ctx, args, stdout, stderr)
	case "transfer":
		return cli.cmdTransfer(ctx, args, stdout, stderr)
	case "play":
		return cli.cmdPlay(ctx, args, stdout, stderr)
	case "pause":
		return cli.cmdPause(ctx, args, stdout, stderr)
	case "next":
		return cli.cmdNext(ctx, args, stdout, stderr)
	case "previous", "prev":
		return cli.cmdPrevious(ctx, args, stdout, stderr)
	case "volume":
		return cli.cmdVolume(ctx, args, stdout, stderr)
	case "playlist":
		return cli.cmdPlaylist(ctx, args, stdout, stderr)
	case "search":
		return cli.cmdSearch(ctx, args, stdout, stderr)
	case "auth":
		return cli.cmdAuth(ctx, args, stdout, stderr)
	default:
		printUsage(stderr)
		return &exitError{code: 2, err: fmt.Errorf("unknown command: %s", cmd)}
	}
}

type cli struct {
	client *spotify.Client
	hc     *http.Client
}

func newCLI() *cli {
	return &cli{hc: spotify.DefaultHTTPClient(spotify.DefaultHTTPClientOptions{})}
}

func (c *cli) ensureClient(ctx context.Context) error {
	if c.client != nil {
		return nil
	}
	creds, err := spotify.LoadCredentialsFromEnv()
	if err != nil {
		return err
	}

	accountsBase := strings.TrimSpace(os.Getenv("SPOTIFY_ACCOUNTS_BASE"))
	cachePath := strings.TrimSpace(os.Getenv("SPOTCTL_TOKEN_CACHE"))

	tok, err := spotify.NewTokenManager(creds, spotify.TokenManagerOptions{
		HTTP:         c.hc,
		AccountsBase: accountsBase,
		CachePath:    cachePath,
	})
	if err != nil {
		return err
	}

	apiBase := strings.TrimSpace(os.Getenv("SPOTIFY_API_BASE"))
	c.client = spotify.NewClient(tok, spotify.ClientOptions{HTTP: c.hc, APIBase: apiBase})
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(`spotctl - Spotify Web API CLI (refresh-token OAuth)

Usage:
  spotctl device list [--json]
  spotctl status [--json]
  spotctl transfer --device <name|id>
  spotctl play --device <name|id> <spotify-uri-or-search>
  spotctl search tracks <query> [--limit N] [--json]
  spotctl pause [--device <name|id>]
  spotctl next [--device <name|id>]
  spotctl previous [--device <name|id>]
  spotctl volume [--device <name|id>] <0-100>

  spotctl playlist create --name <name> [--public] [--description <text>] [--json]
  spotctl playlist add --playlist <id|uri|url> <track-uri...> [--json]
  spotctl playlist privacy --playlist <id|uri|url> (--private|--public) [--json]
  spotctl playlist cleanup [--prefix spotctl-test:] [--regex <re>] [--apply --yes] [--json]

  spotctl auth url --redirect-uri <uri>
  spotctl auth exchange --redirect-uri <uri> (--code <code> | --redirect-url <full-url>)
  spotctl auth login --redirect-uri <https://localhost:port/callback>
  spotctl auth bootstrap-agenix --secrets-dir <path> --redirect-uri <https://localhost:port/callback>

Auth env (values or file paths):
  SPOTIFY_CLIENT_ID
  SPOTIFY_CLIENT_SECRET
  SPOTIFY_REFRESH_TOKEN

Notes:
  - Device targeting is strict: if the requested device isn't listed in /me/player/devices,
    Open Spotify on that device, then retry.
`))
}

func formatDevices(devs []spotify.Device) string {
	if len(devs) == 0 {
		return "(no devices)"
	}
	sorted := append([]spotify.Device(nil), devs...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].IsActive != sorted[j].IsActive {
			return sorted[i].IsActive
		}
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	var b strings.Builder
	for _, d := range sorted {
		active := " "
		if d.IsActive {
			active = "*"
		}
		fmt.Fprintf(&b, "%s %s (%s) id=%s vol=%d\n", active, d.Name, d.Type, d.ID, d.VolumePercent)
	}
	return strings.TrimRight(b.String(), "\n")
}

func strictDeviceMessage(selector string, devs []spotify.Device) string {
	msg := fmt.Sprintf("Spotify device not available: %q\nOpen Spotify on that device, then retry.\n", selector)
	if len(devs) == 0 {
		return msg + "No devices reported by Spotify."
	}
	return msg + "Available devices:\n" + formatDevices(devs)
}
