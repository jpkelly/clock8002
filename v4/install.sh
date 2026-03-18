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

# Install SDL2 runtime libraries and LTC dependencies
echo "Installing runtime libraries..."
sudo apt update
sudo apt install -y libsdl2-2.0-0 libsdl2-gfx-1.0-0 libsdl2-image-2.0-0 libsdl2-ttf-2.0-0 libsdl2-mixer-2.0-0 libgles2 libgl1 libegl1 libltc11 i2c-tools python3-pip python3-venv

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
BOOT_CONFIG_DIR="/boot/firmware/piclock"
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
    rm "${CONFIG_DIR}/clock.ini"
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
elif [ -f "${BOOT_CONFIG_DIR}/clock.ini" ]; then
    # Config on boot partition but no symlink yet
    echo "Found config on boot partition, creating symlink..."
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
else
    # Fresh install — copy default to boot partition and symlink
    echo "Installing default configuration to ${BOOT_CONFIG_DIR}/clock.ini..."
    sudo cp clock.ini "${BOOT_CONFIG_DIR}/clock.ini"
    ln -s "${BOOT_CONFIG_DIR}/clock.ini" "${CONFIG_DIR}/clock.ini"
fi

# Install systemd service
echo "Installing systemd services..."
sed "s|WorkingDirectory=.*|WorkingDirectory=${INSTALL_DIR}|" "${SERVICE_FILE}" | \
    sed "s|ExecStart=.*|ExecStart=${INSTALL_DIR}/sdl-clock --fullscreen|" | \
    sed "s|User=.*|User=$USER|" | \
    sudo tee /etc/systemd/system/clock8002.service > /dev/null

# Install alsa-ltc service if present
if [ -f alsa-ltc.service ]; then
    sed "s|ExecStart=.*|ExecStart=${INSTALL_DIR}/alsa-ltc - 127.0.0.1 1245|" alsa-ltc.service | \
        sed "s|User=.*|User=$USER|" | \
        sudo tee /etc/systemd/system/alsa-ltc.service > /dev/null
fi

sudo systemctl daemon-reload
sudo systemctl enable clock8002

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

    # Install Python dependencies
    echo "Installing OLED Python dependencies..."
    pip3 install --break-system-packages luma.oled Pillow 2>/dev/null || \
        pip3 install luma.oled Pillow

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

echo ""
echo "Installation complete!"
echo ""
echo "  Install directory: ${INSTALL_DIR}"
echo "  Config file:       /boot/firmware/piclock/clock.ini"
echo "  Config symlink:    ~/.config/clock-8001/clock.ini"
echo "  Web UI:            http://$(hostname -I | awk '{print $1}'):8080"
echo "  Credentials:       admin / clockwork"
echo ""
echo "Start the clock now with:"
echo "  sudo systemctl start clock8002"
echo ""
echo "For LTC timecode input (requires USB audio interface):"
echo "  sudo systemctl enable alsa-ltc"
echo "  sudo systemctl start alsa-ltc"
echo ""
echo "OLED display (if installed):"
echo "  sudo systemctl start oled_daemon"
echo ""
echo "  sudo systemctl start alsa-ltc"
echo ""
