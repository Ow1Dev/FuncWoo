#!/bin/bash
set -euo pipefail

mkdir temp
go build -o temp/echo ./examples/echo/main.go

shasum=$(sha256sum temp/echo | awk '{print $1}')

mkdir /var/lib/funcwoo/funcs/${shasum}
mv temp/echo /var/lib/funcwoo/funcs/${shasum}/echo

rm -rf temp
