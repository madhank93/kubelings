#!/usr/bin/env bash
# Publish a kubelings challenge to iximiuz Labs.
#
# Usage: tools/publish.sh <id>
#
# First publish for an <id>: registers remote content (labctl appends a random
# slug suffix, e.g. kb-wl-01-1a2b3c4d), records <id>\t<slug> in .labctl/slugs.tsv,
# and renames challenges/<id> -> challenges/<slug> so default --dir resolves.
# Subsequent runs: push the existing slug's dir with --force.
set -euo pipefail

ID="${1:?usage: publish.sh <id>}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MAP="$ROOT/.labctl/slugs.tsv"
touch "$MAP"

slug_for() { awk -F'\t' -v id="$1" '$1==id{print $2}' "$MAP"; }

SLUG="$(slug_for "$ID")"

if [ -z "$SLUG" ]; then
  # First publish — register remote, capture the suffixed slug.
  [ -d "$ROOT/challenges/$ID" ] || { echo "no such challenge dir: challenges/$ID"; exit 2; }
  TMP="$(mktemp -d)"
  echo "registering remote challenge for $ID ..."
  # --quiet prints just the assigned content name (slug).
  SLUG="$(labctl content create challenge "$ID" --dir "$TMP" --no-open --quiet | tail -1)"
  rmdir "$TMP" 2>/dev/null || true
  [ -n "$SLUG" ] || { echo "failed to obtain slug from labctl create"; exit 1; }
  printf '%s\t%s\n' "$ID" "$SLUG" >>"$MAP"
  if [ "$ID" != "$SLUG" ]; then
    mv "$ROOT/challenges/$ID" "$ROOT/challenges/$SLUG"
  fi
  echo "registered: $ID -> $SLUG"
fi

DIR="$ROOT/challenges/$SLUG"
[ -d "$DIR" ] || { echo "missing dir for slug: $DIR"; exit 2; }

echo "pushing $SLUG ..."
labctl content push challenge "$SLUG" --dir "$DIR" --force
echo "pushed: https://labs.iximiuz.com/challenges/$SLUG"
