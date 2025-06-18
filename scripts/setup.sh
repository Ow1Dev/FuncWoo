#!/bin/bash
set -euo pipefail

sudo mkdir -pv /var/lib/funcwoo/funcs
sudo chown -R "$(id -u):$(id -g)" /var/lib/funcwoo
