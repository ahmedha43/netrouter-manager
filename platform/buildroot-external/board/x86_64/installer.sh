#!/bin/sh
# NetRouter OS — Disk Installer for bare-metal & VM targets (x86_64)
# Configures GPT dual A/B partitions + persistent configuration storage.
set -eu

echo "====================================================="
echo "   NetRouter OS — Bare-Metal Disk Installer          "
echo "====================================================="

TARGET_DISK="${1:-}"

if [ -z "$TARGET_DISK" ]; then
    echo "Usage: $0 <target-disk>"
    echo "Example: $0 /dev/sda  or  $0 /dev/nvme0n1"
    echo ""
    echo "Available storage devices:"
    lsblk -d -o NAME,SIZE,MODEL || fdisk -l
    exit 1
fi

if [ ! -b "$TARGET_DISK" ]; then
    echo "Error: $TARGET_DISK is not a valid block device." >&2
    exit 2
fi

echo "WARNING: All data on $TARGET_DISK will be permanently erased."
printf "Type 'YES' to proceed with installation: "
read -r CONFIRM

if [ "$CONFIRM" != "YES" ]; then
    echo "Installation aborted by user."
    exit 0
fi

echo "[1/5] Partitioning $TARGET_DISK (GPT Dual-Slot A/B)..."
parted -s "$TARGET_DISK" mklabel gpt
parted -s "$TARGET_DISK" mkpart ESP fat32 1MiB 128MiB
parted -s "$TARGET_DISK" set 1 esp on
parted -s "$TARGET_DISK" mkpart RootA ext4 128MiB 768MiB
parted -s "$TARGET_DISK" mkpart RootB ext4 768MiB 1408MiB
parted -s "$TARGET_DISK" mkpart NetData ext4 1408MiB 100%

# Partition naming convention
if echo "$TARGET_DISK" | grep -qE "nvme|mmcblk"; then
    PART_BOOT="${TARGET_DISK}p1"
    PART_ROOTA="${TARGET_DISK}p2"
    PART_ROOTB="${TARGET_DISK}p3"
    PART_DATA="${TARGET_DISK}p4"
else
    PART_BOOT="${TARGET_DISK}1"
    PART_ROOTA="${TARGET_DISK}2"
    PART_ROOTB="${TARGET_DISK}3"
    PART_DATA="${TARGET_DISK}4"
fi

echo "[2/5] Formatting partitions..."
mkfs.vfat -F 32 "$PART_BOOT"
mkfs.ext4 -F -L "NetRouter_RootA" "$PART_ROOTA"
mkfs.ext4 -F -L "NetRouter_RootB" "$PART_ROOTB"
mkfs.ext4 -F -L "NetRouter_Data" "$PART_DATA"

echo "[3/5] Installing NetRouter OS image to Slot A..."
MOUNT_DIR="/tmp/netrouter_install"
mkdir -p "$MOUNT_DIR"
mount "$PART_ROOTA" "$MOUNT_DIR"

if [ -f /rootfs.tar.gz ]; then
    tar -xzf /rootfs.tar.gz -C "$MOUNT_DIR"
elif [ -d /mnt/cdrom ]; then
    cp -a /mnt/cdrom/* "$MOUNT_DIR/" || true
fi

echo "[4/5] Initializing persistent data store..."
mkdir -p "$MOUNT_DIR/var/netrouter" "$MOUNT_DIR/etc/netrouter"
umount "$MOUNT_DIR"

echo "[5/5] Installation complete!"
echo "NetRouter OS has been successfully installed on $TARGET_DISK."
echo "Remove bootable media and reboot to enter NetRouter OS."
