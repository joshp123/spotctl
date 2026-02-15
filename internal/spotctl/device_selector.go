package spotctl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

// parseOptionalDeviceSelector parses --device for commands like pause/next/previous.
// It does NOT require auth/client; callers can ensureClient after parsing.
func parseOptionalDeviceSelector(name string, args []string, stderr io.Writer) (*string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	deviceSel := fs.String("device", "", "Optional device name or id")
	if err := parseFlags(fs, args, stderr); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, &exitError{code: 2, err: fmt.Errorf("%s takes no positional args", name)}
	}
	if *deviceSel == "" {
		return nil, nil
	}
	out := *deviceSel
	return &out, nil
}

func (c *cli) resolveOptionalDeviceID(ctx context.Context, selector *string) (*string, error) {
	if selector == nil {
		return nil, nil
	}
	dev, devs, err := c.client.ResolveDevice(ctx, *selector)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, &exitError{code: 3, err: errors.New(strictDeviceMessage(*selector, devs))}
	}
	id := dev.ID
	return &id, nil
}
