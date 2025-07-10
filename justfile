# List available commands
default:
    @just --list

# Run both Go services concurrently
run:
    set -euo pipefail; \
    trap 'echo "Shutting down..."; kill 0' SIGINT SIGTERM; \
    go run ./cmd/prism/main.go --debug 2>&1 | sed "s/^/[PRISM] /" & \
    go run ./cmd/igniterelay/main.go --debug 2>&1 | sed "s/^/[IGNITERELAY] /" & \
    wait

update:
    nix flake update
    go get -u ./...

package profile='default':
    nix build \
        --json \
        --no-link \
        --print-build-logs \
        '.#{{ profile }}'

generate:
    rm -rfv pkgs/api
    mkdir -pv pkgs/api
    protoc \
      --go_opt=paths=source_relative \
      --go_out=pkgs/api \
      --go-grpc_opt=paths=source_relative \
      --go-grpc_out=pkgs/api \
      --proto_path=api \
      server/server.proto
    protoc \
      --go_opt=paths=source_relative \
      --go_out=pkgs/api \
      --go-grpc_opt=paths=source_relative \
      --go-grpc_out=pkgs/api \
      --proto_path=api \
      communication/communication.proto
