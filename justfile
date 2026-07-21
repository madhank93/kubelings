# kubelings — local dev tasks. Run `just` to list.

# Build + launch the interactive TUI.
tui: build
    ./bin/kubelings

# Build the TUI binary.
build:
    go build -o bin/kubelings ./cmd/kubelings

# Headless: env, cluster status, and lessons (no TUI).
doctor: build
    ./bin/kubelings doctor

# Run Go tests.
test:
    go test ./...

# Cluster lifecycle (delegates to the bash runner).
up:
    scripts/run-challenge-local.sh up

down:
    scripts/run-challenge-local.sh down

# Run a single lesson non-interactively: `just run cronjobs init`
run lesson verb="verify":
    scripts/run-challenge-local.sh {{lesson}} {{verb}}

# List lessons.
list:
    scripts/run-challenge-local.sh list

# Regenerate docs/src/data/catalog.ts from the course (the source of truth).
# Also validates .labctl/cloud-only.tsv against the lessons on disk.
gen-catalog:
    python3 docs/scripts/gen-catalog.py

# Docs site (Astro Starlight, in docs/).
docs-dev:
    cd docs && npm install && npm run dev

docs-build:
    cd docs && npm install && npm run build
