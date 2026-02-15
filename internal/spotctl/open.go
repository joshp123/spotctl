package spotctl

import (
	"context"
	"os/exec"
	"runtime"
)

func openURL(ctx context.Context, u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", u)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", u)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u)
	default:
		return nil
	}
	return cmd.Start()
}
