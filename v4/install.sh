#!/bin/bash
set -e

INSTALL_DIR="/opt/clock8002"
SERVICE_FILE="clock8002.service"

echo "Clock-8002 Installer"
echo "===================="
echo ""

# Check we're on Linux arm64
if [ "$(uname -s)" != "Linux" ] || [ "$(uname -m)" != "aarch64" ]; then
    echo "Warning: This release is built for Linux arm64 (Raspberry Pi 5)."
    echo "Current system: $(uname -s) $(uname -m)"
    read -rp "Continue anyway? [y/N] " reply
    [ "$reply" != "y" ] && exit 1
fi

# Determine the user on whose behalf we are installing.
# install.sh must be run via 'sudo bash install.sh'; $USER/$HOME resolve to root
# under sudo, so we use $SUDO_USER to recover the actual invoking user.
if [ -n "${SUDO_USER:-}" ] && [ "${SUDO_USER}" != "root" ]; then
    INSTALL_USER="${SUDO_USER}"
    INSTALL_HOME="$(getent passwd "${SUDO_USER}" | cut -d: -f6)"
else
    INSTALL_USER="${USER}"
    INSTALL_HOME="${HOME}"
fi

# Stop running services before copying binaries
echo "Stopping services..."
sudo systemctl stop clock8002 alsa-ltc oled_daemon bootsplash 2>/dev/null || true

# Verify release payload integrity when checksum manifest is present
if [ -f SHA256SUMS ]; then
    echo "Verifying release checksums..."
    sha256sum -c --ignore-missing SHA256SUMS
fi

# Install SDL3 runtime libraries and LTC dependencies
echo "Installing runtime libraries..."

# Wait for any first-boot apt process (unattended-upgrades, initial setup) to release its lock.
# Fresh images often have apt-get running in the background, which causes immediate failure.
APT_WAIT_MAX=60
APT_WAIT_COUNT=0
echo "Waiting for package manager lock..."
while fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
    APT_WAIT_COUNT=$((APT_WAIT_COUNT + 1))
    if [ "$APT_WAIT_COUNT" -gt "$APT_WAIT_MAX" ]; then
        echo "ERROR: Package manager lock still held after ${APT_WAIT_MAX}0 seconds." >&2
        echo "Wait for first-boot setup to finish, then re-run install.sh." >&2
        exit 1
    fi
    echo "  package manager is busy, waiting... (${APT_WAIT_COUNT}/${APT_WAIT_MAX})"
    sleep 10
done

sudo apt update

REQUIRED_SDL3_PACKAGES=(
    libsdl3-0
    libsdl3-image0
    libsdl3-ttf0
    libgles2
    libgl1
    libegl1
    libltc11
    i2c-tools
    python3-luma.oled
    fbi
    util-linux-extra
)

if ! sudo apt install -y "${REQUIRED_SDL3_PACKAGES[@]}"; then
    echo "ERROR: Required SDL3/runtime packages could not be installed." >&2
    echo "This release requires Debian Trixie SDL3 packages." >&2
    exit 1
fi

# Verify SDL3 libraries are loadable before continuing.
# Debian/Raspberry Pi OS packages ship versioned SONAMEs (e.g. libSDL3.so.0);
# the unversioned .so symlinks are only in the -dev packages. go-sdl3 loads
# the SONAMEs at runtime, so that is what we check for here.
SDL3_LIB_DIR="/usr/lib/aarch64-linux-gnu"
for lib in libSDL3.so.0 libSDL3_image.so.0 libSDL3_ttf.so.0; do
    if [ ! -e "${SDL3_LIB_DIR}/${lib}" ]; then
        echo "ERROR: ${SDL3_LIB_DIR}/${lib} not found after package installation." >&2
        echo "Install the libsdl3-0, libsdl3-image0, and libsdl3-ttf0 packages." >&2
        exit 1
    fi
done

# Disable ModemManager on clock units unless a cellular modem is intentionally used.
if systemctl list-unit-files | grep -q '^ModemManager\.service'; then
    echo "Disabling ModemManager service..."
    sudo systemctl disable --now ModemManager 2>/dev/null || true
