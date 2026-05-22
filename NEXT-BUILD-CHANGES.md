# Next Build Change List

Date started: 2026-05-22
Branch: feature/root-ram
Target build session baseline: br-root-ram-20260522-090437

## Implemented In Source (Pending Image Build/Test)

- [x] Issue 44 Locked decision: power button support wiring for golden runtime path.
  - Added startup script: `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S04power-button`
  - Added handler: `buildroot-external/board/clock8002-rpi5/golden-working-card/root/power-button.sh`
  - Wired into image assembly: `buildroot-external/board/clock8002-rpi5/post-build.sh`
  - Live validation on running unit: KEY_POWER detection confirmed in log mode.

- [x] Issue 44 Locked decision: machine-id persistence moved out of `/boot/piclock` user config area.
  - Updated store path in `S12machine-id` from `/boot/piclock/machine-id` to `/boot/.piclock-machine-id`
  - File: `buildroot-external/board/clock8002-rpi5/rootfs-overlay/etc/init.d/S12machine-id`

- [x] Issue 44 checklist: Network + WiFi + ini path standardization (remove legacy interface alias dependency).
  - Removed `ifup eth0:1` from runtime start scripts:
    - `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S99clock`
    - `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S99clock_bridge`
    - `buildroot-external/board/clock8002-rpi5/rootfs-overlay/etc/init.d/S99clock`
  - Removed legacy `/boot/interfaces` copy path from:
    - `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S03copy_clock_files`
    - `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S03copy_clock_bridge_files`
  - Removed post-build copy of `etc/network/interfaces` from golden payload in:
    - `buildroot-external/board/clock8002-rpi5/post-build.sh`
  - Intended authoritative network path is `network.ini` + ifupdown apply scripts.

- [x] Issue 44 checklist: backend-aligned apply path for current non-NetworkManager stack.
  - `S43piclock-network-prep` now generates `/etc/network/interfaces` from `/boot/piclock/network.ini` for modes:
    - `dhcp`
    - `static`
    - `dual` (DHCP on `eth0` + static alias `eth0:1`)
  - `S45piclock-network` no longer waits on `nmcli`; directly runs piclock network apply script.
  - `piclock-network.sh` updated to non-NM behavior for:
    - NTP best-effort toggle
    - hostname apply
    - Wi-Fi AP using `wpa_supplicant` (+ `dnsmasq` when available)
  - `v4/network.ini.default` updated to document `dual` mode and `ap_country`.

- [x] Issue 44 follow-up: enable UART1 so Perfect Cue path `/dev/ttyAMA1` is available by default.
  - Added `dtoverlay=uart1` to Buildroot firmware config source:
    - `buildroot-external/board/clock8002-rpi5/config.txt`
  - Added `dtoverlay=uart1` to golden working-card boot config:
    - `buildroot-external/board/clock8002-rpi5/golden-working-card/boot/config.txt`
  - Commit pushed: `59aa59a` (`buildroot: enable uart1 in boot configs`).

## Build/Test Queue (Incremental)

- [ ] Build image with only the currently queued changes.
- [ ] Flash and boot test unit.
- [ ] Validate power button end-to-end in default mode (actual shutdown path).
- [ ] Validate machine-id persists across at least two reboots from `/boot/.piclock-machine-id`.
- [ ] Validate network.ini modes on current stack:
  - [ ] `dhcp` mode: eth0 receives DHCP address only.
  - [ ] `static` mode: eth0 uses configured static address.
  - [ ] `dual` mode: eth0 gets DHCP and eth0:1 gets static alias.
- [ ] Validate Wi-Fi AP options from network.ini (`ap_enabled`, `ap_ssid`, `ap_password`, `ap_channel`, `ap_country`).
- [ ] Validate UART1/Perfect Cue path on flashed image:
  - [ ] `/dev/ttyAMA1` exists after boot.
  - [ ] `cue-serial=/dev/ttyAMA1` receives cue input as expected.
- [ ] Record outcomes on Issue #44 and in HANDOFF.
