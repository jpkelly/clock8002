# Clock8002 Handoff

Last updated: 2026-05-06 (~21:30 UTC)

## Contents

- [Current State](#current-state)
  - [Active Investigation: VL805 xHCI crash on piclockTG](#active-investigation-vl805-xhci-crash-on-piclocktg-2026-04-14)
  - [VL805 xHCI crash — EEPROM firmware implicated (Issue #39)](#vl805-xhci-crash--eeprom-firmware-implicated-issue-39-re-opened-2026-04-14)
  - [Resolved: Static IP timecode not displaying](#resolved-static-ip-timecode-not-displaying)
  - [Fixed in v1.3.0](#fixed-in-v130-2026-04-1011213--alsa-ltc-enhancements--hardware-debugging--subnet-broadcast)
- [Hardware & Firmware Notes](#hardware--firmware-notes)
  - [Pi 5 EEPROM Firmware — VL805 Stability](#pi-5-eeprom-firmware--vl805-stability-2026-04-14)
- [Buildroot Status](#buildroot-status-post-merge)
- [Issue #23 Status (Dual HDMI Output)](#issue-23-status-dual-hdmi-output)
- [Hot-Plug Policy](#hot-plug-policy-current-expected-behavior)
- [Recent Release Notes](#recent-release-notes)
- [OLED Splash Version Overlay](#oled-splash-version-overlay)
- [Release Process](#release-process-current)
- [Repository Instructions Already Added](#repository-instructions-already-added)
- [Hard Rules (Do Not Skip)](#hard-rules-do-not-skip)
- [Useful Commands](#useful-commands)
- [Release Notes Template Workflow](#release-notes-template-workflow)
- [Dev-Deploy Workflow](#dev-deploy-workflow-feature-branch-testing)
- [Buildroot Known Issues](#buildroot-known-issues)
- [SDL3 Migration Status](#sdl3-migration-status)
- [alsa-ltc Enhancements](#alsa-ltc-enhancements-2026-04-11)
- [Next Suggested Release](#next-suggested-release)

---

## Current State

- Repository: jpkelly/clock8002
- Active release line: v1.x
- Latest tagged release: **v1.3.1** — Trixie tarballs on GitHub (default + gerry)
- master HEAD: **`1f8d018`**
- **SDL3 migration**: `feature/sdl3-migration` — Buildroot SDL3 image **BUILT** on cm5 (`a6756f3`). Image `piclockBR-a6756f3-sdcard.img` on Mac Desktop (768 MB). Awaiting flash to piclockBR for hardware validation (Phase 4). See [SDL3 Migration Status](#sdl3-migration-status) and HANDOFF-SDL3.md.
- **1GB Pi 5 board A** (piclockBR.local): running Buildroot SDL2 image `piclockBR-c7e3a60-sdcard.img`. Will be flashed with SDL3 image when ready.
- **1GB Pi 5 board B** (piclockR.local): Trixie, v1.3.1 default. EEPROM 2025-08-28 (factory, not downgraded — canary). Monitor running.
- **2GB Pi 5 board #1** (piclockTG.local): fresh Trixie 6.12.47, v1.3.1 gerry. EEPROM **upgraded to 2025-08-27** (2026-04-15). Soak in progress (started ~06:38 UTC 2026-04-15). Monitor running.
- **2GB Pi 5 board #2** (piclockTD.local): fresh Trixie 6.12.47, v1.3.1 default. EEPROM 2025-05-08 (control, unchanged). ~17h clean as of 2026-04-15 06:43 UTC. Monitor running.
- `buildroot-prototype` branch: fully merged into master
- **VPN note**: `.local` mDNS does not resolve over VPN. Use IPs: TG=10.0.0.128, TD=10.0.0.131, piclockR=10.0.0.121, cm5=10.0.0.101

### Active Investigation: VL805 xHCI Instability — EEPROM Firmware (Issue #39)

**Current status (2026-04-15):** Multi-unit soak in progress to identify the stable EEPROM version. Three units running with monitors, results below.

**Failure history on piclockTG (board #1):**
- Event 1 (2026-04-14): Failed at ~6h50m on EEPROM 2025-07-17 — HC died, required hard power cycle
- Event 2 (2026-04-15 ~01:12 UTC): Failed at ~19h12m on EEPROM 2025-05-08 — usb_set_interface -110 storm, no HC died, recoverable via soft reboot. 733 errors, 366 alsa-ltc restarts.

**Current soak matrix (as of 2026-04-15 06:43 UTC):**
| Unit | IP | RAM | EEPROM | Firmware ID | Role | Uptime | Status |
|------|-----|-----|--------|-------------|------|--------|--------|
| piclockTG | 10.0.0.128 | 2GB | **2025-08-27 (upgraded)** | `000d3ca2` | Test | 4 min | ✅ |
| piclockTD | 10.0.0.131 | 2GB | 2025-05-08 (control) | `69471177` | Control | 17h20m | ✅ |
| piclockR | 10.0.0.121 | 1GB | 2025-08-28 (factory) | `000d3ca2` | Reference | ~20h | ✅ |

**Revised firmware analysis (from official rpi-eeprom release notes):**
- **2025-07-17** — worst: unintended watchdog behavior side effect from `dtoverlay_is_enabled` fix
- **2025-05-08** — partial fix: eliminates undervoltage, extends window but doesn't fully resolve -110 on marginal board
- **2025-08-13** — key fix: "Fix read for cached copy of PMIC sequencer status — previously overwritten by RTC event status." PMIC sequencer controls VL805 power rail sequencing.
- **2025-08-27/28** — best available: includes PMIC sequencer fix + improved HAT+ current reporting (more headroom on 5V rail)
- **Revised hypothesis**: 2025-08-27 is likely the optimal firmware. piclockR on factory 2025-08-28 has been clean at 20h+ while TG on 2025-05-08 failed at 19h. TG now upgraded to 2025-08-27 for direct comparison.

**Key test point:** Watch TG at ~01:40 UTC April 16 (~19h mark from 06:38 UTC reboot). If clean past that, EEPROM 2025-08-27 is confirmed as the fix.

**Monitor restart command (use IP, not .local — mDNS fails over VPN):**
```bash
ssh pi@<IP> 'nohup bash ~/monitor.sh >/dev/null 2>&1 </dev/null & sleep 0.5'
```
Note: monitor does NOT survive reboot — restart manually after each reboot.

**Bug (`install.sh`):** `~/.config/clock-8001/` was created as root when running `sudo bash install.sh` on a fresh system — `sdl-clock` (running as `User=pi`) could not open the log file and exited immediately on every restart attempt. Only affected fresh installs where the directory didn't already exist.

**Fix (`16302b4`):** Added `chown "${INSTALL_USER}:${INSTALL_USER}" "${CONFIG_DIR}"` after `mkdir -p`. One line.

**Process note:** v1.3.0 was released without catching this because soak test units had the directory from a prior deploy. Fresh-install smoke test step added to RELEASING.md (step 7) to prevent recurrence.

### Resolved: Static IP timecode not displaying

**Root cause (2026-04-12):** `sendto(255.255.255.255)` returns `ENETUNREACH` when no default gateway is configured. The gerry `network.ini` comments out the gateway line.

**Fix (`f584ec2`):** Added `resolve_subnet_broadcast()` to `alsa-ltc.c` — uses `getifaddrs()` to resolve `255.255.255.255` → actual subnet broadcast address (e.g., `192.168.8.255`). Prefers wired interfaces (eth*/end*) over wireless. Re-resolves on `sendto()` failure and on every 30s heartbeat (catches DHCP subnet changes). Tested: DHCP reboot x2 ✅, Static reboot ✅.

### Fixed in v1.3.0 (2026-04-10/11/12/13 — alsa-ltc enhancements + hardware debugging + subnet broadcast)
- **alsa-ltc grammar fix + OSC suppression** (`666882e`): Fixed `setted` → `set` in 5 hw_params log messages. Suppressed OSC send error flood — logs once on first failure, logs recovery message with count when send succeeds again.
- **`-v` verbose split** (`c7e3a60`): Card info and 30s heartbeat (frames, LTC count, errors) now always on. `-v` adds activity dots, ALSA HW params, and peak signal level.
- **Configurable `[fps]` argument**: LTC decoder frame rate configurable via CLI (default 25). TouchDesigner source is 30fps.
- **LTC gap detection**: Logs warning when no LTC frame decoded for >1 second. Always on, only fires on anomalies.
- **Hardware fault isolated**: piclockBR Board A (Pi 5) has faulty VL805/PCIe — CM108 USB lockup after 30-100s regardless of dongle, ribbon cable, or software image. Board B is stable. Board A retired.
- **Removed `-v` from service files**: Both Trixie and Buildroot overlay service files updated.
- **Subnet broadcast resolution** (`f584ec2`): `resolve_subnet_broadcast()` using `getifaddrs()` — fixes timecode not displaying when static IP has no default gateway. Prefers wired interfaces, re-resolves on failure and every 30s heartbeat.
- **Install reboot prompt** (`38e74d8`): Added "Reboot to finish installation: sudo reboot" message at end of install.sh output.

## Hardware & Firmware Notes

### Pi 5 EEPROM Firmware — VL805 Stability (updated 2026-04-15)

**Summary of findings across three failure events and three soak units:**

- **2025-07-17** — DO NOT USE. Worst stability. Unintended side effect from `dtoverlay_is_enabled` watchdog fix causes HC died at ~6h50m on board #1.
- **2025-05-08** — Partial improvement. Eliminates undervoltage event. Extends clean window to ~19h on marginal board but does not fully resolve -110. Missing key PMIC sequencer fix.
- **2025-08-27/28** — Recommended. Includes PMIC sequencer status fix (2025-08-13: "Fix read for cached copy of PMIC sequencer status — previously overwritten by RTC event status") and improved PMIC current reporting. piclockR running factory 2025-08-28 clean at 20h+ while TG on 2025-05-08 failed at 19h.

**Upgrade to 2025-08-27 (remote, via SSH):**
```bash
sudo rpi-eeprom-update -d -f /lib/firmware/raspberrypi/bootloader-2712/stable/pieeprom-2025-08-27.bin
sudo reboot
# Verify:
vcgencmd bootloader_version  # → 2025/08/27
```

**Legacy downgrade to 2025-05-08 (still better than 2025-07-17 or newer up to 2025-12-08):**
```bash
sudo rpi-eeprom-update -d -f /lib/firmware/raspberrypi/bootloader-2712/stable/pieeprom-2025-05-08.bin
sudo reboot
```

**Diagnostic signal:** `dmesg | grep Undervoltage` — if present, firmware is drawing too much current during VL805 init. Should be empty on 2025-05-08 and 2025-08-27.

**Monitor tool:** `tools/vl805-usb-health-monitor.sh` — logs temp, USB hub/audio presence, alsa-ltc restarts, USB dmesg errors. 1-minute interval.
Restart after reboot: `ssh pi@<IP> 'nohup bash ~/monitor.sh >/dev/null 2>&1 </dev/null & sleep 0.5'`
(Use IP not .local — mDNS unreliable over VPN)

---

## Buildroot Status (post-merge)

- `buildroot-prototype` branch merged into `master` at `afbbc01` — all Buildroot work is now on master
- Tracking issue: **#28** "Buildroot: post-merge validation and remaining work"
- Build host: pi@cm5.local, `~/buildroot` (Buildroot 2025.02)
- Mesa 25.0.7 (upgraded from 24.0.9 — applied via `buildroot-external/scripts/apply-build-host-patches.sh` — see #29)
- SSH: `BR2_PICLOCKKEY` env var for optional key injection; release images are password-only (`clockworkadmin`)
- **After any clean Buildroot checkout**, run: `buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot`

### Fixed this session (2026-04-06/07)
- **alsa-ltc exit code** (`8a4c562`): All `goto cleanup` paths previously returned 0 — systemd never triggered `Restart=on-failure`. Now defaults to exit code 1; set to 0 only after successful setup; set back to 1 on unrecoverable error (10 consecutive read failures).
- **alsa-ltc snd_pcm_drop** (`8a4c562`): Added `snd_pcm_drop(capture)` before `snd_pcm_close(capture)` to prevent potential hang on broken USB device.
- **Buildroot package recipe** (`71c2321`): Removed `99-alsa-ltc-usb.rules` install line from `clock8002.mk` — file was deleted in v1.2.5 but recipe was not updated, causing Buildroot image builds to fail with `install: cannot stat ... No such file or directory`.
- **USB cable root cause**: piClockN and piclockM failures traced to flat ribbon USB cable (no twisted D+/D- pairs). Shorter twisted cable = intermittent; longer cable = fails every boot. Software recovery: sysfs authorized toggle (`echo 0/1 > /sys/bus/usb/devices/1-1.1/authorized`) proven to clear stuck C-Media capture state.
- **usb-monitor service**: Deployed on piClockN and piclockM — `/home/pi/usb-monitor.sh`, 60s interval, logs USB hub + audio device presence and alsa-ltc restart count.

### Open Buildroot issues
- **#28**: Post-merge validation — Trixie regression test, `broadcast.go` leak fix, boot splash, Wi-Fi AP test
- **#29**: Mesa 25.0.7 + host-xz manual patches — now automated via `buildroot-external/scripts/apply-build-host-patches.sh` (closed)

### Fixed this session (2026-04-01 evening)
- **Sync Time button** (`6f76320`): switched `date --set "@epoch"` → `date -u -s "YYYY-MM-DD HH:MM:SS"` and `hwclock --systohc --utc` → `hwclock -w -u`. POSIX short flags work on both GNU coreutils (Trixie) and BusyBox (Buildroot). Verified: system clock and RTC both set correctly.
- **DT overlays** (`3f70e46` + `123d86c`): config.txt now enables i2c_arm, rtc_bbat_vchg, dwc2 host mode, uart0-3. post-image.sh copies both generic and -pi5 overlay .dtbo variants (firmware's overlay_map.dtb redirects e.g. uart1 → uart1-pi5 on Pi 5). All 5 serial devices now visible (ttyAMA0, ttyAMA1, ttyAMA2, ttyAMA3, ttyAMA10).
- **post-image.sh config.txt sync** (`c0ebb96`): post-image.sh now force-copies board config.txt/cmdline.txt over rpi-firmware cached copies before image generation. Eliminates need for `rpi-firmware-dirclean` after config.txt edits.

### Merge-to-main checklist (tracked in issue #24)
1. SDL rendering fixes (PIXELFORMAT_UNKNOWN, surfaceToABGR8888, SetBlendMode) — master panics without these
2. date/hwclock POSIX short flags in http.go
3. install.sh sudoers rule update (`--systohc --utc` → `-w -u`) — must land with item 2
4. clock8002.service `User=root` vs `User=pi` — needs decision (Buildroot=root, Trixie=pi)
5. piclock-network.sh hostname block moved to end — safe for Trixie, fixes DHCP override race
6. Render diagnostic logging (`logRenderDiag`) — decide: keep or strip
7. Import reordering (cosmetic, harmless)
8. `renderSignal()` missing `SetRenderTarget(nil)` — real bug fix
9. DRM mirror files: `drm_mirror_linux.go`, `drm_mirror_other.go` — new files, direct DRM/KMS mirror for second HDMI
10. `second_display_probe.go` — updated mirror path: `isHDMI1Connected` → `isSpareHDMIConnected`, calls `findSpareHDMIConnector` instead of hardcoded HDMI-A-1
11. DRM cue files: `drm_cue_linux.go`, `drm_cue_other.go` — `updateCueDRMBuffer()` writes icons directly into DRM dumb buffer (Option D)
12. `second_display_probe.go` — cue mode now uses `initDRMMirror()` + `updateCueDRMBuffer()` instead of `fbi`+PNG; removes `bytes`, `image/png`, `os/exec` (fbi) dependency

### Fixed this session (2026-04-01 — EEPROM provisioning)
- **rpi-eeprom package** (`772f5b3`): Added Buildroot external package for Pi 5 EEPROM management (rpi-eeprom-config, rpi-eeprom-update, rpi-eeprom-digest + BCM2712 firmware blobs). Reads EEPROM via nvmem — no vcgencmd needed.
- **Red screen fix**: Changed `BOOT_ORDER` from `0xf461` (SD+NVMe+USB+restart) to `0xf1` (SD-only) on both test units via `rpi-eeprom-config --apply`. Eliminated the red PCIe probe screen caused by bootloader NVMe enumeration on quiet boot.
- **1GB Pi 5 validation**: Image boots and runs on Pi 5 1GB (160MB used / 774MB available).

### Fixed this session (2026-04-02 — EEPROM provisioning final resolution)
- **Boot-partition EEPROM approach removed** (`65af133`): After extensive testing, the `pieeprom.upd` + `recovery.bin` approach baked into the image was proven unreliable. Any Pi 5 whose EEPROM firmware is the same version or newer than Buildroot's blob (Dec 2025) will enter a red screen loop every other boot — the firmware repeatedly tries and fails to apply the downgrade. This includes any unit previously booted with Trixie (which auto-updated EEPROM to Jan 2026).
- **Correct approach**: Manual provisioning via `rpi-eeprom-update -d -f` with a custom config blob. Works unconditionally regardless of installed firmware version. One slow boot (~15s), then clean forever. Documented in README "EEPROM Provisioning (Pi 5)" section.
- **piclockt.local (Pi 5 1GB)** provisioned manually to `BOOT_ORDER=0xf1` ✅ — no more red screen.
- **Issue #26**: Closed — EEPROM provisioning documented as manual step; boot-partition approach removed.

### Fixed this session (2026-04-02 — DRM mirror + cue mode)
- **DRM mirror working** (`2ee57fa`): Root cause found — `findHDMI1Connector()` was hardcoded to target HDMI-A-1, but SDL already renders there. Fix: `findSpareHDMIConnector()` scans all connected HDMI outputs, identifies SDL's CRTC (highest fb_id), picks the other. Both displays now show the clock on piclockBR. DRM state confirmed: plane-2→crtc-92 (SDL, fb=685) + plane-3→crtc-104 (mirror, fb=682).
- **DRM cue mode working** (`a5929ef`): Replaced `fbi`+PNG disk cache path with direct DRM dumb buffer writes (Option D). `probeSecondDisplayOutput()` cue branch calls `initDRMMirror()` then `updateCueDRMBuffer(off)`. `syncSecondDisplayCueDisplay()` calls `updateCueDRMBuffer(desired)` — renders icon via `renderCueVisualImage()` and writes XRGB8888 directly into the dumb buffer. No `fbi` binary or `/dev/fb0` required. Web GUI toggle (PerfectCue section) switches modes live without restart. Verified working on piclockBR at `a5929ef`.

### Remaining work
- DRM mirror simplification: master swap dance (DROP/SET_MASTER) may be unnecessary now — targeting fbcon's CRTC, not SDL's
- DRM mirror robustness testing: service restart, HDMI hot-plug, extended runtime
- Trixie regression testing

## Issue #23 Status (Dual HDMI Output)

- **DRM mirror working on Buildroot** (`2ee57fa` on `buildroot-prototype`) ✓
- **DRM cue mode working on Buildroot** (`a5929ef` on `buildroot-prototype`) ✓
- Implementation uses direct DRM/KMS ioctls (not SDL second window) for both modes.
- Behavior:
  - `--cue-second-display`: HDMI-2 shows full-screen PerfectCue icons (forward/reverse/blank) written directly into DRM dumb buffer — no `fbi`, no `/dev/fb0`.
  - No flag (default): HDMI-2 mirrors the main clock via DRM dumb buffer + CPU pixel copy from SDL renderer.
- Web GUI toggle (PerfectCue section checkbox) switches modes live — no restart required.
- Key files: `drm_mirror_linux.go`, `drm_mirror_other.go`, `drm_cue_linux.go`, `drm_cue_other.go`, `second_display_probe.go`.
- `findSpareHDMIConnector()` is fully dynamic — scans all HDMI outputs, identifies SDL's CRTC by highest fb_id, picks the spare. No hardcoded connector/CRTC IDs.
- Issue remains open pending Trixie validation and robustness testing.

## Hot-Plug Policy (Current Expected Behavior)

- Do not crash on HDMI plug/unplug events.
- Main clock on primary output must continue running regardless of HDMI-2 state.
- Icon mode (`fbi`) should tolerate disconnect/reconnect and resume when HDMI-2 is available.
- Mirror mode (DRM/KMS direct) uses a dumb buffer on the spare CRTC. HDMI-2 reconnect likely requires process restart since CRTC/connector assignment is fixed at init time.
- Test matrix still required before closing issue #23:
  - Boot with HDMI-1 only
  - Boot with HDMI-2 only
  - Boot with both connected
  - Unplug/replug HDMI-2 while running in icon mode
  - Unplug/replug HDMI-2 while running in mirror mode

## Recent Release Notes

- v1.1.7 includes:
  - Dynamic app-version stamping for clock.ini via `clock.AppVersionForConfig()` (uses injected git tag; strips leading `v`).
  - Config rewrite/save paths now use dynamic app-version instead of hardcoded `clock.Version`.
  - Gerry profile updates preserved in `clock.ini.gerry` (text face, Limitimer/TOD-LTC/Playback, row colors, limitimer receive mode).
  - README quick-install URLs updated to v1.1.7.
- GitHub release notes continue to use `.github/release-notes-template.md` with `__VERSION__` substitution.

## OLED Splash Version Overlay

- OLED startup logo now overlays build version in lower-right, raised slightly from bottom.
- Regex now supports major versions beyond v0 (`v[0-9]+...`), so v1.x tags display correctly.

## Release Process (Current)

1. Update `CHANGELOG.md`.
2. Ensure `README.md` quick-install commands point to the new version.
3. Commit, tag, and push:
   - `git tag v1.x.y`
   - `git push origin v1.x.y`
4. Build on pi5start from a fresh clone at the tag:
   - `make release-all GIT_TAG=v1.x.y` (produces default + gerry tarballs)
5. Publish GitHub release with both tarballs.
6. Deploy to piclock.local and run installer.
7. Start and verify `clock8002`, `alsa-ltc`, and `oled_daemon`.
8. Report deployed short commit hash.

## Repository Instructions Already Added

- Build machine: `pi@pi5start.local`
- Test machine: `pi@piclock.local`
- Ask before running tests.
- Include both default and gerry artifacts for each release.
- Use repository version line (`v1.x` and onward), ignore inherited upstream `v4.x` tags.
- Update README quick-install URL/version during release cuts.
- Avoid `pkill -f /opt/clock8002/sdl-clock` and `pkill -f /opt/clock8002/alsa-ltc` in one-shot SSH deploy commands (can trigger SSH exit 255).
- Prefer `systemctl stop` + `systemctl kill` for teardown.
- Run installer with log capture and explicit exit check.

## Hard Rules (Do Not Skip)

- Build host rule: build/release artifacts for clock8002 are authoritative only when built on `pi@pi5start.local` from a fresh clone at target ref.
- Do not use Mac local build results for release/deploy validation.
- Gerry variant rule: deployment is only valid when both `/boot/firmware/piclock/clock.ini` and `/boot/firmware/piclock/network.ini` match Gerry settings.
- Installer behavior note: `install.sh` preserves existing `/boot/firmware/piclock/clock.ini` and only installs packaged clock.ini on fresh install; existing units may require explicit config copy.

## Useful Commands

- Check local working tree:
  - `git status --short`
- Build release artifacts on pi5start (fresh clone approach):
  - `ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && git checkout v1.x.y && make release-all GIT_TAG=v1.x.y'`
- Deploy release tarball to piclock.local from local Mac relay:
  - `scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-v1.x.y-default-linux-arm64.tar.gz /tmp/`
  - `scp /tmp/clock8002-v1.x.y-default-linux-arm64.tar.gz pi@piclock.local:/tmp/`
  - `ssh pi@piclock.local 'set -e; sudo systemctl stop clock8002.service alsa-ltc.service oled_daemon.service || true; sudo systemctl kill clock8002.service alsa-ltc.service oled_daemon.service || true; mkdir -p /tmp/clock8002-install && rm -rf /tmp/clock8002-install/clock8002-v1.x.y-default-linux-arm64; tar xzf /tmp/clock8002-v1.x.y-default-linux-arm64.tar.gz -C /tmp/clock8002-install; cd /tmp/clock8002-install/clock8002-v1.x.y-default-linux-arm64; sudo bash install.sh > /tmp/clock8002-install-v1.x.y.log 2>&1; echo INSTALL_EXIT:$?; sudo systemctl start clock8002.service alsa-ltc.service oled_daemon.service'`
- Verify services on piclock:
  - `ssh pi@piclock.local 'systemctl is-active clock8002 alsa-ltc oled_daemon'`
- Force-apply gerry config pair on existing unit:
  - `ssh pi@piclock.local 'sudo cp /tmp/clock8002-v1.x.y-gerry-linux-arm64/clock.ini /boot/firmware/piclock/clock.ini && sudo cp /tmp/clock8002-v1.x.y-gerry-linux-arm64/network.ini /boot/firmware/piclock/network.ini && sudo reboot'`

## Release Notes Template Workflow

- Template file: `.github/release-notes-template.md`
- Placeholder token: `__VERSION__`
- Generate release notes file:
  - `VERSION=v1.x.y; sed "s/__VERSION__/${VERSION}/g" .github/release-notes-template.md > /tmp/release-notes-${VERSION}.md`
- Publish release with templated notes:
  - `gh release create "${VERSION}" "clock8002-${VERSION}-default-linux-arm64.tar.gz" "clock8002-${VERSION}-gerry-linux-arm64.tar.gz" --title "${VERSION}" --notes-file "/tmp/release-notes-${VERSION}.md"`

## Dev-Deploy Workflow (Feature Branch Testing)

Use this when testing a feature branch on piclock — **not** a formal release.
`install.sh` must always run on the target machine from a flat release directory (never from the source tree — `*.ttf` won't be found).

```bash
# 1. Clone branch, build, and package on pi5start
ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone --depth 1 --branch BRANCH https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && make release NETWORK_CONFIG=default'

# 2. Relay tarball through Mac to piclock
#    (GIT_TAG defaults to nearest ancestor tag, e.g. v1.0.3)
scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-*-default-linux-arm64.tar.gz /tmp/
scp /tmp/clock8002-*-default-linux-arm64.tar.gz pi@piclock.local:/tmp/

# 3. Extract and install on piclock
ssh pi@piclock.local 'cd /tmp && tar xzf clock8002-*-default-linux-arm64.tar.gz && cd clock8002-*-default-linux-arm64 && sudo bash install.sh'

# 4. Verify
ssh pi@piclock.local 'systemctl is-active clock8002'
```

> **After every deploy to the test unit, report the short GitHub commit hash (first 7 characters) that was deployed.**

## Buildroot Known Issues

- **Mesa 25.0.7 + host-xz patches**: Automated via `buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot` — run after any clean Buildroot checkout. See #29 (closed).
- **SSH key hardcoded**: Personal SSH public key in `post-build.sh` — should be env-var driven. Tracked in #30.
- **No root password set**: Buildroot images have blank root password → SSH password login blocked on release images. Tracked in #30 (fix: `clockworkadmin`).
- **`broadcast.go` UDP socket leak**: `singleAddr()`/`broadcastAll()` replace connections without closing old ones — ~1.9 MB/hr leak on Buildroot with stable alsa-ltc; much faster with crash-looping alsa-ltc. Tracked in #28.

## Stability Test — piclockT.local (in progress)

- Unit: Pi 5 1GB, Buildroot image `65af133` (v1.2.3), no swap
- Started: 2026-04-03 06:27 UTC
- Baseline RSS: ~44 MB; leak rate: ~1.9 MB/hr
- 21h checkpoint (03:27 UTC): RSS 86 MB, all services active, no OOM
- 24h checkpoint due: ~2026-04-04 06:27 UTC
- Monitor: `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piclockT.local 'cat /root/stability.log'`
- Pass criteria: RSS ±50 MB from baseline, no OOM, mem-avail > 400 MB, all services active

## Documentation updated this session (2026-04-03)

- `README.md`: restructured — Platform folded into Requirements, EEPROM moved to top-level section, Service Operations trimmed, Creating a Release removed, config files table added
- `RELEASING.md`: new — full Trixie + Buildroot release procedure
- `buildroot-external/README.buildroot.md`: new — Buildroot build/flash/SSH/config reference
- `buildroot-external/README.buildroot.txt`: deleted (superseded)
- Release procedure now lives in `RELEASING.md`; see also `buildroot-external/README.buildroot.md`

### README accuracy audit (2026-04-03 afternoon) — all claims now verified
- Fixed: oled.ini description (`1e33012`)
- Fixed: OLED daemon + I2C enable merged into one conditional bullet (`06839ab`)
- Fixed: clock.ini path clarified as `/boot/piclock/clock.ini`; symlink at `~/.config/clock-8001/clock.ini` noted (`8a675fd`)
- Fixed: `--dump-config` moved from Config section to Building from Source section (`f9a0411`)
- Verified correct: web UI credentials (admin/clockwork), network.ini description, installer bullet list, config file paths

## alsa-ltc Enhancements (2026-04-11)

Committed changes in `v4/alsa-ltc.c`, `v4/alsa-ltc.service`, and Buildroot overlay.

### Binary changes (`alsa-ltc.c`)
- **Always-on diagnostics** (no flag needed): ALSA card info at startup, 30s heartbeat with frame count, LTC decode count, and error tally.
- **`-v` verbose flag**: Activity dots (`.` per decoded LTC frame), ALSA hardware params (buffer_size/period_size/periods) at startup, peak audio signal level (0–32767) in heartbeat.
- **Configurable sample rate**: Default changed 48000 → 44100 (matches upstream). Optional 4th positional arg `[sample-rate]`. Prints warning if hardware negotiates different rate.
- **Early DISCONNECT exit**: `snd_pcm_state()` checked on read errors — if `DISCONNECTED` or `-ENODEV`, exits immediately with explicit message instead of burning 10 retries. PCM state name added to all error log lines.
- Usage: `alsa-ltc [-v] <device> <ip> <port> [sample-rate]`

### Service file changes (`alsa-ltc.service` + Buildroot overlay)
- **`-v` removed from production**: Heartbeat and card info are always-on; add `-v` to command line for USB debugging.
- **`ExecStopPost=+` USB reset**: On failure exit, scans for C-Media USB devices and toggles sysfs `authorized` 0→1 to reset the dongle before the next restart.
- **`Restart=on-failure`**: Replaces `always` — restarts only on error exits, not clean stops.
- **`RestartSec=5`**: Reduced from 30s — faster recovery with early DISCONNECT exit.
- **`StartLimitBurst=5` / `StartLimitIntervalSec=60`**: Caps restart attempts on deeply locked hardware.
- **Dual service file rule**: Both `v4/alsa-ltc.service` and `buildroot-external/.../rootfs-overlay/.../alsa-ltc.service` must be kept in sync.

### Hardware debugging (piclockBR CM108 lockup)
- **Root cause**: Faulty PCIe ribbon cable between Pi 5 SoC and VL805 xHCI USB controller.
- Symptom: `usb_set_interface failed (-110/-62)` after 30-100s on every boot.
- Isolation: swapped dongle (still fails), swapped Pi board (works), swapped PCIe ribbon (works) → cable was the fault.
- Resolution: replaced ribbon cable; CM108 stable past 119s+ on original board.

### Documentation
- CHANGELOG.md: "Version 1.2.10 (unreleased)" section.
- README.md: "alsa-ltc command-line options" table, "USB recovery" subsection.

### Upstream binary analysis (root@192.168.8.245)
- Buildroot 2025.11, kernel 6.12.41-v8, BusyBox init, 22KB stripped aarch64
- Sample rate 44100 confirmed via `objdump -s -j .data`
- Activity dot always on; OSC pretty-printer is dead code
- Missing: retry loop, plughw, snd_pcm_drop, error counter, --version, SO_BROADCAST, signal handler

### Stability test (Apr 11, ~6h)
- piclockBR: clean — 0 USB errors
- piclockTG (Trixie): CM108 lockup at ~6h (`usb_set_interface -110`), rebooted to recover

## SDL3 Migration Status

**Branch:** `feature/sdl3-migration` — HEAD `cd26fde`  
**Full details:** [HANDOFF-SDL3.md](HANDOFF-SDL3.md)

| Phase | Status | Notes |
|---|---|---|
| Phase 1 — Branch established | ✅ | From upstream `f0412525` |
| Phase 2 — SDL3 Buildroot packages | ✅ | sdl3, sdl3-ttf, sdl3-image packages created |
| Phase 3 — Port clock8002 additions | ✅ | All master additions ported; Buildroot configs fixed |
| Phase 4 — Hardware validation | 🔄 | Image built (`a6756f3`); on Mac Desktop — awaiting flash to piclockBR |
| Phase 5 — Read-only rootfs | ⬜ | Deferred until Phase 4 proven |

**Buildroot image:** `piclockBR-a6756f3-sdcard.img` on Mac Desktop (768 MB). Build completed 2026-04-14. Flash manually to piclockBR SD card to begin Phase 4 hardware validation.

**Key changes from SDL2:**
- CGO eliminated — `GOOS=linux GOARCH=arm64 go build` only
- Binary: `sdl-clock` → `sdl3-clock`
- SDL3 loaded via purego/dlopen at runtime (libSDL3.so v3.4.0, libSDL3_ttf.so v3.2.2, libSDL3_image.so v3.2.6)

### Trixie SDL3 Backport Status (2026-05-06)

- Branch: `trixie-sdl3`
- Scope completed in working tree:
  - Added `v4/cmd/sdl3-clock/` from `master`
  - Switched Trixie build/install/service paths from `sdl-clock` to `sdl3-clock`
  - Updated `v4/go.mod`, `v4/go.sum`, vendored SDL3/purego deps, and Buildroot package recipe
- Local build verification (macOS host cross-build):
  - `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOFLAGS="-mod=mod" go build -o sdl3-clock ./cmd/sdl3-clock`
  - Output binary produced: `v4/sdl3-clock` (aarch64 ELF)
- Release packaging on local macOS is expected to fail for `alsa-ltc` (missing ALSA headers). Build/release tarball should be produced on CM5.
- CM5 resource note:
  - A Buildroot job was active (`br-root-ram-74439b3-k641247-usbfix2`), so Trixie SDL3 tarball build was intentionally deferred to avoid contention.
  - Safe sequence: wait for `/tmp/br-root-ram-74439b3-k641247-usbfix2.exit` to report `BR_BUILD_EXIT:0`, then run Trixie `make release` and copy tarball to Mac Desktop.
- Runtime dependency risk on Trixie:
  - `sdl3-clock` requires `libSDL3.so`, `libSDL3_ttf.so`, and `libSDL3_image.so` at runtime; package availability on target image must be validated during install test.

## Next Suggested Release

- No immediate release pending.
- Prerequisites before next release: resolve #28 (Trixie regression test + broadcast.go fix), #29 (build host patches), #30 (SSH/password).
- alsa-ltc enhancements above should be included in next release after build/test validation.