fi

echo "Ensuring rpi-clone is installed..."
if command -v rpi-clone >/dev/null 2>&1; then
    echo "rpi-clone already installed"
else
    curl -fsSL https://raw.githubusercontent.com/geerlingguy/rpi-clone/master/install | sudo bash
fi

# Create install directory
echo "Installing to ${INSTALL_DIR}..."
sudo mkdir -p "${INSTALL_DIR}"

# The release binary is named sdl-clock; source was built from cmd/sdl3-clock.
if [ ! -f sdl-clock ]; then
    echo "ERROR: sdl-clock binary not found in release payload." >&2
    exit 1
fi
sudo cp sdl-clock "${INSTALL_DIR}/"
sudo chmod +x "${INSTALL_DIR}/sdl-clock"

# Bundle private SDL3 3.4.0 libraries so go-sdl3 matches the runtime ABI.
# Debian Trixie ships SDL3 3.2.10, which is missing symbols required by go-sdl3.
if [ -d lib ]; then
    echo "Installing bundled SDL3 libraries..."
    sudo cp -r lib "${INSTALL_DIR}/"
    sudo chmod 755 "${INSTALL_DIR}/lib"
    sudo chmod 644 "${INSTALL_DIR}/lib"/*.so* 2>/dev/null || true
fi

sudo cp *.ttf "${INSTALL_DIR}/"
sudo cp -r fonts "${INSTALL_DIR}/"
sudo cp -r voices "${INSTALL_DIR}/"

# Install alsa-ltc if present
if [ -f alsa-ltc ]; then
    sudo cp alsa-ltc "${INSTALL_DIR}/"
    sudo chmod +x "${INSTALL_DIR}/alsa-ltc"
fi

# Add current user to video and render groups
echo "Adding $INSTALL_USER to video and render groups..."
sudo usermod -aG video,render "$INSTALL_USER"

# Install config to boot partition with symlink
# Pi OS Bookworm/Trixie: /boot/firmware is the FAT32 boot partition, accessible on Mac/PC.
BOOT_CONFIG_DIR="/boot/firmware/piclock"
CONFIG_DIR="$INSTALL_HOME/.config/clock-8001"

sudo mkdir -p "${BOOT_CONFIG_DIR}"
mkdir -p "${CONFIG_DIR}"
chown "${INSTALL_USER}:${INSTALL_USER}" "${CONFIG_DIR}"

# Allow the current user to write to the FAT32 boot partition (enables web UI config saves).
# vfat does not support per-file ownership; uid/gid mount options control effective ownership.
if grep -qE '[[:space:]]/boot/firmware[[:space:]]' /etc/fstab; then
    BOOT_UID="$(id -u "${INSTALL_USER}")"
    BOOT_GID="$(id -g "${INSTALL_USER}")"
    if ! grep -E '[[:space:]]/boot/firmware[[:space:]]' /etc/fstab | grep -q "uid=${BOOT_UID}"; then
        echo "Configuring /boot/firmware to be writable by $INSTALL_USER..."
        sudo sed -i "/[[:space:]]\/boot\/firmware[[:space:]]/ s/defaults/defaults,uid=${BOOT_UID},gid=${BOOT_GID}/" /etc/fstab
    fi
    # Apply uid/gid mount options immediately so config saves work without a reboot.
    sudo umount /boot/firmware && sudo mount /boot/firmware || true
fi

# Label the boot partition so it appears as "piClock" when mounted on Mac/PC.
if [ -b /dev/mmcblk0p1 ]; then
    sudo fatlabel /dev/mmcblk0p1 piClock 2>/dev/null || true
elif [ -b /dev/sda1 ]; then
    sudo fatlabel /dev/sda1 piClock 2>/dev/null || true
fi

# Migrate config files from legacy /boot/piclock/ (ext4) to /boot/firmware/piclock/ (FAT32).
# Previously config files were installed to /boot/piclock/ on ext4, not accessible from Mac/PC.
OLD_BOOT_DIR="/boot/piclock"
if [ -d "${OLD_BOOT_DIR}" ] && [ "${OLD_BOOT_DIR}" != "${BOOT_CONFIG_DIR}" ]; then
    echo "Migrating config files from ${OLD_BOOT_DIR} to ${BOOT_CONFIG_DIR}..."
    for ini in clock.ini network.ini oled.ini; do
        if [ -f "${OLD_BOOT_DIR}/${ini}" ] && [ ! -f "${BOOT_CONFIG_DIR}/${ini}" ]; then
            sudo cp "${OLD_BOOT_DIR}/${ini}" "${BOOT_CONFIG_DIR}/${ini}"
            echo "  Migrated ${ini}"
        fi
    done
    # Update clock.ini symlink if it still points to the old location.
    if [ -L "${CONFIG_DIR}/clock.ini" ]; then
        case "$(readlink "${CONFIG_DIR}/clock.ini")" in
            "${OLD_BOOT_DIR}/"*) ln -sf "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"; echo "  Updated clock.ini symlink" ;;
        esac
    fi
    # Update oled.ini symlink if it still points to the old location.
    if [ -L "${INSTALL_DIR}/oled/oled.ini" ]; then
        case "$(readlink "${INSTALL_DIR}/oled/oled.ini")" in
            "${OLD_BOOT_DIR}/"*) sudo ln -sf "${BOOT_CONFIG_DIR}/oled.ini" "${INSTALL_DIR}/oled/oled.ini"; echo "  Updated oled.ini symlink" ;;
        esac
    fi
    # Remove old directory if now empty.
    sudo rmdir "${OLD_BOOT_DIR}" 2>/dev/null || true
fi

if [ -L "${CONFIG_DIR}/clock.ini" ]; then
    # Already a symlink (previous install migrated it)
    echo "Config symlink already in place."
elif [ -f "${CONFIG_DIR}/clock.ini" ]; then
    # Existing config at old location — migrate to boot partition
    echo "Migrating existing config to ${BOOT_CONFIG_DIR}/clock.ini..."
    sudo cp "${CONFIG_DIR}/clock.ini" "${BOOT_CONFIG_DIR}/clock.ini"
    sudo chown "$INSTALL_USER":"$INSTALL_USER" "${BOOT_CONFIG_DIR}/clock.ini" 2>/dev/null || true
    rm "${CONFIG_DIR}/clock.ini"
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
elif [ -f "${BOOT_CONFIG_DIR}/clock.ini" ]; then
    # Config on boot partition but no symlink yet
    echo "Found config on boot partition, creating symlink..."
    sudo chown "$INSTALL_USER":"$INSTALL_USER" "${BOOT_CONFIG_DIR}/clock.ini" 2>/dev/null || true
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
else
    # Fresh install — copy default to boot partition and symlink
    echo "Installing default configuration to ${BOOT_CONFIG_DIR}/clock.ini..."
    sudo cp clock.ini "${BOOT_CONFIG_DIR}/clock.ini"
    sudo chown "$INSTALL_USER":"$INSTALL_USER" "${BOOT_CONFIG_DIR}/clock.ini" 2>/dev/null || true
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
fi

# Install systemd service
echo "Installing systemd services..."

# Install network config script and service
if [ -f piclock-network.sh ]; then
    sudo cp piclock-network.sh "${INSTALL_DIR}/"
    sudo chmod +x "${INSTALL_DIR}/piclock-network.sh"
fi
if [ -f piclock-network.service ]; then
    sudo cp piclock-network.service /etc/systemd/system/
    sudo systemctl enable piclock-network
fi

# Speed up boot: mask NetworkManager-wait-online. The clock does not need the
# network to be "online" before starting, and on a static-only wired network
# eth0's DHCP probe times out (~45s), stalling boot and delaying both the static
# IP and the address shown on the OLED. piclock-network.sh applies the network
# configuration directly, with its own device-wait retry loop.
if systemctl list-unit-files NetworkManager-wait-online.service >/dev/null 2>&1; then
    echo "Masking NetworkManager-wait-online.service to speed up boot..."
    sudo systemctl mask NetworkManager-wait-online.service 2>/dev/null || true
fi

# Copy sample network.ini to boot partition if not present
if [ ! -f "${BOOT_CONFIG_DIR}/network.ini" ] && [ -f network.ini ]; then
    echo "Installing sample network.ini to ${BOOT_CONFIG_DIR}/..."
    sudo cp network.ini "${BOOT_CONFIG_DIR}/network.ini"
    # Seed hostname from the system hostname set during OS setup (e.g. via RPi Imager).
    SYSTEM_HOSTNAME="$(hostname)"
    sudo sed -i "s/^hostname=.*/hostname=${SYSTEM_HOSTNAME}/" "${BOOT_CONFIG_DIR}/network.ini"
    echo "  Hostname set to: ${SYSTEM_HOSTNAME}"
