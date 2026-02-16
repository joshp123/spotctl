# AGENTS.md (spotctl)

Purpose: coding-agent runbook for the **spotctl** repo (Spotify Web API CLI + OpenClaw plugin export).

## What this repo is

- `spotctl` Go CLI: Spotify playback + devices + minimal playlist ops using official Spotify Web API + refresh token.
- Nix flake exports an `openclawPlugin` providing:
  - `spotctl` on PATH
  - `skills/spotify/SKILL.md` injected into OpenClaw

## Local-first workflow (strong preference)

Spotify API behavior is cloud-side; iterate locally, deploy to the bot host only for wiring verification.

### Dev shell

```bash
cd ~/code/spotctl

devenv shell
```

### Tests

```bash
go test ./...
```

### Local smoke

```bash
./scripts/smoke.sh
```

## Secrets / auth (no leaks)

`spotctl` reads (values or file paths):
- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `SPOTIFY_REFRESH_TOKEN`

Rules:
- Never print these values.
- Prefer pointing env vars at secret files (agenix outputs) instead of literal values.

## Test playlist hygiene (mandatory)

All test/smoke playlists MUST be named with a stable prefix:

- `spotctl-test:<suite>:<YYYYMMDD-HHMMSS>`
  - Example: `spotctl-test:smoke:20260215-220501`

Cleanup (safe by default; deletion requires confirmation):

```bash
spotctl playlist cleanup --prefix spotctl-test:
spotctl playlist cleanup --prefix spotctl-test: --apply --yes
```

## clig.dev principles

`spotctl` should follow https://clig.dev/:
- human output by default; `--json` for automation (stdout)
- errors on stderr; non-zero exit codes
- `--help` and usage work even without auth env (parse flags before `ensureClient()`)
- destructive operations require explicit confirmation (`--apply --yes`)

## Formatting / lint

- Go: `gofmt` (only on `.go` files)
- Shell: do NOT run `gofmt` on `scripts/*.sh` (it will fail on `#`)

## Release / shipping

- This repo is consumed via Nix flake pin (`rev` + `narHash`) in the bot host config.
- After changes:
  1) push to `main`
  2) compute new `narHash`
  3) bump consumer pin (e.g. `djtbot/nix/home/djtbot.nix`)
  4) deploy bot host
