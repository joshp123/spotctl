package spotctl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/joshp123/spotctl/internal/spotify"
)

func humanizeError(err error) error {
	var apiErr *spotify.APIError
	if errors.As(err, &apiErr) {
		m := apiErr.Message
		switch {
		case apiErr.StatusCode == 403 && strings.Contains(m, "Cannot control device volume"):
			return fmt.Errorf("Spotify won't let us change volume for that device via Web API. Adjust volume on the device and retry")
		case apiErr.StatusCode == 403 && strings.Contains(m, "Restriction violated"):
			return fmt.Errorf("Spotify refused that command (restriction). Try again on a different device (Desktop usually works) or start playback in-app first")
		case apiErr.StatusCode == 404 && strings.Contains(m, "No active device"):
			return fmt.Errorf("No active Spotify device. Open Spotify on the target device, then retry")
		}
	}
	return err
}