fi

# Enable Wi-Fi radio and set regulatory domain (persistent)
echo "Configuring Wi-Fi radio..."
sudo raspi-config nonint do_wifi_country US

# Enable Pi 5 RTC battery charging at 3.0V when a backup battery is connected.
RTC_CFG="/boot/firmware/config.txt"
RTC_PARAM="dtparam=rtc_bbat_vchg=3000000"
if [ -f "${RTC_CFG}" ]; then
    if grep -q "^dtparam=rtc_bbat_vchg=" "${RTC_CFG}"; then
        echo "Updating RTC battery charging setting in ${RTC_CFG}..."
        sudo sed -i "s|^dtparam=rtc_bbat_vchg=.*|${RTC_PARAM}|" "${RTC_CFG}"
    else
        echo "Enabling RTC battery charging in ${RTC_CFG}..."
        echo "${RTC_PARAM}" | sudo tee -a "${RTC_CFG}" > /dev/null
    fi
fi

# Enable GPIO/UART serial overlays for Pi 5
CONFIG_FILE="/boot/firmware/config.txt"
UART_OVERLAYS=(
    "dtoverlay=dwc2,dr_mode=host"
    "dtoverlay=uart1"
    "dtoverlay=uart2"
    "dtoverlay=uart3"
    "dtparam=uart0=on"
)

