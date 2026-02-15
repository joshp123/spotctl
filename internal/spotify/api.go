package spotify

import (
	"context"
	"fmt"
	"net/url"
)

// Devices returns the full list of Spotify Connect devices visible to the user.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	var res struct {
		Devices []Device `json:"devices"`
	}
	if err := c.do(ctx, "GET", "/v1/me/player/devices", nil, nil, &res, 200); err != nil {
		return nil, err
	}
	return res.Devices, nil
}

// ResolveDevice matches by exact id or case-insensitive exact name.
// Returns (device,nil,nil) if found.
// Returns (nil,devices,nil) if not found.
func (c *Client) ResolveDevice(ctx context.Context, selector string) (*Device, []Device, error) {
	devs, err := c.Devices(ctx)
	if err != nil {
		return nil, nil, err
	}

	// 1) exact id match
	for _, d := range devs {
		if d.ID == selector {
			dc := d
			return &dc, devs, nil
		}
	}

	// 2) case-insensitive exact name match
	var matches []Device
	for _, d := range devs {
		if stringsEqualFoldTrim(d.Name, selector) {
			matches = append(matches, d)
		}
	}
	if len(matches) == 1 {
		dc := matches[0]
		return &dc, devs, nil
	}

	// Ambiguous => treat as not found (caller can tell user to use id).
	return nil, devs, nil
}

func (c *Client) PlaybackState(ctx context.Context) (*PlaybackState, error) {
	var st PlaybackState
	if err := c.do(ctx, "GET", "/v1/me/player", nil, nil, &st, 200, 204); err != nil {
		return nil, err
	}
	// If Spotify returned 204, st will be the zero value.
	if st.Device.ID == "" && st.Device.Name == "" {
		return nil, nil
	}
	return &st, nil
}

func (c *Client) TransferPlayback(ctx context.Context, deviceID string, play bool) error {
	body := map[string]any{
		"device_ids": []string{deviceID},
		"play":       play,
	}
	return c.do(ctx, "PUT", "/v1/me/player", nil, body, nil, 200, 202, 204)
}

type PlayRequest struct {
	ContextURI string   `json:"context_uri,omitempty"`
	URIs       []string `json:"uris,omitempty"`
}

func (c *Client) Play(ctx context.Context, deviceID string, req PlayRequest) error {
	q := url.Values{}
	q.Set("device_id", deviceID)
	return c.do(ctx, "PUT", "/v1/me/player/play", q, req, nil, 200, 202, 204)
}

func (c *Client) Pause(ctx context.Context, deviceID *string) error {
	q := url.Values{}
	if deviceID != nil {
		q.Set("device_id", *deviceID)
	}
	return c.do(ctx, "PUT", "/v1/me/player/pause", q, nil, nil, 200, 202, 204)
}

func (c *Client) Next(ctx context.Context, deviceID *string) error {
	q := url.Values{}
	if deviceID != nil {
		q.Set("device_id", *deviceID)
	}
	return c.do(ctx, "POST", "/v1/me/player/next", q, nil, nil, 200, 202, 204)
}

func (c *Client) Previous(ctx context.Context, deviceID *string) error {
	q := url.Values{}
	if deviceID != nil {
		q.Set("device_id", *deviceID)
	}
	return c.do(ctx, "POST", "/v1/me/player/previous", q, nil, nil, 200, 202, 204)
}

func (c *Client) Volume(ctx context.Context, deviceID *string, pct int) error {
	q := url.Values{}
	q.Set("volume_percent", fmt.Sprintf("%d", pct))
	if deviceID != nil {
		q.Set("device_id", *deviceID)
	}
	return c.do(ctx, "PUT", "/v1/me/player/volume", q, nil, nil, 200, 202, 204)
}

func (c *Client) SearchTracks(ctx context.Context, query string, limit int) ([]Track, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("type", "track")
	q.Set("limit", fmt.Sprintf("%d", limit))
	var res struct {
		Tracks struct {
			Items []Track `json:"items"`
		} `json:"tracks"`
	}
	if err := c.do(ctx, "GET", "/v1/search", q, nil, &res, 200); err != nil {
		return nil, err
	}
	return res.Tracks.Items, nil
}

func (c *Client) SearchTopTrack(ctx context.Context, query string) (Track, error) {
	items, err := c.SearchTracks(ctx, query, 1)
	if err != nil {
		return Track{}, err
	}
	if len(items) == 0 {
		return Track{}, fmt.Errorf("no search results for %q", query)
	}
	return items[0], nil
}

func (c *Client) GetTrack(ctx context.Context, id string) (Track, error) {
	q := url.Values{}
	q.Set("market", "from_token")
	var t Track
	path := fmt.Sprintf("/v1/tracks/%s", url.PathEscape(id))
	if err := c.do(ctx, "GET", path, q, nil, &t, 200); err != nil {
		return Track{}, err
	}
	return t, nil
}

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

func (c *Client) Me(ctx context.Context) (User, error) {
	var u User
	if err := c.do(ctx, "GET", "/v1/me", nil, nil, &u, 200); err != nil {
		return User{}, err
	}
	return u, nil
}

type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type PlaylistDetails struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URI    string `json:"uri"`
	Public *bool  `json:"public"`
}

func (c *Client) CreatePlaylist(ctx context.Context, name string, public bool, description string) (Playlist, error) {
	body := map[string]any{
		"name":        name,
		"public":      public,
		"description": description,
	}
	var pl Playlist
	// Spotify currently allows playlist creation via /v1/me/playlists.
	// (Some accounts/apps get 403 on /v1/users/{id}/playlists.)
	if err := c.do(ctx, "POST", "/v1/me/playlists", nil, body, &pl, 201); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (c *Client) PlaylistDetails(ctx context.Context, playlistID string) (PlaylistDetails, error) {
	path := fmt.Sprintf("/v1/playlists/%s", url.PathEscape(playlistID))
	q := url.Values{}
	// Reduce payload + rate limit pressure; enough to verify privacy.
	q.Set("fields", "id,name,uri,public")
	var pl PlaylistDetails
	if err := c.do(ctx, "GET", path, q, nil, &pl, 200); err != nil {
		return PlaylistDetails{}, err
	}
	return pl, nil
}

func (c *Client) UpdatePlaylistDetails(ctx context.Context, playlistID string, public *bool, name *string, description *string) error {
	body := map[string]any{}
	if public != nil {
		body["public"] = *public
	}
	if name != nil {
		body["name"] = *name
	}
	if description != nil {
		body["description"] = *description
	}
	path := fmt.Sprintf("/v1/playlists/%s", url.PathEscape(playlistID))
	return c.do(ctx, "PUT", path, nil, body, nil, 200, 202, 204)
}

type AddTracksResult struct {
	SnapshotID string `json:"snapshot_id"`
}

func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID string, uris []string) (AddTracksResult, error) {
	body := map[string]any{"uris": uris}
	var res AddTracksResult
	// Spotify currently supports adding playlist items via /v1/playlists/{id}/items.
	// (Some accounts/apps get 403 on /v1/playlists/{id}/tracks.)
	path := fmt.Sprintf("/v1/playlists/%s/items", url.PathEscape(playlistID))
	if err := c.do(ctx, "POST", path, nil, body, &res, 201); err != nil {
		return AddTracksResult{}, err
	}
	return res, nil
}
