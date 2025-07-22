# Default command: List all available just commands
default:
    @just --list

# Run both Go services concurrently with prefixed logs
run:
    set -euo pipefail; \
    trap 'echo "Shutting down..."; kill 0' SIGINT SIGTERM; \
    go run ./cmd/prism/main.go --debug 2>&1 | sed "s/^/[PRISM] /" & \
    go run ./cmd/igniterelay/main.go --debug 2>&1 | sed "s/^/[IGNITERELAY] /" & \
    wait

lint:
  staticcheck ./...

# Run tests excluding any package matching '/pkgs/api'
test:
   go test $(go list ./... | grep -v -E '(/pkgs/api|/examples)')

# Update nix flakes and Go modules
update:
    nix flake update
    go get -u ./...

# Build nix flake for the specified profile (default: 'default')
package profile='default':
    nix build \
        --json \
        --no-link \
        --print-build-logs \
        '.#{{ profile }}'

# Generate Go code from proto files in the api directory
generate:
    ./scripts/generate.sh
