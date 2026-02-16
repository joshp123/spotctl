package spotctl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

type addQueryResolved struct {
	Query string         `json:"query"`
	Track *spotify.Track `json:"track,omitempty"`
	Error string         `json:"error,omitempty"`
}

type addQueryResult struct {
	Playlist   string             `json:"playlist"`
	TSV        bool               `json:"tsv"`
	Resolved   []addQueryResolved `json:"resolved"`
	AddedURIs  []string           `json:"added_uris"`
	SnapshotID string             `json:"snapshot_id,omitempty"`
	Misses     int                `json:"misses"`
	Added      int                `json:"added"`
	Total      int                `json:"total"`
}

func (c *cli) cmdPlaylistAddQuery(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("playlist add-query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	playlistSel := fs.String("playlist", "", "Playlist id/uri/url")
	fromStdin := fs.Bool("stdin", false, "Read queries from stdin (one per line)")
	tsv := fs.Bool("tsv", false, "Parse stdin as TSV: <artist>\\t<track>")
	limit := fs.Int("limit", 3, "Spotify search limit per query (<=50)")
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

	queries := []string{}
	if *fromStdin {
		stdinQueries, err := readQueriesFromStdin(os.Stdin, *tsv)
		if err != nil {
			return &exitError{code: 2, err: err}
		}
		queries = append(queries, stdinQueries...)
	} else {
		if fs.NArg() == 0 {
			return &exitError{code: 2, err: errors.New("playlist add-query requires at least one query (or pass --stdin)")}
		}
		queries = append(queries, fs.Args()...)
	}
	if len(queries) == 0 {
		return &exitError{code: 2, err: errors.New("no queries provided")}
	}

	pid, err := spotify.NormalizePlaylistID(*playlistSel)
	if err != nil {
		return &exitError{code: 2, err: err}
	}

	if err := c.ensureClient(ctx); err != nil {
		return err
	}

	res := addQueryResult{Playlist: pid, TSV: *tsv, Total: len(queries)}

	uris := []string{}
	ids := []string{}
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}

		items, err := c.client.SearchTracks(ctx, q, *limit)
		if err != nil {
			res.Resolved = append(res.Resolved, addQueryResolved{Query: q, Error: err.Error()})
			continue
		}
		if len(items) == 0 {
			res.Resolved = append(res.Resolved, addQueryResolved{Query: q, Error: "no results"})
			continue
		}

		picked := items[0]
		res.Resolved = append(res.Resolved, addQueryResolved{Query: q, Track: &picked})

		uri, kind, err := spotify.NormalizeURI(picked.URI)
		if err != nil {
			res.Resolved[len(res.Resolved)-1].Error = err.Error()
			res.Resolved[len(res.Resolved)-1].Track = nil
			continue
		}
		if kind != spotify.URIKindTrack {
			res.Resolved[len(res.Resolved)-1].Error = fmt.Sprintf("non-track uri: %s", picked.URI)
			res.Resolved[len(res.Resolved)-1].Track = nil
			continue
		}
		id, err := spotify.TrackIDFromURI(uri)
		if err != nil {
			res.Resolved[len(res.Resolved)-1].Error = err.Error()
			res.Resolved[len(res.Resolved)-1].Track = nil
			continue
		}
		uris = append(uris, uri)
		ids = append(ids, id)
	}

	// Validate tracks exist to prevent hallucinated IDs.
	validURIs := []string{}
	for i, id := range ids {
		t, err := c.client.GetTrack(ctx, id)
		if err != nil {
			continue
		}
		if t.ID == "" {
			continue
		}
		validURIs = append(validURIs, uris[i])
	}

	res.AddedURIs = validURIs
	res.Misses = countMisses(res.Resolved)
	res.Added = len(validURIs)

	if len(validURIs) > 0 {
		addRes, err := c.client.AddTracksToPlaylist(ctx, pid, validURIs)
		if err != nil {
			return err
		}
		res.SnapshotID = addRes.SnapshotID
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	fmt.Fprintf(stdout, "Added %d/%d track(s).\n", res.Added, res.Total)
	if res.Misses > 0 {
		fmt.Fprintf(stderr, "WARN: %d query(ies) had no results or errors. Re-run with --json for details.\n", res.Misses)
	}
	return nil
}

func countMisses(rs []addQueryResolved) int {
	m := 0
	for _, r := range rs {
		if r.Track == nil {
			m++
		}
	}
	return m
}

func readQueriesFromStdin(stdin io.Reader, tsv bool) ([]string, error) {
	s := bufio.NewScanner(stdin)
	out := []string{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if !tsv {
			out = append(out, line)
			continue
		}
		artist, track, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("expected TSV line: <artist>\\t<track>, got: %q", line)
		}
		artist = strings.TrimSpace(artist)
		track = strings.TrimSpace(track)
		if artist == "" || track == "" {
			return nil, fmt.Errorf("invalid TSV line (empty field): %q", line)
		}
		// Fielded Spotify query is much less ambiguous.
		q := fmt.Sprintf("track:%q artist:%q", track, artist)
		out = append(out, q)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("stdin provided no queries")
	}
	return out, nil
}
