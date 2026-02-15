package main

import (
	"context"
	"os"

	"github.com/joshp123/spotctl/internal/spotctl"
)

func main() {
	ctx := context.Background()
	code := spotctl.Main(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

// Keep a tiny main; real logic in internal/spotctl.
// (Lets us unit-test without exec.)
