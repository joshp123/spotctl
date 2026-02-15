package spotctl

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

func (c *cli) cmdSearch(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &exitError{code: 2, err: errors.New("missing subcommand for search")}
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "track", "tracks":
		return c.cmdSearchTracks(ctx, args, stdout, stderr)
	default:
		return &exitError{code: 2, err: fmt.Errorf("unknown search subcommand: %s", sub)}
	}
}

func (c *cli) cmdSearchTracks(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("search tracks", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 10, "Max results (<=50)")
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if jsonTrailing {
		*jsonOut = true
	}
	if fs.NArg() < 1 {
		return &exitError{code: 2, err: errors.New("search requires a query")}
	}
	query := strings.Join(fs.Args(), " ")

	if err := c.ensureClient(ctx); err != nil {
		return err
	}

	items, err := c.client.SearchTracks(ctx, query, *limit)
	if err != nil {
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Query string          `json:"query"`
			Items []spotify.Track `json:"items"`
			Limit int             `json:"limit"`
			Count int             `json:"count"`
		}{Query: query, Items: items, Limit: *limit, Count: len(items)})
	}

	if len(items) == 0 {
		fmt.Fprintln(stdout, "(no results)")
		return nil
	}
	for _, t := range items {
		fmt.Fprintf(stdout, "%s — %s (%s)\n", t.Name, t.DisplayArtists(), t.URI)
	}
	return nil
}
