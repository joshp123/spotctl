#!/usr/bin/env bash
set -euo pipefail

# spotctl local smoke test.
# Expects SPOTIFY_CLIENT_ID/SECRET/REFRESH_TOKEN to be set (values or file paths).

need() {
  if [ -z "${!1:-}" ]; then
    echo "missing env: $1" >&2
    exit 2
  fi
}

need SPOTIFY_CLIENT_ID
need SPOTIFY_CLIENT_SECRET
need SPOTIFY_REFRESH_TOKEN

# Pick a spotctl binary:
# - SPOTCTL_BIN override
# - ./result/bin/spotctl (nix build)
# - spotctl on PATH
# - go run
spotctl_cmd() {
  if [ -n "${SPOTCTL_BIN:-}" ]; then
    echo "$SPOTCTL_BIN"
    return
  fi
  if [ -x "./result/bin/spotctl" ]; then
    echo "./result/bin/spotctl"
    return
  fi
  if command -v spotctl >/dev/null 2>&1; then
    echo "spotctl"
    return
  fi
  echo "go run ./cmd/spotctl"
}

SPOTCTL=$(spotctl_cmd)

pass() { echo "PASS: $*"; }
warn() { echo "WARN: $*"; }
fail() { echo "FAIL: $*"; exit 1; }

run_hard() {
  local name=$1
  shift
  if $SPOTCTL "$@" >/dev/null; then
    pass "$name"
  else
    fail "$name"
  fi
}

run_soft() {
  local name=$1
  shift
  if out=$($SPOTCTL "$@" 2>&1); then
    pass "$name"
  else
    warn "$name :: $out"
  fi
}

echo "== device list =="
devs_json=$($SPOTCTL device list --json)
# If Spotify devices are asleep, the API can report none.
# Retry once after a short pause.
if [ "$(echo "$devs_json" | jq -r '.devices | length')" = "0" ]; then
  sleep 2
  devs_json=$($SPOTCTL device list --json)
fi
echo "$devs_json" | jq '.devices | length' >/dev/null || fail "device list json"
pass "device list"

# choose device
sel="${SPOTCTL_DEVICE:-}"
if [ -z "$sel" ]; then
  sel=$(echo "$devs_json" | jq -r '.devices[] | select(.is_active==true) | .id' | head -1)
fi
if [ -z "$sel" ]; then
  sel=$(echo "$devs_json" | jq -r '.devices[0].id' | head -1)
fi
if [ -z "$sel" ] || [ "$sel" = "null" ]; then
  fail "no spotify devices found (open spotify on a device and retry)"
fi

echo "device=$sel"

echo "== status =="
$SPOTCTL status --json | jq '.active' >/dev/null || fail "status"
pass "status"

echo "== playlist create+add =="
pid=$($SPOTCTL playlist create --name "spotctl-test:smoke:$(date +%Y%m%d-%H%M%S)" --description "spotctl smoke" --json | jq -r .id)
if [ -z "$pid" ] || [ "$pid" = "null" ]; then
  fail "playlist create"
fi
pass "playlist create"
run_hard "playlist add" playlist add --playlist "$pid" spotify:track:3n3Ppam7vgaVa1iaRUc9Lp --json

# search (sanity; also used to avoid hallucinated URIs)
$SPOTCTL search tracks "mr brightside the killers" --limit 3 --json | jq -r '.items[0].uri' >/dev/null || fail "search tracks"
pass "search tracks"

# playback (soft because spotify restrictions vary)
echo "== playback controls (soft) =="
run_soft "pause" pause --device "$sel"
run_soft "play" play --device "$sel" spotify:track:3n3Ppam7vgaVa1iaRUc9Lp
run_soft "next" next --device "$sel"
run_soft "previous" previous --device "$sel"
run_soft "volume 30" volume --device "$sel" 30

# strict targeting should fail

echo "== strict device policy =="
if $SPOTCTL play --device DOES_NOT_EXIST spotify:track:3n3Ppam7vgaVa1iaRUc9Lp >/dev/null 2>&1; then
  fail "strict device failure expected"
fi
pass "strict device failure"

echo OK
