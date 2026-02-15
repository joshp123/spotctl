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

func (c *cli) cmdPlaylist(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	if len(args) == 0 {
		return &exitError{code: 2, err: errors.New("missing subcommand for playlist")}
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "create":
		return c.cmdPlaylistCreate(ctx, args, stdout, stderr)
	case "add":
		return c.cmdPlaylistAdd(ctx, args, stdout, stderr)
	default:
		return &exitError{code: 2, err: fmt.Errorf("unknown playlist subcommand: %s", sub)}
	}
}

func (c *cli) cmdPlaylistCreate(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("playlist create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "Playlist name")
	public := fs.Bool("public", false, "Create as public")
	desc := fs.String("description", "", "Playlist description")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return &exitError{code: 2, err: err}
	}
	if jsonTrailing {
		*jsonOut = true
	}
	if *name == "" {
		return &exitError{code: 2, err: errors.New("missing --name")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("playlist create takes no positional args")}
	}

	pl, err := c.client.CreatePlaylist(ctx, *name, *public, *desc)
	if err != nil {
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(pl)
	}
	fmt.Fprintf(stdout, "Created playlist: %s (%s)\n", pl.Name, pl.URI)
	return nil
}

func (c *cli) cmdPlaylistAdd(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("playlist add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	playlistSel := fs.String("playlist", "", "Playlist id/uri/url")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return &exitError{code: 2, err: err}
	}
	if jsonTrailing {
		*jsonOut = true
	}
	if *playlistSel == "" {
		return &exitError{code: 2, err: errors.New("missing --playlist")}
	}
	if fs.NArg() == 0 {
		return &exitError{code: 2, err: errors.New("playlist add requires at least one track URI")}
	}

	pid, err := spotify.NormalizePlaylistID(*playlistSel)
	if err != nil {
		return &exitError{code: 2, err: err}
	}

	uris := make([]string, 0, fs.NArg())
	ids := make([]string, 0, fs.NArg())
	for _, a := range fs.Args() {
		uri, kind, err := spotify.NormalizeURI(a)
		if err != nil {
			return &exitError{code: 2, err: err}
		}
		if kind != spotify.URIKindTrack {
			return &exitError{code: 2, err: fmt.Errorf("playlist add only supports track URIs in v1: %s", a)}
		}
		id, err := spotify.TrackIDFromURI(uri)
		if err != nil {
			return &exitError{code: 2, err: err}
		}
		uris = append(uris, uri)
		ids = append(ids, id)
	}

	// Validate tracks exist to prevent hallucinated URIs.
	for i, id := range ids {
		t, err := c.client.GetTrack(ctx, id)
		if err != nil {
			return err
		}
		if t.ID == "" {
			return &exitError{code: 2, err: fmt.Errorf("invalid track uri (not found): %s", uris[i])}
		}
	}

	res, err := c.client.AddTracksToPlaylist(ctx, pid, uris)
	if err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	fmt.Fprintf(stdout, "Added %d track(s). Snapshot: %s\n", len(uris), res.SnapshotID)
	return nil
}
