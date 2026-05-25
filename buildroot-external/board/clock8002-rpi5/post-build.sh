#!/bin/sh
set -eu

TARGET_DIR="$1"
BOARD_DIR="$(dirname "$0")"
GOLDEN_DIR="${BOARD_DIR}/golden-working-card"

if [ ! -d "${GOLDEN_DIR}" ]; then
        echo "Missing golden working-card payload: ${GOLDEN_DIR}" >&2
        exit 1
fi

# Set hostname from the copied working-card payload.
if [ -f "${GOLDEN_DIR}/etc/hostname" ]; then
        cp -f "${GOLDEN_DIR}/etc/hostname" "${TARGET_DIR}/etc/hostname"
else
        echo 'piClock' > "${TARGET_DIR}/etc/hostname"
fi

# Copy the working-card runtime files directly into the target rootfs.
if [ -f "${GOLDEN_DIR}/etc/hosts" ]; then
        cp -f "${GOLDEN_DIR}/etc/hosts" "${TARGET_DIR}/etc/hosts"
fi
if [ -d "${GOLDEN_DIR}/etc/init.d" ]; then
        mkdir -p "${TARGET_DIR}/etc/init.d"
        for script in \
                                S04power-button \
                S03copy_alsa-ltc_files \
                S03copy_clock_bridge_files \
                S03copy_clock_files \
                S99alsa-ltc \
                S99clock \
                S99clock_bridge; do
                if [ -f "${GOLDEN_DIR}/etc/init.d/${script}" ]; then
                        cp -f "${GOLDEN_DIR}/etc/init.d/${script}" \
                                "${TARGET_DIR}/etc/init.d/${script}"
                fi
        done
fi
if [ -d "${GOLDEN_DIR}/root" ]; then
        mkdir -p "${TARGET_DIR}/root"
        cp -a "${GOLDEN_DIR}/root/." "${TARGET_DIR}/root/"
fi

# Remove the synthetic /opt -> /root init hooks that do not exist on the
# working card and would overwrite the copied runtime payload.
rm -f "${TARGET_DIR}/etc/init.d/S02setup-root" \
                "${TARGET_DIR}/etc/init.d/S98oled"

# Splash assets and S05bootsplash are kept in the image; whether to display
# the splash at boot is controlled by splash_enabled in /boot/piclock/piclock.ini.

# Make init.d scripts executable.
for script in \
                S04power-button \
        S03copy_alsa-ltc_files \
        S03copy_clock_bridge_files \
        S03copy_clock_files \
        S11modules \
        S12machine-id \
        S43piclock-network-prep \
        S45piclock-network \
        S49sshd-keys \
        S50sshd \
        S99alsa-ltc \
        S99clock \
        S99clock_bridge; do
        F="${TARGET_DIR}/etc/init.d/${script}"
        if [ -f "${F}" ]; then
                chmod +x "${F}"
        fi
done

# Keep the copied working-card binaries and launchers executable.
for entry in \
        sdl-clock \
        alsa-ltc \
        clock-bridge \
                power-button.sh \
        alsa-ltc_pokemon.sh \
        alsa-ltc_cmd.sh \
        clock_pokemon.sh \
        clock_cmd.sh \
        clock_bridge_pokemon.sh \
        clock_bridge_cmd.sh; do
                F="${TARGET_DIR}/root/${entry}"
                if [ -f "${F}" ]; then
                                chmod +x "${F}"
                fi
done

# Mount boot partition at /boot so clock.ini/network.ini are accessible.
mkdir -p "${TARGET_DIR}/boot"
if ! grep -q '/boot' "${TARGET_DIR}/etc/fstab" 2>/dev/null; then
        echo '/dev/mmcblk0p1    /boot   vfat    defaults,nofail 0       0' >> "${TARGET_DIR}/etc/fstab"
fi

# The working card keeps /root persistent; remove any synthetic tmpfs mapping.
sed -i '/[[:space:]]\/root[[:space:]]/d' "${TARGET_DIR}/etc/fstab"

