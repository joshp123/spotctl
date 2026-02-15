---
name: spotify
description: Control Spotify playback + playlists via the official Spotify Web API (OAuth refresh token) using the spotctl CLI.
---

# Spotify Skill (spotctl)

Use `spotctl` for Spotify playback, device targeting, and minimal playlist operations.

## Non-negotiables (v1)

- **Device targeting is strict.** If the requested device isn't available in `spotctl device list`, do **not** guess.
  - Tell the user: **"Open Spotify on that device, then retry."**
- **Never invent Spotify URLs/URIs.**
  - If you need a track URI from text, use `spotctl search tracks ... --json` and copy the returned `uri`.
  - `spotctl playlist add` validates track URIs exist (prevents hallucinated IDs).
- Prefer Spotify **URIs** (`spotify:track:...`, `spotify:playlist:...`) or `open.spotify.com/...` links.

## Auth / env

`spotctl` reads credentials from env variables (either literal values or file paths):

- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `SPOTIFY_REFRESH_TOKEN`

If the command fails with “missing SPOTIFY_*”, ask the operator to provision the secrets.

Required scopes:
- `user-read-playback-state`
- `user-modify-playback-state`
- `user-read-currently-playing`
- `playlist-read-private`
- `playlist-modify-private`
- `playlist-modify-public`
- `user-read-private`

## Common flows

### List devices

Human: “what devices can you play on?” / “what’s my iphone called?”

```bash
spotctl device list
spotctl device list --json
```

### Status

Human: “what’s playing?” / “are we playing right now?”

```bash
spotctl status
spotctl status --json
```

### Play on a specific device (strict)

Human: “play on my iphone” / “play on my mac”

1) Confirm the exact device name/id:
```bash
spotctl device list
```
2) Play:
```bash
spotctl play --device "Josh’s iPhone" spotify:track:3n3Ppam7vgaVa1iaRUc9Lp
```

If user gives a search query instead of a URI:
```bash
spotctl play --device "Josh’s iPhone" "daft punk one more time"
```
(Report what track was selected.)

### Transfer playback

Human: “move it to my phone/mac”

```bash
spotctl transfer --device "Josh’s iPhone"
```

### Playback controls

```bash
spotctl pause --device "Josh’s iPhone"
spotctl next --device "Josh’s iPhone"
spotctl previous --device "Josh’s iPhone"
spotctl volume --device "Josh’s iPhone" 35
```

Notes:
- Spotify may refuse some commands depending on device/context.
  - iPhone volume often fails (“Cannot control device volume”). Tell user to adjust volume on-device.
  - previous/next/pause can fail (“Restriction violated”). Suggest switching to Desktop or starting playback in-app first.

### Playlist ops (minimal v1)

Create playlist (private by default; pass `--public` to make it public):
```bash
spotctl playlist create --name "My New Playlist" --description "made by OpenClaw" --json
```

If Spotify creates it as public anyway (account quirks happen), use:
```bash
spotctl playlist privacy --playlist <playlist-id-or-uri> --private
```

If the API is rate-limited or privacy toggling fails, fallback:
- create an empty **secret/private** playlist in the Spotify client
- then use `spotctl playlist add --playlist <that-playlist> ...`

Search for candidate tracks (for vibe-based playlists):
```bash
spotctl search tracks "lofi focus beats" --limit 10 --json
```

Add tracks (URIs only in v1; use URIs returned by search):
```bash
spotctl playlist add --playlist spotify:playlist:37i9dQZF1DXcBWIGoYBM5M \
  spotify:track:3n3Ppam7vgaVa1iaRUc9Lp spotify:track:7ouMYWpwJ422jRcDASZB7P --json
```

Tip: `--json` can be at the end (agent-friendly).

## Strict device failure message

If device not available, reply succinctly:
- “That device isn’t available right now. Open Spotify on it (unlock / foreground), then retry.”
- Also show the currently-available device list (from `spotctl device list`).
