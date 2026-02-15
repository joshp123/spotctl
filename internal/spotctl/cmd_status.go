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

type statusOutput struct {
	Active    bool            `json:"active"`
	IsPlaying bool            `json:"is_playing"`
	Progress  int             `json:"progress_ms,omitempty"`
	Device    *spotify.Device `json:"device,omitempty"`
	Item      *spotify.Track  `json:"item,omitempty"`
}

func (c *cli) cmdStatus(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if jsonTrailing {
		*jsonOut = true
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("status takes no positional args")}
	}

	if err := c.ensureClient(ctx); err != nil {
		return err
	}

	st, err := c.client.PlaybackState(ctx)
	if err != nil {
		return err
	}

	out := statusOutput{}
	if st != nil {
		out.Active = true
		out.IsPlaying = st.IsPlaying
		out.Progress = st.ProgressMs
		out.Device = &st.Device
		if st.Item.URI != "" {
			item := st.Item
			out.Item = &item
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if st == nil {
		fmt.Fprintln(stdout, "No active playback.")
		return nil
	}

	state := "paused"
	if st.IsPlaying {
		state = "playing"
	}
	if out.Item != nil {
		fmt.Fprintf(stdout, "%s on %s: %s — %s\n", stringsTitle(state), st.Device.Name, out.Item.DisplayName(), out.Item.DisplayArtists())
		return nil
	}
	fmt.Fprintf(stdout, "%s on %s\n", stringsTitle(state), st.Device.Name)
	return nil
}

func stringsTitle(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
