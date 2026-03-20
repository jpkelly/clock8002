# Clock 8002

An HDMI clock display for Raspberry Pi 5, based on the [clock-8001](https://gitlab.com/clock-8001/clock-8001) project. Outputs a full-screen clock to HDMI via SDL2 with KMSDRM (no desktop environment required). Controllable via OSC and a built-in web UI.

This build is intended to be used with the piClock platform.

## Acknowledgements

This project is a fork of **clock-8001** by Depili, developed in co-operation with Daniel Richert and with grants from FUUG — Finnish Unix User Group. The original project is available at https://gitlab.com/clock-8001/clock-8001 and is licensed under the GNU General Public License v2.

Please consider supporting the original clock-8001 development: https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR

## What changed from clock-8001

- Stripped to HDMI-only SDL clock output (removed HUB75 LED matrix, Arduino LED ring, Pimoroni Unicorn HD, Futaba VFD)
- Added "quad" face — text clock with 4 timers
- Added "dual" face — text clock with 2 timers
- Added LTC timecode input via ALSA audio (alsa-ltc)
- Targets Raspberry Pi 5 running 64-bit Debian Trixie (arm64)
- Runs headless via KMSDRM — no X11 or Wayland needed
- GPIO pulse output support retained (periph.io)

## Clock Faces

| Face | Description |
|------|-------------|
| `round` | Single analog-style round clock |
| `dual-round` | Two round clocks side by side |
| `text` | Text clock with 3 timers |
| `quad` | Text clock with 4 timers |
| `dual` | Text clock with 2 timers |
| `single` | Text clock with 1 large timer |
| `max` | Maximal size single timer |
| `countdown` | Countdown to a fixed date/time |
| `192` | Small 192×192px round clock |
| `144` | Small 144×144px round clock |
| `288x144` | Small 288×144px text clock |

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

### 3. Start the clock

```bash
sudo systemctl start clock8002
```

Then open `http://<pi-ip>:8080` to configure (default: **admin** / **clockwork**).

### Pre-boot Network Configuration (optional)

You can configure the Pi's IP address and hostname without SSH by editing `/boot/piclock/network.ini`. This file is on the FAT32 boot partition, so you can mount the SD card on a Mac/PC to edit it.

Settings are applied automatically on every boot by the `piclock-network` service. After editing, just reboot for changes to take effect.

#### DHCP (default)

```ini
[network]
mode=dhcp

[host]
hostname=clock8002
```

#### Static IP

```ini
[network]
mode=static
address=10.0.0.100
netmask=24
gateway=10.0.0.1
dns=10.0.0.1

[host]
hostname=clock8002
```

#### NTP (Network Time Protocol)

By default, NTP is **disabled** so that OSC `settime` commands can set and hold the system clock. If your clocks are on a network with reliable NTP and you don't need OSC time control, you can enable it:

```ini
[network]
ntp=true
```

- `ntp=false` (default) — NTP disabled, OSC settime controls the clock
- `ntp=true` — NTP enabled, system clock syncs automatically from the network

**Notes:**
- Uncomment settings by removing the `#` at the start of the line
- The hostname is set without `.local` — mDNS adds that automatically
- A sample `network.ini` is included in the release tarball and installed to `/boot/piclock/` automatically
- To apply changes without rebooting: `sudo /opt/clock8002/piclock-network.sh`

#### Wi-Fi Access Point (optional)

The Pi can broadcast its own Wi-Fi network, allowing a phone or laptop to connect directly and access the web UI — useful for standalone deployments without existing Wi-Fi infrastructure.

Add a `[wifi]` section to `network.ini`:

```ini
[wifi]
ap_enabled=true
```

- **SSID** defaults to `<hostname>-ap` (e.g. `piclock3-ap`). Override with `ap_ssid=MyNetwork`.
- **Password** defaults to `clockwork`. Override with `ap_password=YourPassword` (minimum 8 characters).
- **Channel** defaults to `6`. Override with `ap_channel=11`.
- The AP uses NetworkManager's shared mode, which hands out DHCP addresses to clients (typically `10.42.0.x`).
- The wired Ethernet connection continues to work alongside the AP.
- Set `ap_enabled=false` to disable and remove the AP.

### 4. Enable LTC timecode (optional)

If you have a USB audio interface receiving LTC timecode:

```bash
sudo systemctl enable alsa-ltc
sudo systemctl start alsa-ltc
```

### 5. GPIO/UART Serial Connections (optional)

The Pi 5 has multiple UART serial interfaces available via GPIO pins. The installer enables them by default, adding the following device tree overlays to `/boot/firmware/config.txt`:

- `dtoverlay=uart1`, `dtoverlay=uart2`, `dtoverlay=uart3` — Secondary UARTs
- `dtparam=uart0=on` — Primary UART (GPIO 14/15)
- `dtoverlay=dwc2,dr_mode=host` — USB host mode for dwc2

#### UART Pin Mapping (Pi 5, all 3.3V logic)

| Device | Function | GPIO | Pin |
|--------|----------|------|-----|
| `/dev/ttyAMA0` | UART0 TX | GPIO 14 | Pin 8 |
| `/dev/ttyAMA0` | UART0 RX | GPIO 15 | Pin 10 |
| `/dev/ttyAMA1` | UART1 TX | GPIO 0 | Pin 27 |
| `/dev/ttyAMA1` | UART1 RX | GPIO 1 | Pin 28 |
| `/dev/ttyAMA2` | UART2 TX | GPIO 4 | Pin 7 |
| `/dev/ttyAMA2` | UART2 RX | GPIO 5 | Pin 29 |
| `/dev/ttyAMA3` | UART3 TX | GPIO 8 | Pin 24 |
| `/dev/ttyAMA3` | UART3 RX | GPIO 9 | Pin 21 |

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

### Service management

```bash
sudo systemctl status clock8002       # check status
sudo systemctl restart clock8002      # restart after config changes
sudo systemctl stop clock8002         # stop the clock
journalctl -u clock8002 -f            # live service logs
cat ~/.config/clock-8001/clock.log    # application log file
```

### LTC timecode service

```bash
sudo systemctl enable alsa-ltc       # enable on boot
sudo systemctl start alsa-ltc        # start LTC decoder
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
