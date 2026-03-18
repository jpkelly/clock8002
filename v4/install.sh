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
sudo apt install -y libsdl2-2.0-0 libsdl2-gfx-1.0-0 libsdl2-image-2.0-0 libsdl2-ttf-2.0-0 libsdl2-mixer-2.0-0 libgles2 libgl1 libegl1 libltc11

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

# Install default config if none exists
CONFIG_DIR="$HOME/.config/clock-8001"
if [ ! -f "${CONFIG_DIR}/clock.ini" ]; then
    echo "Installing default configuration..."
    mkdir -p "${CONFIG_DIR}"
    cp clock.ini "${CONFIG_DIR}/clock.ini"
else
    echo "Existing config found at ${CONFIG_DIR}/clock.ini — not overwriting."
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

echo ""
echo "Installation complete!"
echo ""
echo "  Install directory: ${INSTALL_DIR}"
echo "  Config file:       ~/.config/clock-8001/clock.ini (created on first run)"
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
