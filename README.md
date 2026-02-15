# spotctl

`spotctl` is a minimal Spotify Web API CLI (OAuth refresh token) designed for reliability.

It also exports an **OpenClaw plugin** (`openclawPlugin` flake output) that provides:
- the `spotctl` binary on PATH
- a `spotify` skill (`skills/spotify/SKILL.md`) teaching agents how to use the CLI

## Auth env

`spotctl` reads credentials from env vars (values or file paths):

- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `SPOTIFY_REFRESH_TOKEN`

## Playlist privacy

Playlists created by `spotctl playlist create` are **private by default**. Use `--public` to create a public playlist.

If Spotify creates it as public anyway, you can flip it:

```bash
spotctl playlist privacy --playlist <playlist-id-or-uri> --private
```

Fallback (privacy-first): create the playlist as secret/private in the Spotify client, then add tracks with `spotctl playlist add`.

## Quick smoke

```bash
spotctl device list --json
spotctl status --json
spotctl play --device "My Mac" spotify:track:3n3Ppam7vgaVa1iaRUc9Lp
```

## Refresh token bootstrap

See: `docs/REFRESH_TOKEN.md`