if [ -f "${CONFIG_FILE}" ]; then
    for overlay in "${UART_OVERLAYS[@]}"; do
        if ! grep -q "^${overlay}$" "${CONFIG_FILE}"; then
            echo "Adding UART overlay: ${overlay}"
            echo "${overlay}" | sudo tee -a "${CONFIG_FILE}" > /dev/null
        fi
    done
fi

# Force HDMI output to 1920x1080 (16:9) on both connectors (issue #46).
# Pi 5 (vc4-kms-v3d/KMS) otherwise follows the display EDID native mode (up to 4K)
# and upscales the 1080p canvas. A kernel video= param is the only reliable
# override under KMS; legacy hdmi_group/hdmi_mode config.txt keys are ignored.
# cmdline.txt is a single line; append video= params idempotently.
CMDLINE_HDMI="/boot/firmware/cmdline.txt"
if [ -f "${CMDLINE_HDMI}" ]; then
    for vopt in "video=HDMI-A-1:1920x1080@60" "video=HDMI-A-2:1920x1080@60"; do
        if ! grep -Fq "${vopt}" "${CMDLINE_HDMI}"; then
            echo "Forcing 1080p: adding ${vopt} to cmdline.txt..."
            sudo sed -i "s/$/ ${vopt}/" "${CMDLINE_HDMI}"
        fi
    done
fi

sed -e "s|WorkingDirectory=.*|WorkingDirectory=${INSTALL_DIR}|" \
    -e "s|ExecStart=.*|ExecStart=${INSTALL_DIR}/sdl-clock --fullscreen --font-path=${INSTALL_DIR}|" \
    -e "s|^User=.*|User=$INSTALL_USER|" \
    "${SERVICE_FILE}" | sudo tee /etc/systemd/system/clock8002.service > /dev/null

echo "Installing restricted hwclock sudoers rule..."
printf '%s\n' \
    "Defaults:${INSTALL_USER} !requiretty" \
    "${INSTALL_USER} ALL=(root) NOPASSWD: /usr/sbin/hwclock --systohc --utc, /sbin/hwclock --systohc --utc" | \
    sudo tee /etc/sudoers.d/clock8002-hwclock > /dev/null
