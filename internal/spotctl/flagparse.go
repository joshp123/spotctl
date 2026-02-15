package spotctl

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

func parseFlags(fs *flag.FlagSet, args []string, stderr io.Writer) error {
	if err := fs.Parse(args); err != nil {
		// flag package signals help via ErrHelp.
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(stderr)
			fs.Usage()
			return &exitError{code: 0, err: nil}
		}
		fmt.Fprintln(stderr, err)
		fs.SetOutput(stderr)
		fs.Usage()
		return &exitError{code: 2, err: nil}
	}
	return nil
}
