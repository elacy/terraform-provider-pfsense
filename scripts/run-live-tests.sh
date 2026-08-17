#!/usr/bin/env bash
#
# run-live-tests.sh — run the live (TF_ACC) acceptance tests against a pfSense box.
#
# The provider's acceptance tests are opt-in: they only touch a real pfSense
# instance when TF_ACC=1 is exported, and they read the connection details from
# the environment (PFSENSE_URL / PFSENSE_USERNAME / PFSENSE_PASSWORD, or
# PFSENSE_API_KEY). This script just wires those up from the command line and
# runs `go test` for you, so you don't have to remember the env-var contract.
#
# The endpoint is REQUIRED — there is deliberately no default, because pointing
# live, mutating tests at the wrong box would be bad. Credentials fall back to
# the PFSENSE_* environment variables when not passed explicitly.
#
# Usage:
#   scripts/run-live-tests.sh https://10.99.0.2 --username admin --password secret
#   scripts/run-live-tests.sh --url https://pf.example --api-key "$KEY"
#   PFSENSE_URL=... PFSENSE_USERNAME=... PFSENSE_PASSWORD=... scripts/run-live-tests.sh
#   scripts/run-live-tests.sh https://10.99.0.2 -u admin -p secret -- -run TestAccFirewallAliasResourceLive
#
# Any args after a bare `--` are passed straight through to `go test` (e.g. a
# -run filter to target one test). Without a filter it runs the full suite with
# TF_ACC=1 — equivalent to `make testacc` — so the mock-backed tests run too
# and every live test (the alias CRUD test + the read-only data-source
# inventories) is exercised against the box.
set -euo pipefail

usage() {
  sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
}

URL="${PFSENSE_URL:-}"
USERNAME="${PFSENSE_USERNAME:-}"
PASSWORD="${PFSENSE_PASSWORD:-}"
API_KEY="${PFSENSE_API_KEY:-}"
PASSTHRU=()
POSITIONAL=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    -u|--url)      URL="${2:-}"; shift 2 ;;
    --url=*)       URL="${1#*=}"; shift ;;
    -U|--username) USERNAME="${2:-}"; shift 2 ;;
    --username=*)  USERNAME="${1#*=}"; shift ;;
    -P|--password) PASSWORD="${2:-}"; shift 2 ;;
    --password=*)  PASSWORD="${1#*=}"; shift ;;
    -k|--api-key)  API_KEY="${2:-}"; shift 2 ;;
    --api-key=*)   API_KEY="${1#*=}"; shift ;;
    --)            shift; PASSTHRU+=("$@"); break ;;
    -*)            PASSTHRU+=("$1"); shift ;;   # unknown flag -> pass to go test
    *)             POSITIONAL+=("$1"); shift ;;
  esac
done

# A bare positional URL is the common case: `./run-live-tests.sh https://box`.
if [[ -z "$URL" && ${#POSITIONAL[@]} -gt 0 ]]; then
  URL="${POSITIONAL[0]}"
  POSITIONAL=("${POSITIONAL[@]:1}")
fi
PASSTHRU=("${POSITIONAL[@]}" "${PASSTHRU[@]}")

if [[ -z "$URL" ]]; then
  echo "error: no endpoint given." >&2
  echo "pass it as the first arg or --url, or set PFSENSE_URL." >&2
  echo >&2
  usage >&2
  exit 2
fi

# Match the acceptance-test precheck: an API key alone is fine, or a complete
# username/password pair — but a username without its password is a mistake.
if [[ -z "$API_KEY" && ( -z "$USERNAME" || -z "$PASSWORD" ) ]]; then
  echo "error: need --api-key, or both --username and --password (or their PFSENSE_* env vars)." >&2
  exit 2
fi

export TF_ACC=1
export PFSENSE_URL="$URL"
export PFSENSE_USERNAME="$USERNAME"
export PFSENSE_PASSWORD="$PASSWORD"
export PFSENSE_API_KEY="$API_KEY"

echo "==> live acceptance tests against $URL"
echo "==> auth: $([ -n "$API_KEY" ] && echo 'API key' || echo "username '$USERNAME'")"
echo "==> go test ./... -v -timeout 120m ${PASSTHRU[*]:-}"

go test ./... -v -timeout 120m "${PASSTHRU[@]:-}"
