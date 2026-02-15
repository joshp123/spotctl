package spotify

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type URIKind string

const (
	URIKindUnknown  URIKind = "unknown"
	URIKindTrack    URIKind = "track"
	URIKindAlbum    URIKind = "album"
	URIKindPlaylist URIKind = "playlist"
	URIKindArtist   URIKind = "artist"
	URIKindShow     URIKind = "show"
	URIKindEpisode  URIKind = "episode"
)

var spotifyURIRe = regexp.MustCompile(`^spotify:([a-zA-Z]+):([a-zA-Z0-9]{22})$`)

func NormalizeURI(s string) (uri string, kind URIKind, err error) {
	ss := strings.TrimSpace(s)
	if ss == "" {
		return "", URIKindUnknown, errors.New("empty uri")
	}

	if m := spotifyURIRe.FindStringSubmatch(ss); m != nil {
		kind = URIKind(strings.ToLower(m[1]))
		return ss, kind, nil
	}

	if strings.HasPrefix(ss, "https://open.spotify.com/") || strings.HasPrefix(ss, "http://open.spotify.com/") {
		u, err := url.Parse(ss)
		if err != nil {
			return "", URIKindUnknown, err
		}
		// /track/<id>, /album/<id>, /playlist/<id>, /artist/<id>
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			k := strings.ToLower(parts[0])
			id := parts[1]
			// Some URLs have extra segments, ignore.
			if len(id) == 22 {
				uri := fmt.Sprintf("spotify:%s:%s", k, id)
				return uri, URIKind(k), nil
			}
		}
		return "", URIKindUnknown, fmt.Errorf("unsupported spotify url: %s", ss)
	}

	return "", URIKindUnknown, nil
}

var playlistIDRe = regexp.MustCompile(`^[A-Za-z0-9]{22}$`)

func NormalizePlaylistID(s string) (string, error) {
	uri, kind, err := NormalizeURI(s)
	if err != nil {
		return "", err
	}
	if kind == URIKindPlaylist {
		m := spotifyURIRe.FindStringSubmatch(uri)
		if m == nil {
			return "", fmt.Errorf("invalid playlist uri: %s", s)
		}
		return m[2], nil
	}

	ss := strings.TrimSpace(s)
	if playlistIDRe.MatchString(ss) {
		return ss, nil
	}

	// URL case handled by NormalizeURI above; only playlist URLs allowed.
	if strings.Contains(ss, "open.spotify.com") {
		return "", fmt.Errorf("expected playlist url; got: %s", ss)
	}

	return "", fmt.Errorf("invalid playlist selector: %s", s)
}

func stringsEqualFoldTrim(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
