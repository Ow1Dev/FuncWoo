#!/bin/bash
set -euo pipefail

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log "Starting protobuf generation..."

API_DIR="pkgs/api"

log "Cleaning up existing generated files in ${API_DIR}..."
rm -rfv "${API_DIR}"

log "Creating directory ${API_DIR}..."
mkdir -pv "${API_DIR}"

PROTO_FILES=(
  "server/server.proto"
  "communication/communication.proto"
)

for proto_file in "${PROTO_FILES[@]}"; do
  log "Processing proto file: api/${proto_file}"
  protoc \
    --go_opt=paths=source_relative \
    --go_out="${API_DIR}" \
    --go-grpc_opt=paths=source_relative \
    --go-grpc_out="${API_DIR}" \
    --proto_path=api \
    "api/${proto_file}"
  log "Successfully generated code from ${proto_file}"
done

log "Protobuf generation completed successfully."
