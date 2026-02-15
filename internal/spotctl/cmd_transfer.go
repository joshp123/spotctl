package spotctl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

func (c *cli) cmdTransfer(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	fs := flag.NewFlagSet("transfer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	deviceSel := fs.String("device", "", "Device name or id (strict)")
	play := fs.Bool("play", true, "Start playback after transfer")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if *deviceSel == "" {
		return &exitError{code: 2, err: errors.New("missing --device")}
	}
	if fs.NArg() != 0 {
		return &exitError{code: 2, err: errors.New("transfer takes no positional args")}
	}

	dev, devs, err := c.client.ResolveDevice(ctx, *deviceSel)
	if err != nil {
		return err
	}
	if dev == nil {
		return &exitError{code: 3, err: errors.New(strictDeviceMessage(*deviceSel, devs))}
	}

	if err := c.client.TransferPlayback(ctx, dev.ID, *play); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Transferred to %s (id=%s)\n", dev.Name, dev.ID)
	return nil
}