# Mount tmpfs over /var/lib so seedrng can write runtime data.
# squashfs rootfs is read-only; these dirs are regenerated each boot.
if ! grep -q 'tmpfs.*\s/var/lib\s' "${TARGET_DIR}/etc/fstab" 2>/dev/null; then
        echo 'tmpfs           /var/lib        tmpfs   mode=0755,noatime       0       0' >> "${TARGET_DIR}/etc/fstab"
fi

# Pre-create /boot/piclock directory (will be on the mounted FAT partition).
mkdir -p "${TARGET_DIR}/boot/piclock"

# Allow root SSH login with password (appliance build).
if [ -f "${TARGET_DIR}/etc/ssh/sshd_config" ]; then
        sed -i 's/^#*PermitRootLogin.*/PermitRootLogin yes/' "${TARGET_DIR}/etc/ssh/sshd_config"
fi

# Redirect SSH host keys to /root/ssh.
if [ -f "${TARGET_DIR}/etc/ssh/sshd_config" ]; then
        sed -i '/^[[:space:]]*HostKey[[:space:]]/d' "${TARGET_DIR}/etc/ssh/sshd_config"
        printf '\n# Host keys live in /root/ssh. S49sshd-keys populates from /boot/piclock/ssh/.\nHostKey /root/ssh/ssh_host_rsa_key\nHostKey /root/ssh/ssh_host_ecdsa_key\nHostKey /root/ssh/ssh_host_ed25519_key\n' >> "${TARGET_DIR}/etc/ssh/sshd_config"
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

# Default: use prebuilt kernel payload. Custom kernel compile is
# opt-out via CLOCK8002_PREBUILT_KERNEL=0.
if [ "${CLOCK8002_PREBUILT_KERNEL:-1}" != "0" ]; then
        PREBUILT_STORE="${CLOCK8002_PREBUILT_KERNEL_STORE:-/srv/clock8002/prebuilt-kernel-bundles}"
        DEFAULT_BUNDLE="${PREBUILT_STORE}/current"
        BUNDLE_DIR="${CLOCK8002_PREBUILT_KERNEL_BUNDLE:-${DEFAULT_BUNDLE}}"
        if [ -z "${BUNDLE_DIR}" ] || [ ! -d "${BUNDLE_DIR}" ]; then
                echo "Payload mode requires prebuilt kernel bundle: ${BUNDLE_DIR}" >&2
                exit 1
        fi

        if [ -f "${BUNDLE_DIR}/SHA256SUMS" ]; then
                (cd "${BUNDLE_DIR}" && sha256sum -c SHA256SUMS >/dev/null)
        fi

        MODULES_SRC=""
        if [ -d "${BUNDLE_DIR}/modules/lib/modules" ]; then
                MODULES_SRC="${BUNDLE_DIR}/modules/lib/modules"
        elif [ -d "${BUNDLE_DIR}/modules/lib" ]; then
                MODULES_SRC="${BUNDLE_DIR}/modules/lib"
        elif [ -d "${BUNDLE_DIR}/modules" ]; then
                MODULES_SRC="${BUNDLE_DIR}/modules"
        fi

        if [ -z "${MODULES_SRC}" ]; then
                echo "Plan B bundle missing modules directory (expected modules/ or modules/lib/modules): ${BUNDLE_DIR}" >&2
                exit 1
        fi

        rm -rf "${TARGET_DIR}/lib/modules"
        mkdir -p "${TARGET_DIR}/lib/modules"
        cp -a "${MODULES_SRC}/." "${TARGET_DIR}/lib/modules/"
        # Promoted bundles are read-only by convention; make copied target
        # modules writable so Buildroot cleanup steps can remove stale files.
        chmod -R u+w "${TARGET_DIR}/lib/modules" 2>/dev/null || true
        echo "Payload mode: injected prebuilt kernel modules from ${MODULES_SRC}" >&2
fi

# Enable ALSA extended name hints so that hw:CARD= devices appear in
# snd_device_name_hint() enumeration.  Without this, Buildroot's minimal
# alsa-lib only lists default/sysdefault/front — not hw: — and alsa-ltc's
# auto-detect fails to find capture devices.
cat > "${TARGET_DIR}/etc/asound.conf" <<'EOF'
defaults.namehint.extended on
EOF
