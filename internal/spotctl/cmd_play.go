package spotctl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/joshp123/spotctl/internal/spotify"
)

func (c *cli) cmdPlay(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	deviceSel := fs.String("device", "", "Device name or id (strict)")
	if err := fs.Parse(args); err != nil {
		return &exitError{code: 2, err: err}
	}
	if *deviceSel == "" {
		return &exitError{code: 2, err: errors.New("missing --device")}
	}
	if fs.NArg() != 1 {
		return &exitError{code: 2, err: errors.New("play requires exactly one argument: spotify URI or search query")}
	}
	q := fs.Arg(0)

	dev, devs, err := c.client.ResolveDevice(ctx, *deviceSel)
	if err != nil {
		return err
	}
	if dev == nil {
		return &exitError{code: 3, err: errors.New(strictDeviceMessage(*deviceSel, devs))}
	}

	uri, kind, err := spotify.NormalizeURI(q)
	if err != nil {
		return &exitError{code: 2, err: err}
	}

	if kind == spotify.URIKindUnknown {
		track, err := c.client.SearchTopTrack(ctx, q)
		if err != nil {
			return err
		}
		uri = track.URI
		kind = spotify.URIKindTrack
		fmt.Fprintf(stderr, "Search: %q -> %s — %s (%s)\n", q, track.Name, track.DisplayArtists(), track.URI)
	}

	var req spotify.PlayRequest
	switch kind {
	case spotify.URIKindTrack, spotify.URIKindEpisode:
		req = spotify.PlayRequest{URIs: []string{uri}}
	case spotify.URIKindAlbum, spotify.URIKindPlaylist, spotify.URIKindArtist, spotify.URIKindShow:
		req = spotify.PlayRequest{ContextURI: uri}
	default:
		return &exitError{code: 2, err: fmt.Errorf("unsupported URI kind for play: %s (%s)", kind, uri)}
	}

	if err := c.client.Play(ctx, dev.ID, req); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Play on %s: %s\n", dev.Name, uri)
	return nil
}
