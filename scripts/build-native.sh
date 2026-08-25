#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT/artifacts/linux" "$ROOT/artifacts/windows"

pushd "$ROOT/native" >/dev/null
go test ./internal/...
go build -trimpath -ldflags="-s -w" -o "$ROOT/artifacts/linux/netrouterd" ./cmd/netrouterd
go build -trimpath -ldflags="-s -w" -o "$ROOT/artifacts/linux/netrouterctl" ./cmd/netrouterctl
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
  go build -trimpath -ldflags="-H=windowsgui -s -w" -o "$ROOT/artifacts/windows/NetRouterManager.exe" ./cmd/netrouter-manager
popd >/dev/null

echo "Native artifacts written to artifacts/linux and artifacts/windows"
