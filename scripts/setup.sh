#!/bin/bash
set -euo pipefail

echo "Creating /var/lib/funcwoo/funcs directory with proper permissions..."
sudo mkdir -pv /var/lib/funcwoo/funcs
sudo chown -R "$(id -u):$(id -g)" /var/lib/funcwoo

echo "Building funcwoo/base:latest Docker image..."
docker build -t funcwoo/base:latest -f docker/base/Dockerfile docker/base

echo "Setup complete."
