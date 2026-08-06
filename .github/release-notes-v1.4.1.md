# Clock8002 v1.4.1

## What Changed

### Network
- **Fix HDMI info overlay reporting "Network Mode: Unknown".** The overlay code in `clock/engine.go` was reading the wrong boot-partition path for the network config. It now reads the correct path, so static, DHCP, and dual modes are reported correctly.
- **Implement `mode=dual`.** `piclock-network.sh` now keeps DHCP on the wired connection and adds the configured static address as a secondary IP, matching the behaviour already documented in `network.ini.default` and `boot-README.txt`.
- Correct the stale path comment in `network.ini.default` to point to the boot-partition location (`/boot/firmware/piclock/network.ini`).

### OLED
- **In `mode=dual`, the OLED IP line now alternates every 5 seconds** between the configured static address and the wired DHCP address, so both addresses are visible on the small display. Single/static and DHCP modes still show a single stable IP.

### Release packaging
- **Rename the Trixie installer tarball** from `clock8002-vX.X.X-default-linux-arm64.tar.gz` to `piClock-vX.X.X-linux-arm64.tar.gz`. The `default` variant segment is dropped now that only one variant exists.

## Install (Default)

```bash
wget https://github.com/jpkelly/clock8002/releases/download/v1.4.1/piClock-v1.4.1-linux-arm64.tar.gz
tar xzf piClock-v1.4.1-linux-arm64.tar.gz
cd piClock-v1.4.1-linux-arm64
sudo bash install.sh
```

## Notes

- Built for Raspberry Pi 5, arm64.
- This release is a **network/OLED feature release** — no changes to the LTC capture path or the `sdl3-clock` display binary.
