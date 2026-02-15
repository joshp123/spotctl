# Obtaining a Spotify refresh token (one-time)

Goal: produce a long-lived `SPOTIFY_REFRESH_TOKEN` for `spotctl` / OpenClaw.

You need a Spotify Developer app:
- https://developer.spotify.com/dashboard
- Note **Client ID** + **Client Secret**
- Add a Redirect URI (example below)

## Scopes

Use these scopes (space-separated):

- user-read-playback-state
- user-modify-playback-state
- user-read-currently-playing
- playlist-read-private
- playlist-modify-private
- playlist-modify-public
- user-read-private

## Flow (copy/paste)

1) Export client credentials (values or files):

```bash
export SPOTIFY_CLIENT_ID="..."
export SPOTIFY_CLIENT_SECRET="..."
```

2) Choose a redirect URI that exactly matches your Spotify app settings.

Recommended redirect URI for local bootstrap (matches Spotify UI constraints):

```bash
REDIRECT_URI="https://localhost:8899/callback"
```

3) Automated flow (no copy/paste of `code=`):

```bash
spotctl auth login --redirect-uri "$REDIRECT_URI"
```

Notes:
- `spotctl` runs a local HTTPS callback server with a **self-signed certificate**.
- Your browser will warn; click through (Advanced → proceed).
- The refresh token prints on stdout.

4) Optional: non-interactive write into an agenix secret (no copy/paste):

```bash
cd ~/code/nix/nix-secrets
spotctl auth login --redirect-uri "$REDIRECT_URI" | agenix -e spotify-refresh-token.age
```

## Install into OpenClaw runtime

Provide the refresh token (and client creds) as env vars (values or file paths):

- SPOTIFY_CLIENT_ID
- SPOTIFY_CLIENT_SECRET
- SPOTIFY_REFRESH_TOKEN

On NixOS/OpenClaw hosts, we typically provide these as files under `/run/agenix/...`.
