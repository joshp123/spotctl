package spotify

import (
	"context"
	"fmt"
	"net/url"
)

type myPlaylistsPage struct {
	Items  []Playlist `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	Next   string     `json:"next"`
}

func (c *Client) MyPlaylists(ctx context.Context) ([]Playlist, error) {
	var out []Playlist
	limit := 50
	offset := 0
	for page := 0; page < 50; page++ { // hard cap: 2500 playlists
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("offset", fmt.Sprintf("%d", offset))
		// Reduce payload + rate limit pressure.
		q.Set("fields", "items(id,name,uri),total,limit,offset,next")
		var res myPlaylistsPage
		if err := c.do(ctx, "GET", "/v1/me/playlists", q, nil, &res, 200); err != nil {
			return nil, err
		}
		out = append(out, res.Items...)
		offset = res.Offset + res.Limit
		if res.Next == "" || offset >= res.Total {
			break
		}
	}
	return out, nil
}

func (c *Client) UnfollowPlaylist(ctx context.Context, playlistID string) error {
	path := fmt.Sprintf("/v1/playlists/%s/followers", url.PathEscape(playlistID))
	return c.do(ctx, "DELETE", path, nil, nil, nil, 200, 202, 204)
}
