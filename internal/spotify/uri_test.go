package spotify

import "testing"

func TestNormalizeURI_SpotifyURI(t *testing.T) {
	uri, kind, err := NormalizeURI("spotify:track:3n3Ppam7vgaVa1iaRUc9Lp")
	if err != nil {
		t.Fatal(err)
	}
	if uri != "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp" {
		t.Fatalf("uri=%q", uri)
	}
	if kind != URIKindTrack {
		t.Fatalf("kind=%q", kind)
	}
}

func TestNormalizeURI_SpotifyURL(t *testing.T) {
	uri, kind, err := NormalizeURI("https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M")
	if err != nil {
		t.Fatal(err)
	}
	if uri != "spotify:playlist:37i9dQZF1DXcBWIGoYBM5M" {
		t.Fatalf("uri=%q", uri)
	}
	if kind != URIKindPlaylist {
		t.Fatalf("kind=%q", kind)
	}
}

func TestNormalizePlaylistID(t *testing.T) {
	id, err := NormalizePlaylistID("spotify:playlist:37i9dQZF1DXcBWIGoYBM5M")
	if err != nil {
		t.Fatal(err)
	}
	if id != "37i9dQZF1DXcBWIGoYBM5M" {
		t.Fatalf("id=%q", id)
	}
}
