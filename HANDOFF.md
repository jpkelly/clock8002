# Clock8002 Handoff

Last updated: 2026-04-11

## Current State

- Repository: jpkelly/clock8002
- Active release line: v1.x
- Latest tagged release: **v1.2.9** — Trixie tarballs on GitHub
- Trixie tarballs: `clock8002-v1.2.9-default-linux-arm64.tar.gz`, `clock8002-v1.2.9-gerry-linux-arm64.tar.gz`
- Buildroot SD card image: `piclockBR-71c2321-sdcard.img` (on Mac Desktop) — contains v1.2.6 binaries
- Test unit piclockBR.local: running Buildroot image built from `759dafe` (pre-v1.2.6); stability soak test in progress (started 2026-04-09)
- Test unit piclockG.local: running Trixie, stability soak test in progress (started 2026-04-09)
- piClockN.local: Trixie v1.2.6, shorter USB cable (intermittent failures), usb-monitor service running
- piclockM.local: Trixie v1.2.6, longer USB cable (fails every boot), usb-monitor service running
- `buildroot-prototype` branch: fully merged into master

## Buildroot Status (post-merge)

- `buildroot-prototype` branch merged into `master` at `afbbc01` — all Buildroot work is now on master
- Tracking issue: **#28** "Buildroot: post-merge validation and remaining work"
- Build host: pi@pi5start.local, `~/buildroot` (Buildroot 2025.02)
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

## alsa-ltc Enhancements (2026-04-11, uncommitted)

Changes in `v4/alsa-ltc.c` and `v4/alsa-ltc.service` — not yet committed, built, or tested:

### Binary changes (`alsa-ltc.c`)
- **`-v` verbose flag**: Activity dot (`.` per decoded frame), ALSA card info at startup, 30s heartbeat with frame count and error tally.
- **Configurable sample rate**: Default changed 48000 → 44100 (matches upstream). Optional 4th positional arg `[sample-rate]`. Prints warning if hardware negotiates different rate.
- **Early DISCONNECT exit**: `snd_pcm_state()` checked on read errors — if `DISCONNECTED` or `-ENODEV`, exits immediately with explicit message instead of burning 10 retries. PCM state name added to all error log lines.
- Usage: `alsa-ltc [-v] <device> <ip> <port> [sample-rate]`
- Backwards compatible: existing `alsa-ltc - 255.255.255.255 1245` works unchanged.

### Service file changes (`alsa-ltc.service`)
- **`-v` enabled by default**: Verbose diagnostics captured in journal for USB troubleshooting.
- **`ExecStopPost=+` USB reset**: On failure exit, scans for C-Media USB devices and toggles sysfs `authorized` 0→1 to reset the dongle before the next restart.
- **`Restart=on-failure`**: Replaces `always` — restarts only on error exits (code 1), not clean stops (code 0). Leverages proper exit codes added in v1.2.6.
- **`RestartSec=5`**: Reduced from 30s — faster recovery now that early DISCONNECT exit avoids wasted retries.
- **`StartLimitBurst=5` / `StartLimitIntervalSec=60`**: Caps restart attempts — prevents infinite crash-loop if CM108 is deeply locked (sysfs reset can't recover).

### Documentation
- CHANGELOG.md: added "Version 1.2.10 (unreleased)" section.
- README.md: added "alsa-ltc command-line options" subsection with usage and argument table.

### Upstream binary analysis (root@192.168.8.245)
- Buildroot 2025.11, kernel 6.12.41-v8, BusyBox init, 22KB stripped aarch64
- Sample rate 44100 confirmed via `objdump -s -j .data`
- Activity dot always on; OSC pretty-printer is dead code
- Missing: retry loop, plughw, snd_pcm_drop, error counter, --version, SO_BROADCAST, signal handler

### Stability test (Apr 11, ~6h)
- piclockBR: clean — 0 USB errors
- piclockTG (Trixie): CM108 lockup at ~6h (`usb_set_interface -110`), rebooted to recover

## Next Suggested Release

- No immediate release pending.
- Prerequisites before next release: resolve #28 (Trixie regression test + broadcast.go fix), #29 (build host patches), #30 (SSH/password).
- alsa-ltc enhancements above should be included in next release after build/test validation.
