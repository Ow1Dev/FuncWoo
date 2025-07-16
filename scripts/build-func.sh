#!/bin/bash
set -euo pipefail

build-func() {
  echo "Starting build for: $1"
  temp_dir=$(mktemp -d)
  echo "Using temporary directory: $temp_dir"

  echo "Building Go binary (CGO_ENABLED=0, GOOS=linux)..."
  CGO_ENABLED=0 GOOS=linux go build -o "$temp_dir/main" $1/main.go 

  echo "Computing single SHA256 for folder contents..."
  shasum=$(find "$temp_dir" -type f -print0 | sort -z | xargs -0 cat | sha256sum | awk '{print $1}')
  echo "Checksum: $shasum"

  funcs_dir="/var/lib/noctifunc/funcs/${shasum}"
  action_dir="/var/lib/noctifunc/action"
  routes_dir="/var/lib/noctifunc/routes"

  echo "Creating target directory"
  mkdir -p "$funcs_dir"

  echo "Moving files to target directory..."
  mv "$temp_dir"/* "$funcs_dir/"

  echo "Cleaning up temporary directory..."
  rm -rf "$temp_dir"

  echo "Setting up action reference"
  mkdir -p "$action_dir"
  echo "$shasum" > "$action_dir/$2"

  echo "Setting up route conifg"
  cp ./configs/routes/$2.yml "$routes_dir/$2.yml"

  echo "Build complete"

}

build-func "./examples/echo" "echo"
build-func "./examples/hello" "hello"
