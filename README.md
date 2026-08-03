# Clock 8002

An HDMI clock display for Raspberry Pi 5, controllable via OSC and a built-in web UI.

> **This branch (`master`) installs Clock 8002 on Raspberry Pi OS / Debian Trixie.** This is the primary, actively developed path.  
> For the ready-to-flash Buildroot appliance image, see the [`buildroot`](https://github.com/jpkelly/clock8002/tree/buildroot) branch. That platform is maintained but parked.

## Table of Contents

- [Acknowledgements](#acknowledgements)
- [What changed from clock-8001](#what-changed-from-clock-8001)
- [Requirements](#requirements)
- [Quick Start — Install on Raspberry Pi OS Trixie](#quick-start--install-on-raspberry-pi-os-trixie)
- [First Boot](#first-boot)
- [EEPROM Provisioning (Pi 5)](#eeprom-provisioning-pi-5)
- [Pre-boot Configuration](#pre-boot-configuration)
- [Config Files Overview](#config-files-overview)
- [Web UI](#web-ui)
- [Service Operations](#service-operations)
- [GPIO/UART Serial Connections](#gpiouart-serial-connections)
- [OSC Control](#osc-control)
- [Project Status](#project-status)
- [License](#license)

## Acknowledgements

This project is a fork of **clock-8001** by Depili, developed in co-operation with Daniel Richert and with grants from FUUG — Finnish Unix User Group. The original project is available at https://gitlab.com/clock-8001/clock-8001 and is licensed under the GNU General Public License v2.

Please consider supporting the original clock-8001 development: https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR

## What changed from clock-8001

- Runtime: SDL3, runs headless via KMSDRM (no X11/Wayland)
- Deployment: installer tarball for Raspberry Pi OS / Debian Trixie on arm64
- Features: added `text4`, `text2`, and 3-line text clock faces
- Features: added configurable PerfectCue overlay placement/size in the web UI
- Features: added 2nd HDMI output support — PerfectCue full-screen icons or main clock mirror
- Features: font selection via dropdown in web UI
- Features: OLED display daemon (SSD1306 I2C) with version overlay
- Features: refactored `alsa-ltc` binary for 64-bit arm64
- Features: retained GPIO pulse output support (`periph.io`)
- Platform: targets Raspberry Pi 5 running Raspberry Pi OS Lite (64-bit) / Debian Trixie
- Removed hardware: HUB75 LED matrix, Arduino LED ring, Pimoroni Unicorn HD, Futaba VFD

## Requirements

Built for the **piClock platform** on Raspberry Pi 5, arm64.

- **Hardware**: Raspberry Pi 5 (field units currently include 2GB and 8GB variants)
- **Display**: HDMI via SDL3 KMSDRM (headless, no desktop required)
- **OS**: Raspberry Pi OS Lite (64-bit) / Debian Trixie (arm64), with internet access for the installer
- **Network**: Required for OSC control and web configuration

## Quick Start — Install on Raspberry Pi OS Trixie

### 1. Prepare the SD card

Flash **Raspberry Pi OS Lite (64-bit)** (Debian Trixie) to an SD card using [Raspberry Pi Imager](https://www.raspberrypi.com/software/).

During imager customisation:

- Set hostname (e.g. `piclock`)
- Set username `pi` and password
- Enable SSH
- Configure Wi-Fi or wired networking so the Pi has internet access

Boot the Pi and SSH in:

```bash
ssh pi@piclock.local
```

### 2. Download and install Clock 8002

Download the latest Trixie release tarball from the [Releases](https://github.com/jpkelly/clock8002/releases) page. Look for assets named `clock8002-vX.X.Y-default-linux-arm64.tar.gz`.

```bash
wget https://github.com/jpkelly/clock8002/releases/download/v1.4.0/clock8002-v1.4.0-default-linux-arm64.tar.gz
tar xzf clock8002-v1.4.0-default-linux-arm64.tar.gz
cd clock8002-v1.4.0-default-linux-arm64
sudo bash install.sh
```

The installer will:

- Install SDL3 and other runtime dependencies
- Copy `sdl-clock`, `alsa-ltc`, fonts, voices, and assets to `/opt/clock8002`
- Install default configs to `/boot/firmware/piclock/`
- Install and enable systemd services
- Add the `pi` user to the `video` and `render` groups

Reboot when the installer finishes:

```bash
sudo reboot
```

### 3. Access the web UI

After reboot, open a browser to:

```
http://piClock.local
```

Default credentials: **admin** / **clockwork**

---

## First Boot

On first boot the clock starts automatically. A startup overlay displays the version and IP address for ~30 seconds.

Network settings are read from `/boot/firmware/piclock/network.ini` at boot. To change settings after booting, edit that file via SSH and reboot, or use the web UI.

## EEPROM Provisioning (Pi 5)

Factory Pi 5 units ship with `BOOT_ORDER=0xf461` (SD → USB → network → restart loop). For piClock, change this to `BOOT_ORDER=0xf1` (SD-only). This is a one-time operation per unit.

SSH into the running unit (as the `pi` user) and run:

```bash
printf '[all]\nBOOT_UART=1\nBOOT_ORDER=0xf1\n' > /tmp/eeprom.cfg
BLOB=$(ls /usr/lib/firmware/raspberrypi/bootloader-2712/default/pieeprom-*.bin | sort | tail -1)
rpi-eeprom-config --config /tmp/eeprom.cfg --out /tmp/custom.bin "$BLOB"
sudo rpi-eeprom-update -d -f /tmp/custom.bin
sudo reboot
```

After reboot, verify with `rpi-eeprom-config` — expect `BOOT_ORDER=0xf1`.

## Pre-boot Configuration

Config files live on the FAT32 boot partition at `/boot/firmware/piclock/` (visible as `piClock` when the SD card is mounted on Mac/PC). Edit them before first boot, or via SSH at the same path after booting.

### network.ini

Controls hostname, network mode, NTP, and Wi-Fi AP mode. Changes take effect on reboot.

```ini
[network]
# mode: dhcp, static, or dual
mode=static

# NTP time synchronization (true or false)
ntp=false

# Static IP settings (only used when mode=static or dual)
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

**Network modes:**

| Mode | Behaviour |
|------|-----------|
| `static` | Fixed IP/netmask on eth0. Set `address=`, `netmask=`, and optionally `gateway=` and `dns=`. |
| `dhcp` | DHCP on eth0. IP, gateway, and DNS are provided by the router. |
| `dual` | DHCP on eth0 for the default route, plus a static alias on `eth0:1`. Useful when the unit needs a predictable IP for OSC while still getting internet via DHCP. |

**`gateway=`** — sets the default route. Required if you need internet access (e.g. for NTP or DNS) in `static` mode. Comment it out if piClock is on an isolated network.

**`dns=`** — writes a nameserver to `/etc/resolv.conf` at boot. Only needed in `static` mode; DHCP provides DNS automatically. Example: `dns=1.1.1.1`.

> **NTP on piClock:** Controlled by `ntp=true|false` in `network.ini`, applied via `timedatectl set-ntp` (systemd-timesyncd) at boot. NTP is only needed for initial time accuracy at boot.

piClock can run a Wi-Fi AP alongside its wired or wireless client connection. When active, it creates a hotspot you can join directly from a laptop or phone — useful to wirelessly access the piClock without relying on an existing network.

Set `ap_enabled=true` in `network.ini` to enable it. Set to `false` (the default) to disable.

Once connected to the AP, open `http://piClock.local` (or the unit's AP-side IP) in a browser.

> **OLED indicator:** A Wi-Fi icon appears in the top-right corner of the OLED display when the Wi-Fi AP is active, and is absent when it is off.

### authorized_keys

Place SSH public key(s) in `/boot/firmware/piclock/authorized_keys` for passwordless login as the `pi` user. Applied at every boot — no reflash required.

When the SD card is mounted on your computer, the FAT boot partition will appear as a drive named `piClock`. Add your public key(s) to the file `piclock/authorized_keys` on that partition (create the file if it does not exist).

### oled.ini

Controls the OLED status display. Changes take effect on service restart.

```ini
[oled]
enabled=true
i2c_port=1
i2c_address=0x3c
rotation=2
```

- **enabled** — set to `false` to disable the OLED display entirely.
- **i2c_port**, **i2c_address** — OLED hardware bus configuration. Match your connected display module.
- **rotation** — display rotation: `0`, `1`, `2`, or `3` (each 90°).

## Config Files Overview

| File | Location | Purpose |
|---|---|---|
| `clock.ini` | `/boot/firmware/piclock/clock.ini` | Main clock config — face, colors, sources, timers, OSC, GPIO, web UI port |
| `network.ini` | `/boot/firmware/piclock/network.ini` | Network config — DHCP/static IP, hostname, Wi-Fi AP mode |
| `oled.ini` | `/boot/firmware/piclock/oled.ini` | OLED display — enable, I²C bus, rotation |
| `authorized_keys` | `/boot/firmware/piclock/authorized_keys` | SSH public keys for passwordless login as the `pi` user |

Changes to `clock.ini` take effect on service restart or via the web UI. Changes to `network.ini` take effect on reboot.

## Web UI

Access the configuration interface at `http://piClock.local` (or by IP). Default credentials: **admin** / **clockwork**.

All clock settings — face type, colors, sources, timers, OSC, GPIO — can be changed from the web UI without editing files. Settings are saved to `clock.ini` on the boot partition.

> **Note:** On a new install, the first configuration save will reboot the clock.

## Service Operations

```sh
# Clock
sudo systemctl start clock8002
sudo systemctl stop clock8002
sudo systemctl restart clock8002
sudo systemctl status clock8002

# LTC decoder
sudo systemctl start alsa-ltc
sudo systemctl stop alsa-ltc
sudo systemctl restart alsa-ltc
sudo systemctl status alsa-ltc

# Network configuration
sudo systemctl start piclock-network
sudo systemctl status piclock-network
```

Log file: `~/.config/clock-8001/clock.log` (also viewable in the web UI). `journalctl -u clock8002` only shows startup output before logging redirects to this file.

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

The Trixie `piclock-network.service` handles this automatically in static-IP mode:

1. **Subnet broadcast** — NetworkManager leaves eth0 with `brd 0.0.0.0`. The network script re-adds the address with `broadcast +` so the kernel derives the correct subnet broadcast (e.g. `192.168.8.255` for a /24).
2. **Limited broadcast route** — regardless of whether `gateway=` is configured in `network.ini`, the script adds `ip route replace 255.255.255.255/32 dev eth0`. This makes LTC broadcast reliable with the gateway commented out (the default).

Verify on a running unit:

```sh
ip addr show eth0   # brd should show subnet broadcast, e.g. 192.168.8.255
ip route show       # should include: 255.255.255.255 dev eth0 scope link
```

## GPIO/UART Serial Connections

See the [piClock wiring diagram (PDF)](https://github.com/jpkelly/clock8002/raw/master/docs/piClockWiring.pdf) for a visual overview of UART, RS-485, LTC, and HDMI connections.

The installer enables the required UART overlays in `/boot/firmware/config.txt`. Reboot after installation for the overlays to take effect.

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

Last verified: **2026-08-03**

- This branch (`master`) provides the Raspberry Pi OS / Debian Trixie installer path, and is the primary platform.
- Current release is `v1.4.0` (see Quick Start above).
- The ready-to-flash Buildroot image is on the [`buildroot`](https://github.com/jpkelly/clock8002/tree/buildroot) branch, maintained but parked.
- Build policy and reproducibility rules live in [`docs/build-policy.md`](docs/build-policy.md).
- Current branch operational state and validation targets live in [`HANDOFF.md`](HANDOFF.md).

## License

This project is licensed under the GNU General Public License v2, the same license as the original clock-8001.
