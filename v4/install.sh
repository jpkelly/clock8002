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

# Stop running services before copying binaries
echo "Stopping services..."
sudo systemctl stop clock8002 alsa-ltc oled_daemon bootsplash 2>/dev/null || true

# Verify release payload integrity when checksum manifest is present
if [ -f SHA256SUMS ]; then
    echo "Verifying release checksums..."
    sha256sum -c --ignore-missing SHA256SUMS
fi

# Install SDL2 runtime libraries and LTC dependencies
echo "Installing runtime libraries..."
sudo apt update
sudo apt install -y libsdl2-2.0-0 libsdl2-gfx-1.0-0 libsdl2-image-2.0-0 libsdl2-ttf-2.0-0 libsdl2-mixer-2.0-0 libgles2 libgl1 libegl1 libltc11 i2c-tools python3-luma.oled fbi util-linux-extra

echo "Ensuring rpi-clone is installed..."
if command -v rpi-clone >/dev/null 2>&1; then
    echo "rpi-clone already installed"
else
    curl -fsSL https://raw.githubusercontent.com/geerlingguy/rpi-clone/master/install | sudo bash
fi

# Create install directory
echo "Installing to ${INSTALL_DIR}..."
sudo mkdir -p "${INSTALL_DIR}"
sudo cp sdl-clock "${INSTALL_DIR}/"
sudo cp *.ttf "${INSTALL_DIR}/"
sudo cp -r fonts "${INSTALL_DIR}/"
sudo cp -r voices "${INSTALL_DIR}/"
sudo chmod +x "${INSTALL_DIR}/sdl-clock"

# Install alsa-ltc if present
if [ -f alsa-ltc ]; then
    sudo cp alsa-ltc "${INSTALL_DIR}/"
    sudo chmod +x "${INSTALL_DIR}/alsa-ltc"
fi

# Add current user to video and render groups
echo "Adding $USER to video and render groups..."
sudo usermod -aG video,render "$USER"

# Install config to boot partition with symlink
BOOT_CONFIG_DIR="/boot/piclock"
CONFIG_DIR="$HOME/.config/clock-8001"

sudo mkdir -p "${BOOT_CONFIG_DIR}"
mkdir -p "${CONFIG_DIR}"

if [ -L "${CONFIG_DIR}/clock.ini" ]; then
    # Already a symlink (previous install migrated it)
    echo "Config symlink already in place."
elif [ -f "${CONFIG_DIR}/clock.ini" ]; then
    # Existing config at old location — migrate to boot partition
    echo "Migrating existing config to ${BOOT_CONFIG_DIR}/clock.ini..."
    sudo cp "${CONFIG_DIR}/clock.ini" "${BOOT_CONFIG_DIR}/clock.ini"
    sudo chown "$USER":"$USER" "${BOOT_CONFIG_DIR}/clock.ini"
    rm "${CONFIG_DIR}/clock.ini"
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
elif [ -f "${BOOT_CONFIG_DIR}/clock.ini" ]; then
    # Config on boot partition but no symlink yet
    echo "Found config on boot partition, creating symlink..."
    sudo chown "$USER":"$USER" "${BOOT_CONFIG_DIR}/clock.ini"
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
else
    # Fresh install — copy default to boot partition and symlink
    echo "Installing default configuration to ${BOOT_CONFIG_DIR}/clock.ini..."
    sudo cp clock.ini "${BOOT_CONFIG_DIR}/clock.ini"
    sudo chown "$USER":"$USER" "${BOOT_CONFIG_DIR}/clock.ini"
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

# Copy sample network.ini to boot partition if not present
if [ ! -f "${BOOT_CONFIG_DIR}/network.ini" ] && [ -f network.ini ]; then
    echo "Installing sample network.ini to ${BOOT_CONFIG_DIR}/..."
    sudo cp network.ini "${BOOT_CONFIG_DIR}/network.ini"
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

sed "s|WorkingDirectory=.*|WorkingDirectory=${INSTALL_DIR}|" "${SERVICE_FILE}" | \
    sed "s|ExecStart=.*|ExecStart=${INSTALL_DIR}/sdl-clock --fullscreen|" | \
    sed "s|User=.*|User=$USER|" | \
    sudo tee /etc/systemd/system/clock8002.service > /dev/null

echo "Installing restricted hwclock sudoers rule..."
printf '%s\n' \
    "Defaults:${USER} !requiretty" \
    "${USER} ALL=(root) NOPASSWD: /usr/sbin/hwclock --systohc --utc, /sbin/hwclock --systohc --utc" | \
    sudo tee /etc/sudoers.d/clock8002-hwclock > /dev/null
sudo chmod 440 /etc/sudoers.d/clock8002-hwclock
sudo visudo -cf /etc/sudoers.d/clock8002-hwclock

# Install alsa-ltc service if present
if [ -f alsa-ltc.service ]; then
    sed "s|ExecStart=.*|ExecStart=${INSTALL_DIR}/alsa-ltc - 127.0.0.1 1245|" alsa-ltc.service | \
        sed "s|User=.*|User=$USER|" | \
        sudo tee /etc/systemd/system/alsa-ltc.service > /dev/null
fi

sudo systemctl daemon-reload
sudo systemctl enable clock8002
if [ -f /etc/systemd/system/alsa-ltc.service ]; then
    sudo systemctl enable alsa-ltc
fi

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
    if [ -f oled/piclockLogo.bin ] && [ ! -f "$HOME/piclockLogo.bin" ]; then
        cp oled/piclockLogo.bin "$HOME/piclockLogo.bin"
    fi

    # Enable I2C if not already enabled
    if ! grep -q "^dtparam=i2c_arm=on" /boot/firmware/config.txt 2>/dev/null; then
        echo "Enabling I2C..."
        sudo raspi-config nonint do_i2c 0
    fi

    # Install OLED systemd service
    if [ -f oled/oled_daemon.service ]; then
        sed "s|ExecStart=.*|ExecStart=/usr/bin/python3 ${INSTALL_DIR}/oled/oled_daemon.py|" oled/oled_daemon.service | \
            sed "s|User=.*|User=$USER|" | \
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
echo "  Config file:       /boot/piclock/clock.ini"
echo "  Network config:    /boot/piclock/network.ini"
echo "  Config symlink:    ~/.config/clock-8001/clock.ini"
echo "  Web UI:            http://$(hostname -I | awk '{print $1}'):8080"
echo "  Credentials:       admin / clockwork"
echo ""

# Start services
echo "Starting services..."
sudo systemctl start clock8002 2>/dev/null || true
sudo systemctl start oled_daemon 2>/dev/null || true
sudo systemctl start alsa-ltc 2>/dev/null || true

echo ""
echo "Consistency report"
echo "------------------"
if [ -x "${INSTALL_DIR}/sdl-clock" ]; then
    echo "sdl-clock version:"
    SDL_CLOCK_VERSION="$("${INSTALL_DIR}/sdl-clock" --version 2>/dev/null || true)"
    if [ -n "${SDL_CLOCK_VERSION}" ]; then
        echo "${SDL_CLOCK_VERSION}"
    else
        SDL_GIT_TAG="$(strings "${INSTALL_DIR}/sdl-clock" 2>/dev/null | grep -m1 -oE 'clock\.gitTag=v[0-9]+\.[0-9]+\.[0-9]+' | sed 's/^clock\.gitTag=//')"
        SDL_GIT_COMMIT="$(strings "${INSTALL_DIR}/sdl-clock" 2>/dev/null | grep -m1 -oE 'clock\.gitCommit=[0-9a-f]+' | sed 's/^clock\.gitCommit=//')"
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
