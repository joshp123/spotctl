# Testing (local-first)

Rule: iterate on Spotify Web API behavior **locally** (fast). Use the remote OpenClaw host only to validate Nix wiring (PATH + skills + /run/agenix).

## Prereqs

Environment variables (values OR file paths):

- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `SPOTIFY_REFRESH_TOKEN`

## Quick smoke

From repo root:

```bash
./scripts/smoke.sh
```

Optional:
- `SPOTCTL_BIN=...` to force a specific binary
- `SPOTCTL_DEVICE=...` to force device selection (name or id)

## Notes on expected failures

Spotify Web API can legitimately refuse some commands depending on device/context.
We treat these as **WARN** (not hard failures) in smoke:

- Volume on some devices (often iPhone) → 403 “Cannot control device volume”
- Previous/Next/Pause sometimes → 403 “Restriction violated”

Mitigation: try Desktop device, or start playback in-app first.
