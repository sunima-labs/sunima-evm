#!/usr/bin/env bash
# Sync cosmos-evm mirror with upstream cosmos/evm
# Usage: ./scripts/sync-cosmos-evm.sh [--apply]
# Without --apply: dry-run (fetch + report new commits)
# With --apply: actually fast-forward and push to sunima-labs/cosmos-evm

set -euo pipefail
MIRROR_DIR="${MIRROR_DIR:-/root/projects/cosmos-evm}"
APPLY=false
[ "${1:-}" = "--apply" ] && APPLY=true

cd "$MIRROR_DIR"
git fetch upstream --tags 2>&1 | grep -E "^(\* |  -)" || true
git fetch origin 2>&1 | grep -E "^(\* |  -)" || true

LOCAL_HEAD=$(git rev-parse origin/main)
UPSTREAM_HEAD=$(git rev-parse upstream/main)

if [ "$LOCAL_HEAD" = "$UPSTREAM_HEAD" ]; then
  echo "in sync — no new upstream commits"
  exit 0
fi

NEW_COUNT=$(git rev-list --count $LOCAL_HEAD..$UPSTREAM_HEAD)
echo "upstream ahead by $NEW_COUNT commits"
git log --oneline $LOCAL_HEAD..$UPSTREAM_HEAD | head -20

if $APPLY; then
  echo "applying — fast-forward + push"
  git checkout main
  git merge --ff-only upstream/main
  git push origin main --tags
  echo "synced ✓"
else
  echo "dry-run — re-run with --apply to fast-forward and push"
fi
