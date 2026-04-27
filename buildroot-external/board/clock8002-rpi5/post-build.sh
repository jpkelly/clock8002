#!/bin/sh
set -eu

TARGET_DIR="$1"

# Set hostname.
echo 'piClock' > "${TARGET_DIR}/etc/hostname"

# Make init.d scripts executable.
for script in \
        S02setup-root \
        S03copy_alsa-ltc_files \
        S03copy_clock_files \
        S04power-button \
        S05bootsplash \
        S11modules \
        S45piclock-network \
        S98oled \
        S99alsa-ltc \
        S99clock; do
        F="${TARGET_DIR}/etc/init.d/${script}"
        if [ -f "${F}" ]; then
                chmod +x "${F}"
        fi
done

# Make pokemon/cmd scripts executable (now in /opt/clock8002/, copied to /root/ at boot).
for script in \
        alsa-ltc_pokemon.sh \
        alsa-ltc_cmd.sh \
        clock_pokemon.sh \
        clock_cmd.sh \
        power-button.sh; do
        F="${TARGET_DIR}/opt/clock8002/${script}"
        if [ -f "${F}" ]; then
                chmod +x "${F}"
        fi
done

# Mount boot partition at /boot so clock.ini/network.ini are accessible.
mkdir -p "${TARGET_DIR}/boot"
if ! grep -q '/boot' "${TARGET_DIR}/etc/fstab" 2>/dev/null; then
        echo '/dev/mmcblk0p1    /boot   vfat    defaults,nofail 0       0' >> "${TARGET_DIR}/etc/fstab"
fi

# Mount /root as tmpfs to protect the SD card from write wear.
# Appended here (not via overlay fstab) so the Buildroot-generated fstab
# entries for /proc, /sys, /run, /tmp etc. are preserved intact.
if ! grep -q 'tmpfs.*\s/root\s' "${TARGET_DIR}/etc/fstab" 2>/dev/null; then
        echo 'tmpfs           /root           tmpfs   mode=0700,noatime       0       0' >> "${TARGET_DIR}/etc/fstab"
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

# Allow root SSH login with password (appliance build).
if [ -f "${TARGET_DIR}/etc/ssh/sshd_config" ]; then
        sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' "${TARGET_DIR}/etc/ssh/sshd_config"
fi

# SSH authorized key (dev builds only).
# Always remove any stale key from prior builds first — output/target is not
# wiped by clock8002-dirclean, so an old key can persist across rebuilds.
# Only inject a key if BR2_PICLOCKKEY is explicitly set at build time.
# Release builds: leave BR2_PICLOCKKEY unset — no key is baked into the image.
rm -f "${TARGET_DIR}/root/.ssh/authorized_keys"
if [ -n "${BR2_PICLOCKKEY:-}" ]; then
        mkdir -p "${TARGET_DIR}/root/.ssh"
        echo "${BR2_PICLOCKKEY}" > "${TARGET_DIR}/root/.ssh/authorized_keys"
        chmod 700 "${TARGET_DIR}/root/.ssh"
        chmod 600 "${TARGET_DIR}/root/.ssh/authorized_keys"
fi

# Enable ALSA extended name hints so that hw:CARD= devices appear in
# snd_device_name_hint() enumeration.  Without this, Buildroot's minimal
# alsa-lib only lists default/sysdefault/front — not hw: — and alsa-ltc's
# auto-detect fails to find capture devices.
cat > "${TARGET_DIR}/etc/asound.conf" <<'EOF'
defaults.namehint.extended on
EOF
