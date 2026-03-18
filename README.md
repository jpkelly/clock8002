# Clock 8002

An HDMI clock display for Raspberry Pi 5, based on the [clock-8001](https://gitlab.com/clock-8001/clock-8001) project. Outputs a full-screen clock to HDMI via SDL2 with KMSDRM (no desktop environment required). Controllable via OSC.

## Acknowledgements

This project is a fork of **clock-8001** by Depili, developed in co-operation with Daniel Richert and with grants from FUUG — Finnish Unix User Group. The original project is available at https://gitlab.com/clock-8001/clock-8001 and is licensed under the GNU General Public License v2.

Please consider supporting the original clock-8001 development: https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR

## What changed from clock-8001

- Stripped to HDMI-only SDL clock output (removed HUB75 LED matrix, Arduino LED ring, Pimoroni Unicorn HD, Futaba VFD)
- Targets Raspberry Pi 5 running 64-bit Debian Trixie (arm64)
- Runs headless via KMSDRM — no X11 or Wayland needed
- GPIO pulse output support retained (periph.io)

## Requirements

- Raspberry Pi 5 running Raspberry Pi OS / Debian Trixie (64-bit)
- HDMI display
- Network connection (for OSC control and web configuration)

## Setup

### 1. Install dependencies

```bash
sudo apt update && sudo apt install -y golang libsdl2-dev libsdl2-gfx-dev libsdl2-image-dev libsdl2-ttf-dev libsdl2-mixer-dev
```

### 2. Clone and build

```bash
git clone https://github.com/jpkelly/clock8002.git
cd clock8002/v4
go build ./cmd/sdl-clock
ln -s ttf_fonts/*.ttf .
```

### 3. Test run

```bash
SDL_VIDEODRIVER=kmsdrm ./sdl-clock --fullscreen
```

The first run creates a default config at `~/.config/clock-8001/clock.ini`. Edit it to set your timezone, clock face, colors, etc. You can also dump a fresh default config with:

```bash
./sdl-clock --dump-config > ~/.config/clock-8001/clock.ini
```

### 4. Install as a system service

```bash
sudo usermod -aG video,render $USER
sudo cp clock8002.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable clock8002
sudo systemctl start clock8002
```

### Service management

```bash
sudo systemctl status clock8002       # check status
sudo systemctl restart clock8002      # restart
sudo systemctl stop clock8002         # stop
cat ~/.config/clock-8001/clock.log    # view logs
```

## Configuration

Edit `~/.config/clock-8001/clock.ini`. Key settings:

| Setting | Description | Default |
|---------|-------------|---------|
| `Face` | Clock face: `round`, `text`, `small`, `countdown`, etc. | `round` |
| `FullScreen` | Start in full screen mode | `false` |
| `source1.tod` | Enable time-of-day on source 1 | `false` |
| `source1.timezone` | Timezone for source 1 | `Europe/Helsinki` |
| `text-color` | Main text color (hex) | `#FF8000` |
| `BackgroundColor` | Background color (hex) | `#000000` |
| `ListenAddr` | OSC listen address | `0.0.0.0:1245` |
| `HTTPPort` | Web config interface port | `:8080` |

## Web Configuration

Access the web configuration interface at `http://<pi-ip>:8080`. Default credentials: `admin` / `clockwork`.

## Platform

- **Target**: Raspberry Pi 5, arm64, Debian Trixie
- **Display**: HDMI via SDL2 KMSDRM (headless, no desktop)
- **Language**: Go with SDL2 CGo bindings

## OSC Control

See the [original clock-8001 OSC documentation](https://gitlab.com/clock-8001/clock-8001/-/blob/master/v4/osc.md) for the full list of OSC commands.

## License

This project is licensed under the GNU General Public License v2, the same license as the original clock-8001.
