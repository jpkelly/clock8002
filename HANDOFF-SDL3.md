# SDL3 Migration Handoff

**Branch:** `feature/sdl3-migration`  
**Started:** 2026-04-13  
**Base:** upstream clock-8001 commit `f0412525` (Feature: Add SDL3 version of the clock, 2026-04-08)

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

### ⬜ Phase 3 — Port clock8002 additions
Port these from `master` onto `feature/sdl3-migration` as discrete commits:

**Additive (new files/dirs, no upstream conflict):**
- `buildroot-external/` — entire directory (minus SDL2-specific parts of clock8002.mk)
- `v4/alsa-ltc.c`, `v4/alsa-ltc.service`
- `v4/install.sh`
- `v4/clock.ini.gerry`, `v4/clock_g.ini`, `v4/network.ini.gerry`
- `v4/piclock-network.sh`, `v4/piclock-network.service`
- `v4/clock8002.service` (update binary: `sdl-clock` → `sdl3-clock`)
- `v4/oled/` daemon
- `tools/`, `pi345build/`
- `HANDOFF.md`, `RELEASING.md`

**Makefile additions:**
- `release` target, gerry variant, alsa-ltc build
- Update `sdl-clock` references → `sdl3-clock`

**clock8002.mk changes (Phase 3b):**

Current (complex, CGO):
```makefile
CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
  CGO_CFLAGS="-I$(STAGING)/usr/include/SDL2" \
  CGO_LDFLAGS="-L$(STAGING)/usr/lib -lSDL2" \
  GOOS=linux GOARCH=arm64 go build ./cmd/sdl-clock
```

New (trivial, no CGO):
```makefile
GOOS=linux GOARCH=arm64 go build ./cmd/sdl3-clock
```

**defconfig changes (Phase 3c):**
- Remove: `BR2_PACKAGE_SDL2`, Mesa 25.0.7 manual patch
- Add: `BR2_PACKAGE_SDL3=y`, `BR2_PACKAGE_SDL3_TTF=y`, `BR2_PACKAGE_SDL3_IMAGE=y`

### ⬜ Phase 4 — Hardware validation on piclockBR
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

---

## Build Workflow (unchanged from master)

```bash
# On Mac — edit, commit, push
git push origin feature/sdl3-migration

# On cm5
cd ~/clock8002 && git pull --ff-only
cd ~/buildroot && make clock8002-dirclean && make > /tmp/br-build.log 2>&1

# Transfer image
scp pi@cm5.local:~/buildroot/output/images/sdcard.img \
    ~/Desktop/piclockBR-<HASH>-sdcard.img
```
