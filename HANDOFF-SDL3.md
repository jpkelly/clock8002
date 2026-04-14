# SDL3 Migration Handoff

**Branch:** `feature/sdl3-migration`  
**Started:** 2026-04-13  
**Base:** upstream clock-8001 commit `f0412525` (Feature: Add SDL3 version of the clock, 2026-04-08)

---

## Contents

- [Strategic Decision](#strategic-decision)
- [Upstream Remote](#upstream-remote)
- [Runtime SDL3 Library Versions](#runtime-sdl3-library-versions)
- [Migration Phases](#migration-phases)
  - [✅ Phase 1 — Branch established](#-phase-1--branch-established-commit-485f801)
  - [✅ Phase 2 — SDL3 Buildroot packages](#-phase-2--sdl3-buildroot-packages-commit-dbbab85)
  - [✅ Phase 3 — Port clock8002 additions](#-phase-3--port-clock8002-additions-commits-1624c60852a99-2026-04-1314)
  - [⬜ Phase 4 — Hardware validation](#-phase-4--hardware-validation-on-piclockbr---next)
  - [⬜ Phase 5 — Read-only rootfs](#-phase-5--read-only-rootfs-appliance-hardening)
- [What Carries Forward UNCHANGED](#what-carries-forward-unchanged)
- [What Changes](#what-changes)
- [SDL3 Architecture Notes](#sdl3-architecture-notes)
  - [libudev — intentionally disabled](#libudev--intentionally-disabled--dsdl_libudevoff)
- [Build Workflow](#build-workflow-unchanged-from-master)

---

## Strategic Decision

clock8002 is migrating from SDL2 (CGO, complex cross-compilation) to SDL3 via the
`github.com/Zyko0/go-sdl3` purego wrapper. This eliminates CGO from the Go build
entirely — `GOOS=linux GOARCH=arm64 go build` is all that is needed.

The upstream project `gitlab.com/clock-8001/clock-8001` added SDL3 support on 2026-04-08
with a new binary `v4/cmd/sdl3-clock`. This branch starts from that upstream commit and
ports all clock8002-specific additions onto it.

### Key choices
- **SDL2 dropped entirely.** Hard cut to `sdl3-clock` only. No fallback binary maintained.
- **Go module path unchanged.** Kept as `gitlab.com/clock-8001/clock-8001/v4` to allow
  clean upstream merges. The binary is an appliance — no one imports it as a library.
- **Build machine unchanged.** pi@cm5.local runs Buildroot as before.
- **Read-only rootfs deferred.** squashfs + overlayfs is Phase 5, after SDL3 is proven on hardware.

---

## Upstream Remote

```bash
git remote add upstream https://gitlab.com/clock-8001/clock-8001.git
git fetch upstream
```

To pull future upstream changes onto this branch:
```bash
git fetch upstream
git merge upstream/master
```

---

## Runtime SDL3 Library Versions

These must match `github.com/Zyko0/go-sdl3 v0.1.0`. All are loaded at runtime via
purego/dlopen — NOT linked at build time.

| Library | Version |
|---|---|
| libSDL3.so | SDL3 **v3.4.0** |
| libSDL3_ttf.so | SDL3_ttf **v3.2.2** |
| libSDL3_image.so | SDL3_image **v3.2.6** |

---

## Migration Phases

### ✅ Phase 1 — Branch established (commit `485f801`)
- `upstream` remote added
- `feature/sdl3-migration` created from `f0412525`, pushed to GitHub

### ✅ Phase 2 — SDL3 Buildroot packages (commit `dbbab85`)
Three packages created in `buildroot-external/package/`:

| Package | Provides | Version | Notes |
|---|---|---|---|
| `sdl3` | libSDL3.so | 3.4.0 | KMSDRM backend, ALSA, no X11/Wayland/OpenGL |
| `sdl3-ttf` | libSDL3_ttf.so | 3.2.2 | harfbuzz + freetype |
| `sdl3-image` | libSDL3_image.so | 3.2.6 | PNG + JPEG only |

Config.in registration deferred to Phase 3 (arrives with full buildroot-external port).

### ✅ Phase 3 — Port clock8002 additions (commits `1624c60`–`85d2a99`, 2026-04-13/14)

All clock8002-specific additions from `master` ported onto this branch:

**Commits in this phase:**

| Commit | Description |
|---|---|
| `1624c60` | phase3: port clock8002 additions from master (alsa-ltc, install.sh, oled, Makefile, service files, tools, pi345build, configs) |
| `f00734e` | vendor: regenerate vendor directory (`go mod vendor`) |
| `5a9c063` | buildroot: defconfig.sample SDL2 → SDL3 |
| `893d522` | buildroot: clock8002 Config.in add `select` for SDL3/SDL3_ttf/SDL3_image/libltc |
| `f887949` | buildroot: external Config.in add SDL3 package sources |
| `85d2a99` | buildroot: sdl3 Config.in remove `select BR2_PACKAGE_EUDEV` (conflicts with systemd udev) |
| `52c832c` | buildroot: sdl3 drop udev dependency, disable `SDL_LIBUDEV` (sdl3.mk had `eudev` in `SDL3_DEPENDENCIES`) |
| `df7a7fb` | docs: HANDOFF-SDL3 document libudev disabled rationale |

**What was ported:**
- `buildroot-external/` — full directory (clock8002 package, configs, board overlay, scripts)
- `v4/alsa-ltc.c`, `v4/alsa-ltc.service`
- `v4/install.sh` — binary name updated: `sdl-clock` → `sdl3-clock`
- `v4/clock.ini.gerry`, `v4/clock_g.ini`, `v4/network.ini.gerry`
- `v4/piclock-network.sh`, `v4/piclock-network.service`
- `v4/clock8002.service` — `ExecStart` updated to `sdl3-clock`
- `v4/oled/` daemon
- `tools/`, `pi345build/`
- `v4/Makefile` — merged master release infra (release, gerry variant, alsa-ltc) + SDL3 build targets
- `v4/clock/` Go files from master (clock-8002 branding, network detection)

**clock8002.mk change (Phase 3b, included in `1624c60`):**  
Dropped CGO entirely. Was:
```makefile
CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
  CGO_CFLAGS="-I$(STAGING)/usr/include/SDL2" \
  CGO_LDFLAGS="-L$(STAGING)/usr/lib -lSDL2" \
  GOOS=linux GOARCH=arm64 go build ./cmd/sdl-clock
```
Now:
```makefile
GOOS=linux GOARCH=arm64 go build ./cmd/sdl3-clock
```

**defconfig changes (Phase 3c, `5a9c063`):**
- Removed: `BR2_PACKAGE_SDL2` and related Mesa 25.0.7 manual patch entries
- Added: `BR2_PACKAGE_SDL3=y`, `BR2_PACKAGE_SDL3_IMAGE=y`, `BR2_PACKAGE_SDL3_TTF=y`

**Buildroot Config.in fixes (`893d522`, `f887949`, `85d2a99`):**
- `clock8002/Config.in`: added `select` statements for all SDL3 + libltc deps (required by Buildroot dependency checker)
- `buildroot-external/Config.in`: sourced SDL3, SDL3_ttf, SDL3_image packages (were missing, caused "dependency chain" error)
- `sdl3/Config.in`: removed `select BR2_PACKAGE_EUDEV` (eudev conflicts with systemd, which already provides udev)

**State after Phase 3:** Branch HEAD `df7a7fb` on `feature/sdl3-migration`. cm5 `~/buildroot/configs/clock8002_rpi5_defconfig` updated directly with SDL3 entries. **Build confirmed running on cm5 as of 2026-04-14. Next: wait for build to complete → flash piclockBR → Phase 4 hardware validation.**

### ⬜ Phase 4 — Hardware validation on piclockBR  ← **NEXT**
Key items to verify:
- Clock renders on HDMI via KMSDRM (no GLES2 patches needed — SDL3 handles natively)
- USB audio / alsa-ltc working (C code unchanged)
- OLED daemon working
- Web config UI accessible
- LTC timecode display working
- PerfectCue, Mitti, Millumin integrations working

### ⬜ Phase 5 — Read-only rootfs (appliance hardening)
- `BR2_TARGET_ROOTFS_SQUASHFS=y`
- overlayfs + tmpfs init script
- Update partition layout: FAT (boot) + squashfs (rootfs), no ext4
- All persistent state already on FAT `/boot/piclock/` — no clock code changes needed

---

## What Carries Forward UNCHANGED

Everything that was painful to get working in Buildroot is retained:

- Kernel pin (`359f37f0` / 6.12.47) — fixes USB audio xHCI regression on VL805
- `otg_mode=1` + `dtoverlay=dwc2,dr_mode=host` — fixes VL805 USB-C interference
- Module autoloading (host-kmod XZ, depmod masked)
- WiFi AP (wpa_supplicant D-Bus/AP support)
- OLED daemon + Python luma packages
- Boot splash (fbv + bootsplash-fbv.service)
- Power button (systemd-logind)
- `piclock-network.service` + RequiresMountsFor=/boot
- DT overlays, post-image.sh, post-build.sh
- Pi 5 D0 DTB (`bcm2712d0-rpi-5-b.dtb`), `os_check=0`
- `apply-build-host-patches.sh` on cm5

## What Changes

- SDL2 Buildroot packages → SDL3 packages (Phase 2)
- Mesa 25.0.7 manual patch → likely no longer needed
- clock8002.mk Go build step → drops CGO entirely
- Binary name: `sdl-clock` → `sdl3-clock`
- GLES2 rendering fixes (`PIXELFORMAT_UNKNOWN`, `surfaceToABGR8888`, `SetBlendMode`) → dropped

---

## SDL3 Architecture Notes

- `sdl3_libs_linux.go` calls `sdl.LoadLibrary(sdl.Path())` at startup — dlopen `libSDL3.so.0`
- KMSDRM detection is native in SDL3: auto-sizes to display mode, no manual hacks needed
- Cross-compile: `GOOS=linux GOARCH=arm64 go build` — no sysroot, no headers
- `arm64` Linux not on go-sdl3's official supported list, but upstream CI builds it successfully
- Hardware validation (Phase 4) needed to confirm KMSDRM works on Pi 5

### libudev — intentionally disabled (`-DSDL_LIBUDEV=OFF`)

SDL3's KMSDRM backend optionally uses libudev to watch `/dev/input` for joystick/gamepad
hotplug events. We explicitly disable it (`SDL3_CONF_OPTS += -DSDL_LIBUDEV=OFF`, no
`eudev` in `SDL3_DEPENDENCIES`) because:

- The clock appliance has no joysticks or HID input devices to enumerate.
- The Buildroot image uses systemd (which provides udev); adding eudev as a separate
  dependency causes a hard Buildroot error ("both systemd and eudev selected as udev
  providers").
- SDL3's ALSA audio backend (`-DSDL_ALSA=ON`) uses libasound directly — it does not go
  through libudev. The LTC USB audio device is handled entirely by the separate `alsa-ltc`
  binary via libasound and is unaffected by this setting.

---

## Build Workflow (unchanged from master)

```bash
# On Mac — edit, commit, push
git push origin feature/sdl3-migration

# On cm5 — pull latest and start/restart build inside screen
ssh pi@cm5.local
screen -S sdl3-build          # create new session (or use -r to reattach)
cd ~/clock8002 && git pull --ff-only origin feature/sdl3-migration
cd ~/buildroot && make BR2_EXTERNAL=~/clock8002/buildroot-external clock8002_rpi5_defconfig
make > /tmp/br-sdl3-build.log 2>&1

# Detach from screen without killing: Ctrl-A then D

# On Mac — monitor build log without attaching to screen
ssh pi@cm5.local 'tail -f /tmp/br-sdl3-build.log'

# Reattach to screen — must SSH interactively first, then reattach
ssh pi@cm5.local
screen -r sdl3-build

# Check if build finished (tail last 20 lines)
ssh pi@cm5.local 'tail -20 /tmp/br-sdl3-build.log'

# Transfer image
scp pi@cm5.local:~/buildroot/output/images/sdcard.img \
    ~/Desktop/piclockBR-<HASH>-sdcard.img
```
