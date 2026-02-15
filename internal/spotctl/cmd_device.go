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

func (c *cli) cmdDevice(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &exitError{code: 2, err: errors.New("missing subcommand for device")}
	}

	sub := args[0]
	args = args[1:]
	if sub == "list" {
		return c.cmdDeviceList(ctx, args, stdout, stderr)
	}
	return &exitError{code: 2, err: fmt.Errorf("unknown device subcommand: %s", sub)}
}

func (c *cli) cmdDeviceList(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	jsonTrailing, args := popBoolFlag(args, "--json")
	fs := flag.NewFlagSet("device list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return &exitError{code: 2, err: err}
	}
	if jsonTrailing {
		*jsonOut = true
	}

	devs, err := c.client.Devices(ctx)
	if err != nil {
		return err
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Devices []spotify.Device `json:"devices"`
		}{Devices: devs})
	}

	fmt.Fprintln(stdout, formatDevices(devs))
	return nil
}
