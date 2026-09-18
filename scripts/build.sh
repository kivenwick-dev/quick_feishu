#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# 1) build frontend (outputs to internal/server/web)
if [ -d frontend ]; then
  (cd frontend && npm run build)
fi

mkdir -p dist
GOOS=darwin GOARCH=arm64 go build -o dist/quick-feishu-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o dist/quick-feishu-darwin-amd64 .
GOOS=linux  GOARCH=amd64 go build -o dist/quick-feishu-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/quick-feishu-windows-amd64.exe .
echo "build complete -> dist/"
