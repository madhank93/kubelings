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
