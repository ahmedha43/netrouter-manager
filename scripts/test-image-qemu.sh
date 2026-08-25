#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT/artifacts/buildroot-x86_64}"
KERNEL="$OUTPUT_DIR/images/bzImage"
ROOTFS="$OUTPUT_DIR/images/rootfs.ext2"

for command in debugfs qemu-system-x86_64 timeout; do
  command -v "$command" >/dev/null || { echo "Missing required command: $command" >&2; exit 2; }
done
for image in "$KERNEL" "$ROOTFS"; do
  [[ -f "$image" ]] || { echo "Missing image: $image" >&2; exit 2; }
done

WORK_DIR="$(mktemp -d)"
TEST_ROOTFS="$WORK_DIR/netrouter-qemu-proof.ext2"
PROOF_SCRIPT="$WORK_DIR/S99qemu-proof"
SERIAL_LOG="$WORK_DIR/qemu.log"
cleanup() {
  if [[ "${KEEP_QEMU_WORKDIR:-0}" == "1" ]]; then
    echo "Retained QEMU test data in $WORK_DIR" >&2
  else
    rm -rf "$WORK_DIR"
  fi
}
trap cleanup EXIT

cat >"$PROOF_SCRIPT" <<'EOF'
#!/bin/sh
RESULT=/etc/netrouter-qemu-proof.result
record() { printf '%s\n' "$1" >> "$RESULT"; printf '%s\n' "$1" > /dev/console; }

record 'NETROUTER_QEMU_PROOF_BEGIN'
attempt=0
while [ ! -S /run/netrouterd.sock ] && [ "$attempt" -lt 20 ]; do
  sleep 1
  attempt=$((attempt + 1))
done

if [ ! -S /run/netrouterd.sock ]; then
  record 'NETROUTER_QEMU_PROOF_FAIL socket_missing'
  poweroff -f
  exit 1
fi

if /usr/bin/netrouterctl --socket /run/netrouterd.sock --command status >>"$RESULT" 2>&1; then
  record 'NETROUTER_QEMU_PROOF_OK socket_and_status'
else
  record 'NETROUTER_QEMU_PROOF_FAIL netrouterctl'
fi

sync
sleep 1
poweroff -f
EOF

cp --reflink=auto "$ROOTFS" "$TEST_ROOTFS"
debugfs -w -R "write $PROOF_SCRIPT /etc/init.d/S99qemu-proof" "$TEST_ROOTFS" >/dev/null 2>&1
debugfs -w -R 'set_inode_field /etc/init.d/S99qemu-proof mode 0100755' "$TEST_ROOTFS" >/dev/null 2>&1

set +e
timeout 120 qemu-system-x86_64 \
  -m 512M -display none -monitor none -serial "file:$SERIAL_LOG" -no-reboot \
  -kernel "$KERNEL" \
  -append 'root=/dev/vda rw console=ttyS0,115200 earlyprintk=serial,ttyS0,115200' \
  -drive "file=$TEST_ROOTFS,format=raw,if=virtio"
qemu_status=$?
set -e

if [[ "$qemu_status" -ne 0 ]]; then
  echo "QEMU did not shut down cleanly (status $qemu_status)." >&2
  tail -80 "$SERIAL_LOG" >&2 || true
  exit "$qemu_status"
fi

proof="$(debugfs -R 'cat /etc/netrouter-qemu-proof.result' "$TEST_ROOTFS" 2>/dev/null || true)"
printf '%s\n' "$proof"
if ! grep -q 'NETROUTER_QEMU_PROOF_OK socket_and_status' <<<"$proof"; then
  echo "NetRouter daemon proof did not succeed." >&2
  exit 1
fi
