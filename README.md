# Clock 8002

An HDMI clock display for Raspberry Pi 5, based on the [clock-8001](https://gitlab.com/clock-8001/clock-8001) project. Outputs a full-screen clock to HDMI via SDL2 with KMSDRM (no desktop environment required). Controllable via OSC and a built-in web UI.

This build is intended to be used with the piClock platform.

## Table of Contents

- [Acknowledgements](#acknowledgements)
- [What changed from clock-8001](#what-changed-from-clock-8001)
- [Requirements](#requirements)
- [Quick Install (pre-built binary)](#quick-install-pre-built-binary)
- [GPIO/UART Serial Connections](#gpiouart-serial-connections)
- [Building from Source](#building-from-source)
- [Service Operations](#service-operations)
- [Updating](#updating)
- [Cloning the Image](#cloning-the-image)
- [Configuration](#configuration)
- [Platform](#platform)
- [OSC Control](#osc-control)
- [Troubleshooting](#troubleshooting)
- [Creating a Release](#creating-a-release)
- [License](#license)

## Acknowledgements

This project is a fork of **clock-8001** by Depili, developed in co-operation with Daniel Richert and with grants from FUUG — Finnish Unix User Group. The original project is available at https://gitlab.com/clock-8001/clock-8001 and is licensed under the GNU General Public License v2.

Please consider supporting the original clock-8001 development: https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR

## What changed from clock-8001

- Runtime: runs headless via KMSDRM (no X11/Wayland)
- Features: added "quad" and "dual" text clock faces
- Features: added LTC timecode input via ALSA audio (`alsa-ltc`)
- Features: retained GPIO pulse output support (`periph.io`)
- Platform: targets Raspberry Pi 5 running 64-bit Debian Trixie (arm64)
- Removed hardware: HUB75 LED matrix, Arduino LED ring, Pimoroni Unicorn HD, Futaba VFD

## Requirements

- Raspberry Pi 5 running Raspberry Pi OS / Debian Trixie (64-bit)
- HDMI display
- Network connection (for OSC control and web configuration)

## Quick Install (pre-built binary)

If a release tarball is available, you can install without compiling:

### 1. Download the latest release

```bash
wget https://github.com/jpkelly/clock8002/releases/download/v0.2.7/clock8002-v0.2.7-default-linux-arm64.tar.gz
```

### 2. Extract and install

```bash
tar xzf clock8002-v0.2.7-default-linux-arm64.tar.gz
cd clock8002-v0.2.7-default-linux-arm64
./install.sh
```

The install script will:
- Install SDL2, Mesa GL, LTC, and OLED runtime libraries
- Copy the binary, alsa-ltc, OLED daemon, and assets to `/opt/clock8002`
- Install a default `clock.ini` config (quad face with world clocks)
- Set up OLED display daemon (SSD1306 I2C) if present
- Enable I2C for OLED
- Add your user to video/render groups
- Install and enable the systemd services

The installer starts the clock service automatically. Then open `http://<pi-ip>:8080` to configure (default: **admin** / **clockwork**).

### Pre-boot Network Configuration (optional)

Edit `/boot/piclock/network.ini` on the SD card's FAT32 boot partition (mountable on Mac/PC) to configure the Pi without SSH. Settings apply automatically on boot via the `piclock-network` service.

Example `network.ini` with common configurations:

```ini
# Clock-8002 Network Configuration
# Place this file at /boot/piclock/network.ini and reboot to apply

[network]
mode=dhcp                          # dhcp or static
ntp=false                          # Disable if using OSC settime
# address=10.0.0.100               # Static IP only
# netmask=24
# gateway=10.0.0.1
# dns=1.1.1.1

[host]
hostname=piClock                   # Hostname (without .local)

[wifi]
ap_enabled=true                    # Broadcast Wi-Fi access point
ap_ssid=piClock-ap                 # SSID (defaults to <hostname>-ap)
ap_password=clockwork              # Password (min 8 characters)
ap_channel=6                       # Wi-Fi channel
```

**Notes:**
- Edit directly or uncomment lines by removing `#`
- To apply changes without rebooting: `sudo /opt/clock8002/piclock-network.sh`
- NTP defaults to disabled so OSC `settime` commands can hold the system clock reliably
- The Wi-Fi AP shares the wired connection; both work simultaneously

## GPIO/UART Serial Connections

The Pi 5 has multiple UART serial interfaces available via GPIO pins. `install.sh` enables them by default by adding the following device tree overlays to `/boot/firmware/config.txt`:

- `dtoverlay=uart1`, `dtoverlay=uart2`, `dtoverlay=uart3` — Secondary UARTs
- `dtparam=uart0=on` — Primary UART (GPIO 14/15)
- `dtoverlay=dwc2,dr_mode=host` — USB host mode for dwc2

#### UART Pin Mapping (Pi 5, all 3.3V logic)

| Device | Function | GPIO | Pin | Notes |
|--------|----------|------|-----|-------|
| `/dev/ttyAMA0` | UART0 TX | GPIO 14 | Pin 8 | Primary UART |
| `/dev/ttyAMA0` | UART0 RX | GPIO 15 | Pin 10 | Primary UART |
| `/dev/ttyAMA1` | UART1 TX | GPIO 0 | Pin 27 | |
| `/dev/ttyAMA1` | UART1 RX | GPIO 1 | Pin 28 | |
| `/dev/ttyAMA2` | UART2 TX | GPIO 4 | Pin 7 | |
| `/dev/ttyAMA2` | UART2 RX | GPIO 5 | Pin 29 | |
| `/dev/ttyAMA3` | UART3 TX | GPIO 8 | Pin 24 | |
| `/dev/ttyAMA3` | UART3 RX | GPIO 9 | Pin 21 | |
| `/dev/ttyAMA10` | BT TX/RX | – | – | **Bluetooth UART** — enabled by default; see below to disable |

#### Using the Bluetooth UART (/dev/ttyAMA10)

By default, `/dev/ttyAMA10` is reserved for Bluetooth. If you need to use it as a regular serial interface, disable Bluetooth:

Add this line to `/boot/firmware/config.txt`:

```ini
dtparam=bluetooth=off
```

Then reboot. This frees up AMA10 for serial communication and disables the on-board Bluetooth radio.

After installation, a reboot is required for the overlays to take effect. To apply overlays without rebooting:

```bash
sudo dtoverlay -R           # Remove current overlays (may disrupt display)
sudo dtoverlay uart1 uart2 uart3  # Re-apply overlays
```

---

## Building from Source

### 1. Install system dependencies

```bash
sudo apt update
sudo apt install -y git golang libsdl2-dev libsdl2-gfx-dev libsdl2-image-dev libsdl2-ttf-dev libsdl2-mixer-dev libltc-dev libasound2-dev
```

This installs Go (1.22+ from Trixie repos), SDL2 development libraries for the CGo bindings, and ALSA/LTC libraries for the timecode decoder.

### 2. Clone the repository

```bash
cd ~
git clone https://github.com/jpkelly/clock8002.git
cd clock8002/v4
```

### 3. Build

```bash
make build
make alsa-ltc
```

Or build individually:

```bash
go build ./cmd/sdl-clock
gcc -O2 -o alsa-ltc alsa-ltc.c -lasound -lltc
```

The first Go build downloads module dependencies and compiles everything. This takes a few minutes on a Pi 5.

### 4. Set up fonts

The text clock faces require TTF fonts that live in the `ttf_fonts/` subdirectory. Create symlinks so the binary can find them in the working directory:

```bash
ln -sf ttf_fonts/*.ttf .
```

### 5. Add your user to video/render groups

SDL2 KMSDRM needs direct access to the display hardware:

```bash
sudo usermod -aG video,render $USER
```

Log out and back in (or reboot) for group changes to take effect.

### 6. Test run

```bash
SDL_VIDEODRIVER=kmsdrm ./sdl-clock --fullscreen
```

The clock should appear on the HDMI display. A startup overlay shows the version and IP address for about 30 seconds. Press Escape or Ctrl-C to exit.

On first run, a default config is created at `~/.config/clock-8001/clock.ini`. Logs are written to `~/.config/clock-8001/clock.log`.

### 7. Install as a system service

```bash
sudo cp clock8002.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable clock8002
sudo systemctl start clock8002
```

The service runs as your user, starts on boot, and auto-restarts on crash.

For runtime service commands, see [Service Operations](#service-operations).

## Service Operations

### Clock service

```bash
sudo systemctl status clock8002       # check status
sudo systemctl start clock8002        # start clock service
sudo systemctl restart clock8002      # restart after config changes
sudo systemctl stop clock8002         # stop the clock
sudo systemctl enable clock8002       # enable on boot
journalctl -u clock8002 -f            # live service logs
cat ~/.config/clock-8001/clock.log    # application log file
```

### LTC service

```bash
sudo systemctl enable alsa-ltc       # enable on boot
sudo systemctl start alsa-ltc        # start LTC decoder
sudo systemctl restart alsa-ltc      # restart LTC decoder
sudo systemctl stop alsa-ltc         # stop LTC decoder
sudo systemctl status alsa-ltc       # check status
journalctl -u alsa-ltc -f            # live LTC logs
```

The LTC decoder auto-detects USB audio capture devices and sends decoded timecode via OSC to the clock. Enable LTC on a source in the web UI or config file (`source1.ltc=true`).

## Updating

After pulling new changes, rebuild and restart:

```bash
cd ~/clock8002
git pull
cd v4
make build
make alsa-ltc
sudo systemctl restart clock8002
sudo systemctl restart alsa-ltc
```

If `clock8002.service` was modified, also run:

```bash
sudo cp clock8002.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl restart clock8002
```

## Cloning the Image

`rpi-clone` is installed by `install.sh` and can be used to clone a running system to another SD card.

Basic workflow:

```bash
lsblk
sudo rpi-clone -f -v sda
```

Notes:
- Replace `/dev/sda` with your actual target device (the whole disk, not a partition)
- The SD card target should be plugged into a USB port on the Pi
- `rpi-clone` overwrites the target disk. Confirm device name carefully before running.

## Configuration

### Web UI

Access the web configuration interface at `http://<pi-ip>:8080`. Default credentials: **admin** / **clockwork**.

All clock settings — face type, colors, sources, timers, OSC, GPIO — can be changed from the web UI without editing files.

### Config file

Edit `~/.config/clock-8001/clock.ini` directly. To generate a fresh default config:

```bash
cd ~/clock8002/v4
SDL_VIDEODRIVER=kmsdrm ./sdl-clock --dump-config > ~/.config/clock-8001/clock.ini
```

Key settings:

| Setting | Description | Default |
|---------|-------------|---------|
| `Face` | Clock face (see table above) | `round` |
| `FullScreen` | Start in full screen mode | `false` |
| `source1.tod` | Enable time-of-day on source 1 | `false` |
| `source1.timezone` | Timezone for source 1 | `Europe/Helsinki` |
| `Row1Color` | Text color for timer row 1 (hex) | `#FF8000` |
| `BackgroundColor` | Background color (hex) | `#000000` |
| `ListenAddr` | OSC listen address | `0.0.0.0:1245` |
| `HTTPPort` | Web config interface port | `:8080` |

## Platform

- **Target**: Raspberry Pi 5, arm64, Debian Trixie
- **Display**: HDMI via SDL2 KMSDRM (headless, no desktop)
- **Language**: Go with SDL2 CGo bindings
- **Web UI**: Built-in HTTP server on port 8080

## OSC Control

See the [original clock-8001 OSC documentation](https://gitlab.com/clock-8001/clock-8001/-/blob/master/v4/osc.md) for the full list of OSC commands.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Black screen / no output | Make sure `SDL_VIDEODRIVER=kmsdrm` is set and user is in `video` and `render` groups |
| "Couldn't open RobotoMono" | Run `ln -sf ttf_fonts/*.ttf .` in the v4 directory (source builds only) |
| Clock exits silently | Check `~/.config/clock-8001/clock.log` for errors |
| Web UI not accessible | Verify the service is running with `systemctl status clock8002` and check firewall |
| Config changes not applied | Restart the service: `sudo systemctl restart clock8002` |

## Creating a Release

On the Pi (where the binary is built natively):

```bash
cd ~/clock8002/v4
git tag vX.X.X
git push origin vX.X.X
make release
```

This produces `clock8002-vX.X.X-linux-arm64.tar.gz` containing sdl-clock, alsa-ltc, default config, fonts, voices, service files, and install script.

Upload to GitHub Releases:

```bash
# Install gh CLI if needed: sudo apt install gh
gh auth login
gh release create vX.X.X clock8002-vX.X.X-linux-arm64.tar.gz --title "vX.X.X" --notes "Release notes"
```

## License

This project is licensed under the GNU General Public License v2, the same license as the original clock-8001.
