package spotctl

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
)

func (c *cli) cmdPause(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	deviceID, err := c.optionalDeviceArg(ctx, "pause", args, stderr)
	if err != nil {
		return err
	}
	if err := c.client.Pause(ctx, deviceID); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Paused.")
	return nil
}

func (c *cli) cmdNext(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	deviceID, err := c.optionalDeviceArg(ctx, "next", args, stderr)
	if err != nil {
		return err
	}
	if err := c.client.Next(ctx, deviceID); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Next.")
	return nil
}

func (c *cli) cmdPrevious(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	deviceID, err := c.optionalDeviceArg(ctx, "previous", args, stderr)
	if err != nil {
		return err
	}
	if err := c.client.Previous(ctx, deviceID); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Previous.")
	return nil
}

func (c *cli) cmdVolume(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if err := c.ensureClient(ctx); err != nil {
		return err
	}
	fs := flag.NewFlagSet("volume", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	deviceSel := fs.String("device", "", "Device name or id")
	if err := parseFlags(fs, args, stderr); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return &exitError{code: 2, err: errors.New("volume requires one arg: 0-100")}
	}
	pct, err := strconv.Atoi(fs.Arg(0))
	if err != nil || pct < 0 || pct > 100 {
		return &exitError{code: 2, err: errors.New("volume must be an int 0-100")}
	}

	var deviceID *string
	if *deviceSel != "" {
		dev, devs, err := c.client.ResolveDevice(ctx, *deviceSel)
		if err != nil {
			return err
		}
		if dev == nil {
			return &exitError{code: 3, err: errors.New(strictDeviceMessage(*deviceSel, devs))}
		}
		tmp := dev.ID
		deviceID = &tmp
	}

	if err := c.client.Volume(ctx, deviceID, pct); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Volume set to %d%%\n", pct)
	return nil
}

func (c *cli) optionalDeviceArg(ctx context.Context, name string, args []string, stderr io.Writer) (*string, error) {
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
	dev, devs, err := c.client.ResolveDevice(ctx, *deviceSel)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, &exitError{code: 3, err: errors.New(strictDeviceMessage(*deviceSel, devs))}
	}
	id := dev.ID
	return &id, nil
}
