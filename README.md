# Clock 8002

An HDMI clock display for Raspberry Pi 5, running as a minimal appliance on a custom **Buildroot** image. Controllable via OSC and a built-in web UI.

## Table of Contents

- [Acknowledgements](#acknowledgements)
- [What changed from clock-8001](#what-changed-from-clock-8001)
- [Project Status](#project-status)
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

- **Hardware**: Raspberry Pi 5 (field units currently include 2GB and 8GB variants)
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

**Option A — Raspberry Pi Imager (recommended):** Use **[Raspberry Pi Imager](https://www.raspberrypi.com/software/)** — choose "Use custom image" and select the `.img` file. This works on macOS, Windows, and Linux.

**Option B — `dd` (macOS):**

```bash
# Identify your SD card disk number (look for the correct size)
diskutil list external physical

# Flash (replace diskN with your disk number)
diskutil unmountDisk /dev/diskN
sudo dd if=/path/to/piClock-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress
diskutil eject /dev/diskN
```

**Option C — `dd` (Linux):**

```bash
# Identify your SD card device (look for the correct size)
lsblk

# Flash (replace sdX or mmcblkX with your device)
sudo umount /dev/sdX*
sudo dd if=/path/to/piClock-<COMMIT>-sdcard.img of=/dev/sdX bs=4M status=progress
sync
```

> **Warning:** Flashing overwrites all data on the SD card. Verify you have selected the correct device before proceeding.

### 4. Boot and access the web UI

Insert the SD card and power on. After ~30 seconds, open a browser to:

```
http://piClock.local
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

Controls hostname, network mode, NTP, and Wi-Fi AP mode. Changes take effect on reboot.

```ini
[network]
# mode: dhcp, static, or dual
mode=static

# NTP time synchronization (true or false)
ntp=false

# Static IP settings (only used when mode=static)
address=192.168.8.245
netmask=24
#gateway=192.168.8.1
#dns=1.1.1.1

[host]
hostname=piClock

[wifi]
ap_enabled=false
ap_ssid=piClock-ap
ap_password=clockwork
ap_channel=6
ap_country=US
```

#### Wi-Fi Access Point

piClock can run a Wi-Fi AP alongside its wired or wireless client connection. When active, it creates a hotspot you can join directly from a laptop or phone — useful to wirelessly access the piClock without relying on an existing network.

Set `ap_enabled=true` in `network.ini` to enable it. Set to `false` (the default) to disable.

Once connected to the AP, open `http://piClock.local` (or the unit's AP-side IP) in a browser.

> **OLED indicator:** The small dot in the top-right corner of the OLED display is lit when the Wi-Fi AP is active, and dark when it is off.

### authorized_keys

Place SSH public key(s) in `/boot/piclock/authorized_keys` for passwordless root login. Applied at every boot — no reflash required.

When the SD card is mounted on your computer, the FAT boot partition will appear as a drive named `piClock`. Add your public key(s) to the file `piclock/authorized_keys` on that partition (create the file if it does not exist).

## Config Files Overview

| File | Location | Purpose |
|---|---|---|
| `clock.ini` | `/boot/piclock/clock.ini` | Main clock config — face, colors, sources, timers, OSC, GPIO, web UI port |
| `network.ini` | `/boot/piclock/network.ini` | Network config — DHCP/static IP, hostname, Wi-Fi AP mode |
| `authorized_keys` | `/boot/piclock/authorized_keys` | SSH public keys for passwordless root login |

Changes to `clock.ini` take effect on service restart or via the web UI. Changes to `network.ini` take effect on reboot.

## Web UI

Access the configuration interface at `http://piClock.local` (or by IP). Default credentials: **admin** / **clockwork**.

All clock settings — face type, colors, sources, timers, OSC, GPIO — can be changed from the web UI without editing files. Settings are saved to `clock.ini` on the boot partition.

> **Note:** On a new install, the first configuration save will reboot the clock.

## Service Operations

```sh
# Clock
/etc/init.d/S99clock start
/etc/init.d/S99clock stop
/etc/init.d/S99clock restart
ps | grep sdl-clock          # status

# LTC decoder
/etc/init.d/S99alsa-ltc start
/etc/init.d/S99alsa-ltc stop
/etc/init.d/S99alsa-ltc restart
ps | grep alsa-ltc           # status
```

Log file: `/var/log/messages` (syslog)

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

#### Broadcast addressing and gateway

`alsa-ltc` sends timecode to `255.255.255.255` (the IPv4 *limited broadcast* address), which all hosts on the local network receive. The Linux kernel requires a route for this address — without one it returns `ENETUNREACH` and alsa-ltc silently fails to send.

piClock's `setup.sh` handles this automatically in static-IP mode:

1. **Subnet broadcast** — BusyBox `ifup` leaves eth0 with `brd 0.0.0.0`. `setup.sh` re-adds the address with `broadcast +` so the kernel derives the correct subnet broadcast (e.g. `192.168.8.255` for a /24).
2. **Limited broadcast route** — regardless of whether `gateway=` is configured in `network.ini`, `setup.sh` adds `ip route replace 255.255.255.255/32 dev eth0`. This makes LTC broadcast reliable with the gateway commented out (the default).

Verify on a running unit:

```sh
ip addr show eth0   # brd should show subnet broadcast, e.g. 192.168.8.255
ip route show       # should include: 255.255.255.255 dev eth0 scope link
```

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

See the local OSC reference in [`v4/osc.md`](v4/osc.md) for the full command set used by this repository.

Upstream reference is also available at [clock-8001/v4/osc.md](https://gitlab.com/clock-8001/clock-8001/-/blob/master/v4/osc.md).

## Project Status

Last verified: **2026-05-25**

- This README is the operator/user quick-start reference.
- Build policy and reproducibility rules live in [`docs/build-policy.md`](docs/build-policy.md).
- Current branch operational state and validation targets live in [`HANDOFF.md`](HANDOFF.md).
- Build host workflow details live in [`buildroot-external/README.buildroot.md`](buildroot-external/README.buildroot.md).

## License

This project is licensed under the GNU General Public License v2, the same license as the original clock-8001.
