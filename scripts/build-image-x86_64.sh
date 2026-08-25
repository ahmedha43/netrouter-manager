#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILDROOT_DIR="${BUILDROOT_DIR:-$ROOT/.cache/buildroot-2026.05}"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT/artifacts/buildroot-x86_64}"

if [[ ! -d "$BUILDROOT_DIR" ]]; then
  echo "Buildroot is not present at $BUILDROOT_DIR" >&2
  echo "Download Buildroot 2026.05 there, then run this command again." >&2
  exit 2
fi

make -C "$BUILDROOT_DIR" O="$OUTPUT_DIR" BR2_EXTERNAL="$ROOT/platform/buildroot-external" netrouter_x86_64_defconfig
make -C "$BUILDROOT_DIR" O="$OUTPUT_DIR"

echo "Image artifacts are in $OUTPUT_DIR/images"
