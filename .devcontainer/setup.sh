#!/usr/bin/env bash
# One-time dev container provisioning: install mise, then the pinned toolchain
# (go / kubectl / kind / yq from mise.toml) plus `just`. Runs as onCreateCommand.
set -euo pipefail

echo "==> Installing mise"
curl -fsSL https://mise.run | sh

export PATH="$HOME/.local/bin:$PATH"

# Activate mise in interactive bash so shims stay on PATH after the container
# is attached (belt-and-suspenders with the PATH set in devcontainer.json).
if ! grep -q 'mise activate bash' "$HOME/.bashrc" 2>/dev/null; then
  echo 'eval "$(~/.local/bin/mise activate bash)"' >> "$HOME/.bashrc"
fi

echo "==> Installing pinned toolchain from mise.toml (go, kubectl, kind, yq)"
mise trust --quiet
mise install

# `just` isn't pinned in mise.toml (not part of the runtime toolchain); grab the
# latest here so `just <task>` works out of the box.
echo "==> Installing just"
mise use -g just@latest

echo "==> Warming the Go build cache"
eval "$(mise activate bash --shims)"
go build ./... || echo "note: go build reported issues (non-fatal for provisioning)"

echo "==> Done. Try:  just doctor   |   just up   |   just tui   |   just docs-dev"
