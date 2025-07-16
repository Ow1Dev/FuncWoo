#!/bin/bash
set -euo pipefail

echo "Creating /var/lib/noctifunc/funcs directory with proper permissions..."
sudo mkdir -pv /var/lib/noctifunc/{funcs,routes}
sudo chown -R "$(id -u):$(id -g)" /var/lib/noctifunc

echo "Building noctifunc/base:latest Docker image..."
docker build -t noctifunc/base:latest -f docker/base/Dockerfile docker/base

echo "Setup complete."
