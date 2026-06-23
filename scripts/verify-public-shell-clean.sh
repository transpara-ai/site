#!/usr/bin/env bash
set -euo pipefail

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

set +e
rg -n \
  --glob '!**/*_test.go' \
  --glob '!**/*_templ.go' \
  --glob '!third_party/**' \
  --glob '!content/**' \
  --glob '!docs/**' \
  -e 'AI colleague' \
  -e 'Try it free' \
  -e '#e8a0b8' \
  -e 'lovyou-ai' \
  -e 'Humans and agents, building together' \
  -e 'AI agent work together' \
  -e 'Take care of your human' \
  auth cmd/site profile static views graph/views.templ graph/handlers.go \
  | grep -v 'public-shell-clean: allow quarantined legacy slug metadata' >"$tmp"
status=$?
set -e

if [[ $status -eq 0 ]]; then
  cat "$tmp"
  echo "public-shell cleanup violation: legacy brand string or color returned"
  exit 1
fi

if [[ $status -ne 1 ]]; then
  echo "public-shell cleanup scan failed"
  exit "$status"
fi
