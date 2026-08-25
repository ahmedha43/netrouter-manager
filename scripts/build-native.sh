#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT/artifacts/linux" "$ROOT/artifacts/windows"

# 1. Build Linux Daemon and CLI Control Tool (Go)
pushd "$ROOT/native" >/dev/null
go test ./internal/...
go build -trimpath -ldflags="-s -w" -o "$ROOT/artifacts/linux/netrouterd" ./cmd/netrouterd
go build -trimpath -ldflags="-s -w" -o "$ROOT/artifacts/linux/netrouterctl" ./cmd/netrouterctl
popd >/dev/null

# 2. Build 100% Native C++ Win32 WinBox Desktop Client
if command -v x86_64-w64-mingw32-g++ >/dev/null 2>&1; then
  echo "Compiling native C++ Win32 WinBox Client..."
  x86_64-w64-mingw32-g++ -std=c++20 -O2 -mwindows -s -w \
    -o "$ROOT/artifacts/windows/NetRouterManager.exe" \
    "$ROOT/native-winbox/main.cpp" \
    -lcomctl32 -lws2_32 -lgdi32 -luser32
else
  echo "Fallback building Go client..."
  pushd "$ROOT/native" >/dev/null
  CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    go build -trimpath -ldflags="-H=windowsgui -s -w" -o "$ROOT/artifacts/windows/NetRouterManager.exe" ./cmd/netrouter-manager
  popd >/dev/null
fi

echo "Native artifacts written to artifacts/linux and artifacts/windows"