sudo chmod 440 /etc/sudoers.d/clock8002-hwclock
sudo visudo -cf /etc/sudoers.d/clock8002-hwclock

# Install alsa-ltc service if present
if [ -f alsa-ltc.service ]; then
    sed "s|ExecStart=.*|ExecStart=${INSTALL_DIR}/alsa-ltc - 255.255.255.255 1245|" alsa-ltc.service | \
        sed "s|User=.*|User=$INSTALL_USER|" | \
        sudo tee /etc/systemd/system/alsa-ltc.service > /dev/null
fi

# Remove legacy udev rule (alsa-ltc is now managed as a plain systemd service)
sudo rm -f /etc/udev/rules.d/99-alsa-ltc-usb.rules

# Remove stale ALSA state file — a corrupt asound.state written during a crash-loop
# will be restored by alsa-restore.service at boot, preventing alsa-ltc from opening
# the audio device. Remove it so ALSA uses clean device defaults on next boot.
sudo rm -f /var/lib/alsa/asound.state
# Mask alsa-store/alsa-restore so a future crash-loop can never write or replay a
# corrupt state file. alsa-ltc sets all hw_params programmatically; there is no
# mixer state worth preserving on these units.
sudo systemctl mask alsa-store.service alsa-restore.service 2>/dev/null || true

sudo systemctl daemon-reload
sudo systemctl enable clock8002
sudo systemctl enable alsa-ltc
sudo udevadm control --reload-rules

# Install OLED display daemon if present
if [ -d oled ]; then
    echo "Installing OLED display daemon..."
    sudo mkdir -p "${INSTALL_DIR}/oled"
    sudo cp oled/oled_daemon.py "${INSTALL_DIR}/oled/"
    sudo chmod +x "${INSTALL_DIR}/oled/oled_daemon.py"

    # Install OLED config to boot partition with symlink
    if [ -L "${INSTALL_DIR}/oled/oled.ini" ]; then
        echo "OLED config symlink already in place."
    elif [ -f "${INSTALL_DIR}/oled/oled.ini" ]; then
        echo "Migrating OLED config to ${BOOT_CONFIG_DIR}/oled.ini..."
        sudo cp "${INSTALL_DIR}/oled/oled.ini" "${BOOT_CONFIG_DIR}/oled.ini"
        sudo rm "${INSTALL_DIR}/oled/oled.ini"
        sudo ln -s "${BOOT_CONFIG_DIR}/oled.ini" "${INSTALL_DIR}/oled/oled.ini"
    elif [ -f "${BOOT_CONFIG_DIR}/oled.ini" ]; then
        echo "Found OLED config on boot partition, creating symlink..."
        sudo ln -s "${BOOT_CONFIG_DIR}/oled.ini" "${INSTALL_DIR}/oled/oled.ini"
    elif [ -f oled/oled.ini ]; then
        echo "Installing default OLED config to ${BOOT_CONFIG_DIR}/oled.ini..."
        sudo cp oled/oled.ini "${BOOT_CONFIG_DIR}/oled.ini"
        sudo ln -s "${BOOT_CONFIG_DIR}/oled.ini" "${INSTALL_DIR}/oled/oled.ini"
    fi

    # Copy splash logo to user home
    if [ -f oled/piclockLogo.bin ] && [ ! -f "$INSTALL_HOME/piclockLogo.bin" ]; then
        cp oled/piclockLogo.bin "$INSTALL_HOME/piclockLogo.bin"
    fi

    # Enable I2C if not already enabled
    if ! grep -q "^dtparam=i2c_arm=on" /boot/firmware/config.txt 2>/dev/null; then
        echo "Enabling I2C..."
        sudo raspi-config nonint do_i2c 0
    fi

    # Install OLED systemd service
    if [ -f oled/oled_daemon.service ]; then
        sed "s|ExecStart=.*|ExecStart=/usr/bin/python3 ${INSTALL_DIR}/oled/oled_daemon.py|" oled/oled_daemon.service | \
            sed "s|User=.*|User=$INSTALL_USER|" | \
            sudo tee /etc/systemd/system/oled_daemon.service > /dev/null
        sudo systemctl daemon-reload
        sudo systemctl enable oled_daemon
    fi
