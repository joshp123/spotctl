package spotctl

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/joshp123/spotctl/internal/spotify"
)

func (c *cli) cmdPlaylistPrivacy(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("playlist privacy", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	playlistSel := fs.String("playlist", "", "Playlist id/uri/url")
	makePublic := fs.Bool("public", false, "Make playlist public")
	makePrivate := fs.Bool("private", false, "Make playlist private/secret")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if jsonTrailing {
		*jsonOut = true
	}
	if *playlistSel == "" {
		return &exitError{code: 2, err: errors.New("missing --playlist")}
	}
	if (*makePublic && *makePrivate) || (!*makePublic && !*makePrivate) {
		return &exitError{code: 2, err: errors.New("playlist privacy requires exactly one of --public or --private")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("playlist privacy takes no positional args")}
	}

	pid, err := spotify.NormalizePlaylistID(*playlistSel)
	if err != nil {
		return &exitError{code: 2, err: err}
	}

	pub := *makePublic
	if *makePrivate {
		pub = false
	}
	if err := c.client.UpdatePlaylistDetails(ctx, pid, &pub, nil, nil); err != nil {
		return err
	}

	det, err := c.client.PlaylistDetails(ctx, pid)
	if err != nil {
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(det)
	}
	state := "private"
	if det.Public != nil && *det.Public {
		state = "public"
	}
	fmt.Fprintf(stdout, "Playlist is now %s: %s (%s)\n", state, det.Name, det.URI)
	_ = stderr
	return nil
}
