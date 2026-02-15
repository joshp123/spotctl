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

## CLI principles

`spotctl` aims to follow https://clig.dev/ principles:
- human-readable output by default
- machine-friendly JSON when `--json` is set (stdout)
- actionable errors on stderr with non-zero exit codes
- non-interactive commands by default (auth helpers are the exception)

## Playlist privacy

Playlists created by `spotctl playlist create` are **private by default**. Use `--public` to create a public playlist.

If Spotify creates it as public anyway, you can flip it:

```bash
spotctl playlist privacy --playlist <playlist-id-or-uri> --private
```

Fallback (privacy-first): create the playlist as secret/private in the Spotify client, then add tracks with `spotctl playlist add`.

## Test playlist naming + cleanup

When creating playlists for tests/smoke, use the standard prefix:
- `spotctl-test:<suite>:<YYYYMMDD-HHMMSS>`

Cleanup:
```bash
spotctl playlist cleanup --prefix spotctl-test: --apply --yes
```

## Quick smoke

```bash
spotctl device list --json
spotctl status --json
spotctl play --device "My Mac" spotify:track:3n3Ppam7vgaVa1iaRUc9Lp
```

## Refresh token bootstrap

See: `docs/REFRESH_TOKEN.md`
