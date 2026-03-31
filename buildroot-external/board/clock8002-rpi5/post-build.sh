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

# Enable piclock-network service (applies /boot/piclock/network.ini at boot).
ln -sf /etc/systemd/system/piclock-network.service \
	"${TARGET_DIR}/etc/systemd/system/multi-user.target.wants/piclock-network.service"

# Make piclock-network.sh executable.
chmod +x "${TARGET_DIR}/opt/clock8002/piclock-network.sh"

# Mount boot partition at /boot so network.ini is accessible at runtime.
mkdir -p "${TARGET_DIR}/boot"
if ! grep -q '/boot' "${TARGET_DIR}/etc/fstab" 2>/dev/null; then
	echo '/dev/mmcblk0p1	/boot	vfat	defaults,nofail	0	0' >> "${TARGET_DIR}/etc/fstab"
fi

# Pre-create /boot/piclock directory (will be on the mounted FAT partition).
mkdir -p "${TARGET_DIR}/boot/piclock"

# Symlink ~/.config/clock-8001/clock.ini -> /boot/piclock/clock.ini
# sdl-clock reads config from XDG config dir; production uses this symlink.
mkdir -p "${TARGET_DIR}/root/.config/clock-8001"
ln -sf /boot/piclock/clock.ini "${TARGET_DIR}/root/.config/clock-8001/clock.ini"

# Patch service files: Buildroot runs as root, not pi.
for svc in clock8002.service alsa-ltc.service; do
	F="${TARGET_DIR}/usr/lib/systemd/system/${svc}"
	[ -f "${F}" ] && sed -i 's/^User=pi$/User=root/' "${F}"
done

# Allow root SSH login with password (appliance build).
if [ -f "${TARGET_DIR}/etc/ssh/sshd_config" ]; then
	sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' "${TARGET_DIR}/etc/ssh/sshd_config"
fi
