#!/usr/bin/env bash
# Convert a kubelings challenge (challenges/<slug>/index.md) into an iximiuz
# course LESSON: a lesson index.md (playground + tasks) plus a unit-1.md whose
# body is the challenge's prose with a ::simple-task bound to the verify task.
#
# Usage:
#   tools/challenge-to-lesson.sh <challenge-dir> <dest-lesson-dir> <name> <slug> [verify-task]
#
# Single source of truth stays the challenge; re-run to regenerate the lesson.
set -euo pipefail

CH="${1:?challenge dir}"; DEST="${2:?dest lesson dir}"
NAME="${3:?lesson name}"; SLUG="${4:?lesson slug}"
VTASK="${5:-verify_done}"
[ -f "$CH/index.md" ] || { echo "no $CH/index.md"; exit 2; }

frontmatter() { awk '/^---$/{c++; next} c==1{print}' "$1/index.md"; }
body()        { awk 'f{print} /^---$/{c++; if(c==2)f=1}' "$1/index.md"; }

mkdir -p "$DEST"
FM="$(frontmatter "$CH")"
TITLE="$(echo "$FM" | yq -r '.title')"

# Lesson frontmatter: keep title/description/playground/tasks; lessons don't take
# categories/tagz/difficulty.
{
  echo '---'
  echo "$FM" | NAME="$NAME" SLUG="$SLUG" DATE="$(date +%F)" yq -P '{
    "kind":"lesson",
    "title":.title,
    "description":.description,
    "name":strenv(NAME),
    "slug":strenv(SLUG),
    "createdAt":strenv(DATE),
    "playground":.playground,
    "tasks":.tasks
  }'
  echo '---'
} > "$DEST/index.md"

# Unit: challenge prose + a simple-task that turns green when verify passes.
{
  echo '---'
  echo 'kind: unit'
  echo "title: \"$TITLE\""
  echo "name: ${SLUG}-unit"
  echo '---'
  echo
  body "$CH"
  cat <<EOF

::simple-task
---
:tasks: tasks
:name: $VTASK
---
#active
Solve the task above — this check turns green once verification passes.

#completed
✅ Solved — nicely done!
::
EOF
} > "$DEST/unit-1.md"

echo "wrote $DEST/{index.md,unit-1.md}  (from $CH, verify=$VTASK)"
