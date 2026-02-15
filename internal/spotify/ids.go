package spotify

import (
	"fmt"
)

func TrackIDFromURI(uri string) (string, error) {
	_, kind, err := NormalizeURI(uri)
	if err != nil {
		return "", err
	}
	if kind != URIKindTrack {
		return "", fmt.Errorf("not a track uri: %s", uri)
	}
	m := spotifyURIRe.FindStringSubmatch(uri)
	if m == nil {
		return "", fmt.Errorf("invalid spotify uri: %s", uri)
	}
	return m[2], nil
}
