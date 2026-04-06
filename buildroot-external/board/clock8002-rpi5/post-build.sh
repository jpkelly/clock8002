#!/bin/sh
set -eu

TARGET_DIR="$1"

# Keep network and service behavior deterministic for appliance builds.
mkdir -p "${TARGET_DIR}/etc/systemd/system/multi-user.target.wants"

# Disable ModemManager by masking it in the target image when present.
mkdir -p "${TARGET_DIR}/etc/systemd/system"
ln -sf /dev/null "${TARGET_DIR}/etc/systemd/system/ModemManager.service"

# Mask systemd-networkd — NetworkManager is the sole network manager.
# networkd conflicts with NM (dual DHCP leases, ignores NM keyfiles).
ln -sf /dev/null "${TARGET_DIR}/etc/systemd/system/systemd-networkd.service"
ln -sf /dev/null "${TARGET_DIR}/etc/systemd/system/systemd-networkd.socket"
ln -sf /dev/null "${TARGET_DIR}/etc/systemd/system/systemd-networkd-wait-online.service"
rm -f "${TARGET_DIR}/etc/systemd/network/"*.network

# Disable mDNS in systemd-resolved — avahi-daemon handles mDNS exclusively.
mkdir -p "${TARGET_DIR}/etc/systemd/resolved.conf.d"
cat > "${TARGET_DIR}/etc/systemd/resolved.conf.d/no-mdns.conf" <<'EOF'
[Resolve]
MulticastDNS=no
EOF

# Set hostname.
echo 'piclockBR' > "${TARGET_DIR}/etc/hostname"

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

# Symlink oled.ini so daemon finds config relative to its own directory.
mkdir -p "${TARGET_DIR}/opt/clock8002/oled"
ln -sf /boot/piclock/oled.ini "${TARGET_DIR}/opt/clock8002/oled/oled.ini"

# Enable oled_daemon service.
ln -sf /usr/lib/systemd/system/oled_daemon.service \
	"${TARGET_DIR}/etc/systemd/system/multi-user.target.wants/oled_daemon.service"

# Enable bootsplash service (fbv-based, shows splash until clock8002 takes over).
ln -sf /usr/lib/systemd/system/bootsplash.service \
	"${TARGET_DIR}/etc/systemd/system/multi-user.target.wants/bootsplash.service"

# Patch service files: Buildroot runs as root, not pi.
# Also ensure stdout/stderr go to journal for crash diagnostics.
for svc in clock8002.service alsa-ltc.service oled_daemon.service; do
	F="${TARGET_DIR}/usr/lib/systemd/system/${svc}"
	if [ -f "${F}" ]; then
		sed -i 's/^User=pi$/User=root/' "${F}"
		grep -q '^StandardOutput=' "${F}" || sed -i '/^\[Service\]/a StandardOutput=journal\nStandardError=journal' "${F}"
	fi
done

# Allow root SSH login with password (appliance build).
if [ -f "${TARGET_DIR}/etc/ssh/sshd_config" ]; then
	sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' "${TARGET_DIR}/etc/ssh/sshd_config"
fi

# Install SSH authorized key for passwordless root login (dev builds only).
# Set BR2_PICLOCKKEY to a public key string before building to inject it.
# Release builds: leave BR2_PICLOCKKEY unset — no key is baked into the image.
if [ -n "${BR2_PICLOCKKEY:-}" ]; then
        mkdir -p "${TARGET_DIR}/root/.ssh"
        echo "${BR2_PICLOCKKEY}" > "${TARGET_DIR}/root/.ssh/authorized_keys"
        chmod 700 "${TARGET_DIR}/root/.ssh"
        chmod 600 "${TARGET_DIR}/root/.ssh/authorized_keys"
fi

# Mask depmod.service — the build-time depmod (host-kmod with +XZ) produces
# correct modules.alias for .ko.xz modules. The target's kmod may lack XZ
# support, so a boot-time depmod -a would overwrite the good file with an
# empty one. Masking preserves the build-time output.
ln -sf /dev/null "${TARGET_DIR}/etc/systemd/system/depmod.service"

# Enable ALSA extended name hints so that hw:CARD= devices appear in
# snd_device_name_hint() enumeration.  Without this, Buildroot's minimal
# alsa-lib only lists default/sysdefault/front — not hw: — and alsa-ltc's
# auto-detect fails to find capture devices.
cat > "${TARGET_DIR}/etc/asound.conf" <<'EOF'
defaults.namehint.extended on
EOF
