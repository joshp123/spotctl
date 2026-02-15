package spotctl

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

type cleanupMatch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type cleanupResult struct {
	Prefix       string         `json:"prefix,omitempty"`
	Regex        string         `json:"regex,omitempty"`
	Apply        bool           `json:"apply"`
	Matched      []cleanupMatch `json:"matched"`
	DeletedIDs   []string       `json:"deleted_ids,omitempty"`
	SkippedIDs   []string       `json:"skipped_ids,omitempty"`
	ErrorDeleted []string       `json:"error_deleted,omitempty"`
}

func (c *cli) cmdPlaylistCleanup(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("playlist cleanup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	prefix := fs.String("prefix", "spotctl-test:", "Only match playlists whose name starts with this prefix")
	re := fs.String("regex", "", "Match playlist names by regex (in addition to prefix)")
	apply := fs.Bool("apply", false, "Actually delete/unfollow matched playlists")
	yes := fs.Bool("yes", false, "Confirm deletion (required with --apply)")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if jsonTrailing {
		*jsonOut = true
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("playlist cleanup takes no positional args")}
	}
	if *apply && !*yes {
		return &exitError{code: 2, err: errors.New("refusing to delete without --yes (use --apply --yes)")}
	}

	var rx *regexp.Regexp
	if strings.TrimSpace(*re) != "" {
		var err error
		rx, err = regexp.Compile(*re)
		if err != nil {
			return &exitError{code: 2, err: fmt.Errorf("invalid --regex: %w", err)}
		}
	}

	pls, err := c.client.MyPlaylists(ctx)
	if err != nil {
		return err
	}

	res := cleanupResult{Prefix: *prefix, Regex: *re, Apply: *apply}
	for _, pl := range pls {
		match := false
		if *prefix != "" && strings.HasPrefix(pl.Name, *prefix) {
			match = true
		}
		if rx != nil && rx.MatchString(pl.Name) {
			match = true
		}
		if !match {
			continue
		}
		res.Matched = append(res.Matched, cleanupMatch{ID: pl.ID, Name: pl.Name, URI: pl.URI})
	}

	if *apply {
		for _, m := range res.Matched {
			pid, err := spotify.NormalizePlaylistID(m.ID)
			if err != nil {
				res.SkippedIDs = append(res.SkippedIDs, m.ID)
				continue
			}
			if err := c.client.UnfollowPlaylist(ctx, pid); err != nil {
				res.ErrorDeleted = append(res.ErrorDeleted, m.ID)
				// keep going
				continue
			}
			res.DeletedIDs = append(res.DeletedIDs, m.ID)
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	if len(res.Matched) == 0 {
		fmt.Fprintln(stdout, "No playlists matched.")
		return nil
	}

	if !*apply {
		fmt.Fprintf(stdout, "Matched %d playlist(s). Re-run with --apply --yes to delete.\n", len(res.Matched))
		for _, m := range res.Matched {
			fmt.Fprintf(stdout, "- %s :: %s\n", m.ID, m.Name)
		}
		return nil
	}

	fmt.Fprintf(stdout, "Deleted %d playlist(s).\n", len(res.DeletedIDs))
	if len(res.ErrorDeleted) > 0 {
		fmt.Fprintf(stderr, "WARN: failed to delete %d playlist(s) (rate limit or permissions)\n", len(res.ErrorDeleted))
	}
	return nil
}
