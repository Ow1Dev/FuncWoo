# List available commands
default:
    @just --list

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
      server.proto
