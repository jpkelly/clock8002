# Clock 8002

An HDMI clock display for Raspberry Pi 5, based on the [clock-8001](https://gitlab.com/clock-8001/clock-8001) project. Outputs a full-screen clock to HDMI via SDL2 with KMSDRM (no desktop environment required). Controllable via OSC and a built-in web UI.

## Acknowledgements

This project is a fork of **clock-8001** by Depili, developed in co-operation with Daniel Richert and with grants from FUUG — Finnish Unix User Group. The original project is available at https://gitlab.com/clock-8001/clock-8001 and is licensed under the GNU General Public License v2.

Please consider supporting the original clock-8001 development: https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR

## What changed from clock-8001

- Stripped to HDMI-only SDL clock output (removed HUB75 LED matrix, Arduino LED ring, Pimoroni Unicorn HD, Futaba VFD)
- Added "quad" face — text clock with 4 timers
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

Go to https://github.com/jpkelly/clock8002/releases and download the `clock8002-vX.X.X-linux-arm64.tar.gz` file to your Pi.

Or from the command line:

```bash
# Replace vX.X.X with the actual version
wget https://github.com/jpkelly/clock8002/releases/download/vX.X.X/clock8002-vX.X.X-linux-arm64.tar.gz
```

### 2. Extract and install

```bash
tar xzf clock8002-v*-linux-arm64.tar.gz
cd clock8002-v*-linux-arm64
./install.sh
```

The install script will:
- Install SDL2 runtime libraries (no dev packages or Go needed)
- Copy the binary and assets to `/opt/clock8002`
- Add your user to video/render groups
- Install and enable the systemd service

### 3. Start the clock

```bash
sudo systemctl start clock8002
```

Then open `http://<pi-ip>:8080` to configure (default: **admin** / **clockwork**).

---

## Building from Source

### 1. Install system dependencies

```bash
sudo apt update
sudo apt install -y git golang libsdl2-dev libsdl2-gfx-dev libsdl2-image-dev libsdl2-ttf-dev libsdl2-mixer-dev
```

This installs Go (1.22+ from Trixie repos) and all SDL2 development libraries needed for the CGo bindings.

### 2. Clone the repository

```bash
cd ~
git clone https://github.com/jpkelly/clock8002.git
cd clock8002/v4
```

### 3. Build

```bash
go build ./cmd/sdl-clock
```

The first build downloads Go module dependencies and compiles everything. This takes a few minutes on a Pi 5.

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

## Updating

After pulling new changes, rebuild and restart:

```bash
cd ~/clock8002
git pull
cd v4
go build ./cmd/sdl-clock
sudo systemctl restart clock8002
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
git tag v0.0.1
make release
```

This produces `clock8002-v0.0.1-linux-arm64.tar.gz` containing the binary, fonts, voices, service file, and install script.

Upload to GitHub Releases:

```bash
# Install gh CLI if needed: sudo apt install gh
gh auth login
gh release create v0.0.1 clock8002-v0.0.1-linux-arm64.tar.gz --title "v0.0.1" --notes "Initial release"
```

## License

This project is licensed under the GNU General Public License v2, the same license as the original clock-8001.
