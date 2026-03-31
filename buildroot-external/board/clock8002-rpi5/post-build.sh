#!/bin/sh
set -eu

TARGET_DIR="$1"

# Keep network and service behavior deterministic for appliance builds.
mkdir -p "${TARGET_DIR}/etc/systemd/system/multi-user.target.wants"

# Disable ModemManager by masking it in the target image when present.
mkdir -p "${TARGET_DIR}/etc/systemd/system"
ln -sf /dev/null "${TARGET_DIR}/etc/systemd/system/ModemManager.service"

# Enable login prompt on HDMI (tty1).
mkdir -p "${TARGET_DIR}/etc/systemd/system/getty.target.wants"
ln -sf /usr/lib/systemd/system/getty@.service \
	"${TARGET_DIR}/etc/systemd/system/getty.target.wants/getty@tty1.service"
