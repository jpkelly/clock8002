# Clock 8002

An HDMI clock display for Raspberry Pi 5, running as a minimal appliance on a custom **Buildroot** image. No desktop environment required. Controllable via OSC and a built-in web UI.

> **Legacy Trixie/SDL2 path:** The Debian Trixie (Raspberry Pi OS) deployment path is preserved on the [`trixie` branch](../../tree/trixie). New deployments should use this branch.

## Table of Contents

- [Acknowledgements](#acknowledgements)
- [What changed from clock-8001](#what-changed-from-clock-8001)
- [Requirements](#requirements)
- [Quick Start — Flash the Image](#quick-start--flash-the-image)
- [First Boot](#first-boot)
- [EEPROM Provisioning (Pi 5)](#eeprom-provisioning-pi-5)
- [Pre-boot Configuration](#pre-boot-configuration)
- [Config Files Overview](#config-files-overview)
- [Web UI](#web-ui)
- [Service Operations](#service-operations)
- [GPIO/UART Serial Connections](#gpiouart-serial-connections)
- [OSC Control](#osc-control)
- [Building from Source](#building-from-source)
- [Legacy Trixie/SDL2 Path](#legacy-trixiesdl2-path)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Acknowledgements

This project is a fork of **clock-8001** by Depili, developed in co-operation with Daniel Richert and with grants from FUUG — Finnish Unix User Group. The original project is available at https://gitlab.com/clock-8001/clock-8001 and is licensed under the GNU General Public License v2.

Please consider supporting the original clock-8001 development: https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR

## What changed from clock-8001

- Runtime: SDL3, runs headless via KMSDRM (no X11/Wayland)
- Deployment: minimal Buildroot appliance image — flash to SD card, boot, done
- Features: added `text4`, `text2`, and 3-line text clock faces
- Features: added configurable PerfectCue overlay placement/size in the web UI
- Features: added 2nd HDMI output support — PerfectCue full-screen icons or main clock mirror
- Features: font selection via dropdown in web UI
- Features: OLED display daemon (SSD1306 I2C) with version overlay
- Features: refactored `alsa-ltc` binary for 64-bit arm64
- Features: retained GPIO pulse output support (`periph.io`)
- Platform: targets Raspberry Pi 5 with custom Buildroot image (musl libc, BusyBox init, SDL3)
- Removed hardware: HUB75 LED matrix, Arduino LED ring, Pimoroni Unicorn HD, Futaba VFD

## Requirements

Built for the **piClock platform** on Raspberry Pi 5, arm64.

- **Hardware**: Raspberry Pi 5 (1GB or 2GB)
- **Display**: HDMI via SDL3 KMSDRM (headless, no desktop required)
- **OS**: Custom Buildroot image — pre-built images on the [Releases](https://github.com/jpkelly/clock8002/releases) page
- **Network**: Required for OSC control and web configuration

## Quick Start — Flash the Image

### 1. Download the latest release image

Get the pre-built SD card image from the [Releases](https://github.com/jpkelly/clock8002/releases) page. The image is named `piClock-<commit>-sdcard.img`.

### 2. (Optional) Pre-configure before first boot

Mount the SD card's FAT32 boot partition (appears as `piClock` on Mac/PC) and edit:

- `piclock/network.ini` — set hostname, static IP, Wi-Fi AP settings (applied on first boot)
- `piclock/authorized_keys` — add SSH public key(s) for passwordless root login

### 3. Flash to SD card

Identify the SD card device:

```bash
diskutil list external physical
```

Flash (replace `diskN` with your actual disk number):

```bash
diskutil unmountDisk /dev/diskN
sudo dd if=/Users/yourname/Desktop/piClock-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress
diskutil eject /dev/diskN
```

> **Warning:** `dd` overwrites without prompting. Verify the disk number carefully before running.

Alternatively, use **Raspberry Pi Imager** → "Use custom image" to flash the `.img` file.

### 4. Boot and access the web UI

Insert the SD card and power on. After ~20 seconds, open a browser to:

```
http://piClock.local:8080
```

Default credentials: **admin** / **clockwork**

Default SSH access:

```bash
ssh root@piClock.local   # password: clockworkadmin
```

---

## First Boot

On first boot the clock starts automatically. A startup overlay displays the version and IP address for ~30 seconds.

If you pre-configured `network.ini` before flashing, network settings are applied on first boot. To change settings after booting, edit `/boot/piclock/network.ini` via SSH and reboot.

## EEPROM Provisioning (Pi 5)

Factory Pi 5 units ship with `BOOT_ORDER=0xf461` (SD → USB → network → restart loop). For piClock, change this to `BOOT_ORDER=0xf1` (SD-only). This is a one-time operation per unit.

SSH into the running unit and run:

```bash
printf '[all]\nBOOT_UART=1\nBOOT_ORDER=0xf1\n' > /tmp/eeprom.cfg
BLOB=$(ls /usr/lib/firmware/raspberrypi/bootloader-2712/default/pieeprom-*.bin | sort | tail -1)
rpi-eeprom-config --config /tmp/eeprom.cfg --out /tmp/custom.bin "$BLOB"
rpi-eeprom-update -d -f /tmp/custom.bin
reboot
```

After reboot, verify with `rpi-eeprom-config` — expect `BOOT_ORDER=0xf1`.

## Pre-boot Configuration

Config files live on the FAT32 boot partition at `/boot/piclock/` (visible as `piClock` when the SD card is mounted on Mac/PC). Edit them before first boot, or via SSH at the same path after booting.

### network.ini

Controls hostname, DHCP/static IP, NTP, and Wi-Fi AP mode. Changes take effect on reboot.

```ini
[network]
mode=dhcp                          # dhcp or static
ntp=false                          # disable if using OSC settime
# address=10.0.0.100               # static IP only
# netmask=24
# gateway=10.0.0.1
# dns=8.8.8.8
hostname=piClock
ap-ssid=piClock-ap
ap-passphrase=clockwork1
ap-channel=6
```

#### Wi-Fi Access Point

piClock can run a Wi-Fi AP alongside its wired or wireless client connection. When active, it creates a hotspot you can join directly from a laptop or phone to reach the web UI — useful on sets where there is no existing Wi-Fi network.

Set `ap-ssid` and `ap-passphrase` in `network.ini` to enable it. Leave `ap-ssid` blank (or remove the line) to disable the AP entirely.

Once connected to the AP, open `http://piClock.local:8080` (or the unit's AP-side IP) in a browser.

> **OLED indicator:** The small dot in the top-right corner of the OLED display is lit when the Wi-Fi AP is active, and dark when it is off.

### authorized_keys

Place SSH public key(s) in `/boot/piclock/authorized_keys` for passwordless root login. Applied at every boot — no reflash required.

```bash
# From Mac, with SD card mounted as piClock:
echo "$(cat ~/.ssh/id_ed25519.pub)" >> /Volumes/piClock/piclock/authorized_keys
```

## Config Files Overview

| File | Location | Purpose |
|---|---|---|
| `clock.ini` | `/boot/piclock/clock.ini` | Main clock config — face, colors, sources, timers, OSC, GPIO, web UI port |
| `network.ini` | `/boot/piclock/network.ini` | Network config — DHCP/static IP, hostname, Wi-Fi AP mode |
| `oled.ini` | `/boot/piclock/oled.ini` | OLED hardware config — enable/disable, I2C port, I2C address, rotation |
| `authorized_keys` | `/boot/piclock/authorized_keys` | SSH public keys for passwordless root login |

`/opt/clock8002/clock.ini` and `/opt/clock8002/oled/oled.ini` are symlinks into `/boot/piclock/`. Changes to `clock.ini` take effect on service restart or via the web UI. Changes to `network.ini` and `oled.ini` take effect on reboot.

## Web UI

Access the configuration interface at `http://<pi-ip>:8080`. Default credentials: **admin** / **clockwork**.

All clock settings — face type, colors, sources, timers, OSC, GPIO — can be changed from the web UI without editing files. Settings are saved to `clock.ini` on the boot partition.

## Service Operations

```bash
# Clock
systemctl start clock8002
systemctl stop clock8002
systemctl restart clock8002
systemctl status clock8002
journalctl -u clock8002 -f

# LTC decoder
systemctl start alsa-ltc
systemctl stop alsa-ltc
systemctl restart alsa-ltc
systemctl status alsa-ltc
journalctl -u alsa-ltc -f

# OLED daemon
systemctl start oled_daemon
systemctl stop oled_daemon
systemctl status oled_daemon
```

Log file: `/root/.config/clock-8001/clock.log`

### LTC (alsa-ltc)

The LTC decoder auto-detects USB audio capture devices and sends decoded timecode via OSC to the clock. Enable LTC on a source in the web UI or config file (`source1.ltc=true`).

```
alsa-ltc [-v] <alsa-device> <OSC-destination-ip> <OSC-port> [sample-rate] [fps]
```

| Argument | Description |
|---|---|
| `-v` | Verbose — prints `.` per decoded frame, ALSA params, signal level |
| `<alsa-device>` | ALSA capture device, or `-` for auto-detect |
| `<OSC-destination-ip>` | IP to send OSC timecode to (`255.255.255.255` for subnet broadcast) |
| `<OSC-port>` | UDP port (default: 1245) |
| `[sample-rate]` | Audio sample rate in Hz (default: 44100) |
| `[fps]` | LTC frame rate: 24, 25, or 30 (default: 25) |

## GPIO/UART Serial Connections

See the [piClock wiring diagram (PDF)](docs/piClockWiring.pdf) for a visual overview of UART, RS-485, LTC, and HDMI connections.

UART overlays are pre-configured in the Buildroot image `config.txt`. No manual overlay installation required.

#### UART Pin Mapping (Pi 5, 3.3V logic, current piclock wiring)

| Device | Function | GPIO | Pin | Notes |
|--------|----------|------|-----|-------|
| `/dev/ttyAMA0` | UART0 TX | GPIO 14 | Pin 8 | Primary UART |
| `/dev/ttyAMA0` | UART0 RX | GPIO 15 | Pin 10 | Primary UART |
| `/dev/ttyAMA1` | UART1 TX | GPIO 0 | Pin 27 | PerfectCue TX (TTL to RS485 converter) |
| `/dev/ttyAMA1` | UART1 RX | GPIO 1 | Pin 28 | PerfectCue RX (TTL to RS485 converter) |
| `/dev/ttyAMA2` | UART2 TX | GPIO 4 | Pin 7 | |
| `/dev/ttyAMA2` | UART2 RX | GPIO 5 | Pin 29 | |
| `/dev/ttyAMA3` | UART3 TX | GPIO 8 | Pin 24 | Limitimer TX (TTL to RS485 converter) |
| `/dev/ttyAMA3` | UART3 RX | GPIO 9 | Pin 21 | Limitimer RX (TTL to RS485 converter) |

## OSC Control

See the [original clock-8001 OSC documentation](https://gitlab.com/clock-8001/clock-8001/-/blob/master/v4/osc.md) for the full list of OSC commands.

## Building from Source

The Buildroot image is built on a dedicated build host (Pi CM5). See [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md) for full build, deploy, and release procedures.

For a quick binary cross-compile (code changes only, no full image rebuild):

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOFLAGS=-mod=vendor go build -o /tmp/sdl3-clock-linux-arm64 ./cmd/sdl3-clock
```

Deploy directly to a running unit:

```bash
systemctl stop clock8002
scp /tmp/sdl3-clock-linux-arm64 root@piClock.local:/opt/clock8002/sdl-clock
ssh root@piClock.local 'systemctl start clock8002'
```

> No `install.sh` exists on Buildroot — deploy binaries directly.

## Legacy Trixie/SDL2 Path

The original SDL2/Debian Trixie deployment path (using `install.sh` on Raspberry Pi OS) is preserved on the [`trixie` branch](../../tree/trixie). It is no longer actively developed but remains available for reference.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Black screen / no output | Check `systemctl status clock8002` — ensure DRM/KMS is not held by another process |
| Text invisible on display | Verify Mesa 25.0.7 patches were applied to the Buildroot build host (see [issue #29](https://github.com/jpkelly/clock8002/issues/29)) |
| Clock exits silently | Check `/root/.config/clock-8001/clock.log` and `journalctl -u clock8002` |
| Web UI not accessible | Verify `systemctl status clock8002` is active and check network connectivity |
| Config changes not applied | Restart the service: `systemctl restart clock8002` |
| SSH: too many auth failures | Use `-o IdentitiesOnly=yes -i ~/.ssh/id_ed25519` to specify the key |
| alsa-ltc not decoding LTC | Check `systemctl status alsa-ltc`; verify USB audio device is connected |

## License

This project is licensed under the GNU General Public License v2, the same license as the original clock-8001.
