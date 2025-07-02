#!/bin/bash
set -euo pipefail

echo "Starting build for: ./examples/echo/"
temp_dir=$(mktemp -d)
echo "Using temporary directory: $temp_dir"

echo "Building Go binary (CGO_ENABLED=0, GOOS=linux)..."
CGO_ENABLED=0 GOOS=linux go build -o "$temp_dir/echo" ./examples/echo/main.go 

echo "Computing single SHA256 for folder contents..."
shasum=$(find "$temp_dir" -type f -print0 | sort -z | xargs -0 cat | sha256sum | awk '{print $1}')
echo "Checksum: $shasum"

funcs_dir="/var/lib/funcwoo/funcs/${shasum}"
action_dir="/var/lib/funcwoo/action"

echo "Creating target directory"
mkdir -p "$funcs_dir"

echo "Moving files to target directory..."
mv "$temp_dir"/* "$funcs_dir/"

echo "Cleaning up temporary directory..."
rm -rf "$temp_dir"

echo "Setting up action reference"
mkdir -p "$action_dir"
echo "$shasum" > "$action_dir/echo"

echo "Build complete"

