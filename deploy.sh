#!/usr/bin/env bash
set -euo pipefail

REPO="$HOME/transpara-ai/repos/lovyou-ai-site"

step() { echo "==> $1"; }
fail() { echo "FAIL: $1" >&2; exit 1; }

cd "$REPO" || fail "cd to $REPO"

step "templ generate"
export PATH="$HOME/go/bin:$PATH"
templ generate || fail "templ generate"

step "go build"
go build -o site ./cmd/site/ || fail "go build"

step "setcap"
sudo /usr/sbin/setcap 'cap_net_bind_service=+ep' ./site || fail "setcap"

step "restart service"
systemctl --user restart lovyou-ai-site || fail "systemctl restart"

step "health check"
sleep 2
status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/health)
[ "$status" = "200" ] || fail "health check returned $status"

echo "deploy complete — health 200"
