# Clock8002 Handoff

Last updated: 2026-04-01

## Current State

- Repository: jpkelly/clock8002
- Active release line: v1.x
- Latest release published: v1.1.7
- Test machine deployment status: v1.1.7 gerry installed on piclock.local, services active (`clock8002`, `alsa-ltc`, `oled_daemon`)
- Deployed commit on piclock: `c46dd4d`
- Issue #23 implementation is merged to `master` via squash commit `d62c48e`
- README update for 2nd HDMI feature wording is on `master` at `c4f67b5`

## Buildroot Prototype (Issue #24)

- Branch: `buildroot-prototype`, HEAD: `772f5b3` (pushed)
- Test units: piclockBR.local — Pi 5 2GB (10.0.0.163), Pi 5 1GB (10.0.0.171)
- SD card image: `piclockBR-772f5b3-sdcard.img` (768 MB), flashed and running on both units
- Build host: pi@pi5start.local, `~/buildroot` (Buildroot 2025.02)
- All Critical items complete — sdl-clock fully operational with GLES2/KMSDRM
- Mesa 25.0.7 (upgraded from 24.0.9 — manual change on build host, not in git)

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

### Fixed this session (2026-04-01 — EEPROM provisioning)
- **rpi-eeprom package** (`772f5b3`): Added Buildroot external package for Pi 5 EEPROM management (rpi-eeprom-config, rpi-eeprom-update, rpi-eeprom-digest + BCM2712 firmware blobs). Reads EEPROM via nvmem — no vcgencmd needed.
- **Red screen fix**: Changed `BOOT_ORDER` from `0xf461` (SD+NVMe+USB+restart) to `0xf1` (SD-only) on both test units via `rpi-eeprom-config --apply`. Eliminated the red PCIe probe screen caused by bootloader NVMe enumeration on quiet boot.
- **1GB Pi 5 validation**: Image boots and runs on Pi 5 1GB (160MB used / 774MB available).

### Remaining work
- **EEPROM provisioning in image build** (TODO): Current image has leftover `recovery.bin` + `pieeprom.upd` from manual EEPROM fix — accidentally auto-provisions new Pis. Should be made intentional: add post-image hook or clock8002.mk step that pre-builds `pieeprom.upd` with `BOOT_ORDER=0xf1` and places it + `recovery.bin` on boot partition. Every fresh Pi gets provisioned on first boot (one slow boot, then clean forever).
- Second HDMI output
- Trixie regression testing

## Issue #23 Status (Dual HDMI Output)

- Core feature is implemented and merged.
- Behavior now:
  - `cue-second-display = true`: HDMI-2 shows full-screen PerfectCue icons (forward/reverse/blank) via `fbi`.
  - `cue-second-display = false`: HDMI-2 mirrors the main clock in real-time via an SDL second window.
- Issue remains open pending hard testing and hot-plug validation.

## Hot-Plug Policy (Current Expected Behavior)

- Do not crash on HDMI plug/unplug events.
- Main clock on primary output must continue running regardless of HDMI-2 state.
- Icon mode (`fbi`) should tolerate disconnect/reconnect and resume when HDMI-2 is available.
- Mirror mode (SDL/kmsdrm second display) may require process restart after HDMI-2 reconnect, because display enumeration is effectively fixed at SDL startup.
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
- Gerry variant rule: deployment is only valid when both `/boot/piclock/clock.ini` and `/boot/piclock/network.ini` match Gerry settings.
- Installer behavior note: `install.sh` preserves existing `/boot/piclock/clock.ini` and only installs packaged clock.ini on fresh install; existing units may require explicit config copy.

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
  - `ssh pi@piclock.local 'sudo cp /tmp/clock8002-v1.x.y-gerry-linux-arm64/clock.ini /boot/piclock/clock.ini && sudo cp /tmp/clock8002-v1.x.y-gerry-linux-arm64/network.ini /boot/piclock/network.ini && sudo chown pi:pi /boot/piclock/clock.ini /boot/piclock/network.ini && sudo reboot'`

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

- **Sync Time button broken**: BusyBox `date` doesn't support `--set "@epoch"` (needs `-s`); BusyBox `hwclock` doesn't support `--systohc --utc` (needs `-w -u`). Running as root means sudo fallback is unnecessary. Fix: update `setTimeHandler()` and `syncHardwareClock()` in `v4/cmd/sdl-clock/http.go`.
- **System clock wrong after boot**: Shows Oct 2024 — no NTP or manual time set. Depends on sync time fix.
- **Mesa 25.0.7 changes not in git**: Manual modifications on pi5start build host — must be re-applied after clean Buildroot checkout.
- **host-xz libtool bug**: `acl_cv_wl="-Wl,"` workaround in xz.mk — also not in git.

## Next Suggested Release

- No immediate release pending.
- Before next release: complete issue #23 hard testing/hot-plug matrix and document results in issue #23.