fi

# Install boot splash screen
if [ -d splash ]; then
    echo "Installing boot splash screen..."
    sudo cp splash/bootsplash.png "${INSTALL_DIR}/bootsplash.png"

    # Install bootsplash systemd service
    if [ -f splash/bootsplash.service ]; then
        sudo cp splash/bootsplash.service /etc/systemd/system/
        sudo systemctl daemon-reload
        sudo systemctl enable bootsplash
    fi

    # Patch cmdline.txt to suppress Pi logos and kernel messages
    CMDLINE="/boot/firmware/cmdline.txt"
    if [ -f "${CMDLINE}" ]; then
        for opt in logo.nologo quiet "loglevel=1" "console=tty3"; do
            if ! grep -q "${opt}" "${CMDLINE}"; then
                echo "Adding ${opt} to cmdline.txt..."
                sudo sed -i "s/$/ ${opt}/" "${CMDLINE}"
            fi
        done
    fi
fi

echo ""
echo "Installation complete!"
echo ""
echo "  Install directory: ${INSTALL_DIR}"
echo "  Config file:       /boot/firmware/piclock/clock.ini"
echo "  Network config:    /boot/firmware/piclock/network.ini"
echo "  Config symlink:    ~/.config/clock-8001/clock.ini"
echo "  Web UI:            http://$(hostname -I | awk '{print $1}'):8080"
echo "  Credentials:       admin / clockwork"
echo ""

# Start services
echo "Starting services..."
sudo systemctl start clock8002 2>/dev/null || true
sudo systemctl start oled_daemon 2>/dev/null || true
# alsa-ltc is started automatically by udev when a USB audio device appears

echo ""
echo "Consistency report"
echo "------------------"
if [ -x "${INSTALL_DIR}/sdl-clock" ]; then
    echo "sdl-clock version:"
    SDL_CLOCK_VERSION=$("${INSTALL_DIR}/sdl-clock" --version 2>/dev/null || true)
    if [ -n "${SDL_CLOCK_VERSION}" ]; then
        echo "${SDL_CLOCK_VERSION}"
    else
        SDL_GIT_TAG=$(strings "${INSTALL_DIR}/sdl-clock" 2>/dev/null | grep -m1 -oE 'clock\.gitTag=v[0-9]+\.[0-9]+\.[0-9]+' | sed 's/^clock\.gitTag=//')
        SDL_GIT_COMMIT=$(strings "${INSTALL_DIR}/sdl-clock" 2>/dev/null | grep -m1 -oE 'clock\.gitCommit=[0-9a-f]+' | sed 's/^clock\.gitCommit=//')
        if [ -n "${SDL_GIT_TAG}" ] || [ -n "${SDL_GIT_COMMIT}" ]; then
            echo "${INSTALL_DIR}/sdl-clock version ${SDL_GIT_TAG:-unknown} commit ${SDL_GIT_COMMIT:-unknown}"
        else
            echo "(version info not found)"
        fi
    fi
    echo "sdl-clock sha256:"
    sha256sum "${INSTALL_DIR}/sdl-clock"
fi
if [ -x "${INSTALL_DIR}/alsa-ltc" ]; then
    echo "alsa-ltc version:"
    "${INSTALL_DIR}/alsa-ltc" --version || true
    echo "alsa-ltc sha256:"
    sha256sum "${INSTALL_DIR}/alsa-ltc"
fi
if [ -f /etc/systemd/system/alsa-ltc.service ]; then
    echo "alsa-ltc.service sha256:"
    sha256sum /etc/systemd/system/alsa-ltc.service
    echo "alsa-ltc.service enabled/active:"
    systemctl is-enabled alsa-ltc 2>/dev/null || true
    systemctl is-active alsa-ltc 2>/dev/null || true
fi

echo ""
echo "Reboot to finish installation: sudo reboot"
