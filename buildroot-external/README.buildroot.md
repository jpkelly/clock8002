# clock8002 Buildroot Image — Developer Reference

This document covers building, flashing, and maintaining the Buildroot-based SD card image for clock8002 on Raspberry Pi 5. For user-facing setup and configuration, see the [main README](../README.md).

Current branch baseline (2026-05-24):

- Active dev/RC branch: `feature/root-ram`
- Active build tree on cm5: `~/clock8002-root-ram`
- Primary validation target: `.245` (with `.246` as reference/research only)

When this branch is promoted, replace branch-specific wording with `master` equivalents.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Build Host Setup](#build-host-setup)
  - [Manual Build-Host Patches (not in git)](#manual-build-host-patches-not-in-git)
    - [1. Mesa 25.0.7 upgrade](#1-mesa-2507-upgrade)
    - [2. host-xz libtool workaround](#2-host-xz-libtool-workaround)
- [Building an Image](#building-an-image)
  - [Payload Kernel Policy (Mandatory)](#payload-kernel-policy-mandatory)
  - [Full image build](#full-image-build)
  - [Partial rebuilds](#partial-rebuilds-when-to-use-each)
  - [Release build (password only, no SSH key)](#release-build-password-only-no-ssh-key)
  - [Dev build (SSH key + password)](#dev-build-ssh-key--password)
  - [Defconfig notes](#defconfig-notes)
- [Flashing an Image](#flashing-an-image)
  - [Copy image to Mac](#copy-image-to-mac)
  - [Flash to SD card](#flash-to-sd-card-run-manually--never-automated)
- [EEPROM Provisioning (Pi 5)](#eeprom-provisioning-pi-5)
- [SSH Access](#ssh-access)
- [Deploying a Binary (Without Reflash)](#deploying-a-binary-without-reflash)
- [BusyBox Compatibility Notes](#busybox-compatibility-notes)
- [Service Management](#service-management)
- [Config Files](#config-files)
- [Known Issues / Open Work](#known-issues--open-work)
- [Test Units](#test-units)

## Overview

The Buildroot image produces a minimal, deterministic SD card image. Key properties:

- Minimal init — much faster boot than Trixie
- Low RSS baseline (~44 MB for sdl-clock vs ~497 MB on Trixie)
- No swap partition — OOM kills are abrupt, not gradual
- No `pi` user — runs as `root`
- No `apt` — updates require a full image rebuild and reflash
- GLES2/KMSDRM renderer (no X11)
- Mesa 25.0.7 (see [Manual Build-Host Patches](#manual-build-host-patches) below)

## Architecture

```
buildroot-external/
  external.desc / external.mk / Config.in
  package/clock8002/
    clock8002.mk          — builds sdl-clock + alsa-ltc, installs services
  board/clock8002-rpi5/
    post-build.sh         — rootfs config, SSH, service masks
    post-image.sh         — genimage + mtools boot partition injection
    config.txt            — Pi 5 D0 DTB, overlays, KMSDRM
    cmdline.txt           — quiet boot flags
    linux.config          — kernel fragment
    genimage.cfg.in       — partition layout
    rootfs-overlay/       — files copied verbatim into rootfs
  configs/
    clock8002_rpi5_defconfig.sample  — pinned Buildroot 2025.02 defconfig
```

`sdl-clock` and `alsa-ltc` are built directly from `~/clock8002-root-ram/v4` on the build host — not from a tarball download.

## Build Host Setup

**Build host:** `pi@cm5.local` (CM5, 8GB RAM, NVMe)  
**Buildroot version:** 2025.02  
**Buildroot path:** `~/buildroot`  
**External tree:** `~/clock8002-root-ram/buildroot-external` (via `BR2_EXTERNAL`)

### Manual Build-Host Patches

Two changes must be applied to `~/buildroot/` after any clean Buildroot checkout. Run the script from this repo:

```bash
buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot
```

The script is idempotent — safe to run multiple times. See issue #29 for background.

#### 1. Mesa 25.0.7 upgrade

Mesa 24.0.9 (Buildroot 2025.02 default) has a V3D GLES2 alpha blending bug that makes text rendering invisible on Pi 5. Mesa 25.0.7 fixes it.

The script applies these changes to `~/buildroot/package/mesa3d/`:

- `mesa3d.mk`: `version = 24.0.9` → `25.0.7`
- `mesa3d.hash`: sha256/sha512 updated for new tarball
- `mesa3d.mk`: adds `host-python-pyyaml` dependency (required by Mesa 25.0.7)
- `mesa3d.mk`: removes deprecated meson options: `gallium-omx`, `dri3`, `gallium-vc4-neon`
- `patches/`: removes 4 incompatible patches (OpenCL/uClibc/ARM32 NEON — not needed for glibc aarch64)

**Failure mode if missing:** image builds successfully but text rendering is invisible on display.

#### 2. host-xz libtool workaround

host-xz 5.6.4 has a libtool bug where `acl_cv_wl` is set to `""` instead of `"-Wl,"`.

The script adds `acl_cv_wl="-Wl,"` to the configure env in `~/buildroot/package/xz/xz.mk`:

```makefile
CXX="$(HOSTCXX_NOCCACHE)" acl_cv_wl="-Wl,"
```

**Failure mode if missing:** build fails during host-xz compilation.

## Building an Image

Before any build, always sync the clock8002 source on the build host:

```bash
ssh pi@cm5.local 'cd ~/clock8002-root-ram && git pull --ff-only'
```

### Kernel Build Policy

**Default:** Use prebuilt kernel payload mode (`CLOCK8002_PREBUILT_KERNEL=1`).

**Kernel-from-source is opt-in:** Only use `CLOCK8002_PREBUILT_KERNEL=0` when intentionally validating custom kernel changes.

Prebuilt payload requirements (when opted in):

1. `Image`
2. DTBs (`dtbs/` or `dtb/`) plus `overlays/`
3. Modules (`modules/` or `modules/lib/modules/`)

`buildroot-external/scripts/build-with-kernel-fallback.sh` is a payload-only wrapper (legacy name retained). It runs one payload-mode build and fails closed on invalid bundles.

Payload build command (default `current` bundle):

```bash
ssh pi@cm5.local 'cd ~/buildroot && ~/clock8002-root-ram/buildroot-external/scripts/build-with-kernel-fallback.sh ~/buildroot'
```

Payload build command (explicit bundle override):

```bash
ssh pi@cm5.local 'cd ~/buildroot && ~/clock8002-root-ram/buildroot-external/scripts/build-with-kernel-fallback.sh ~/buildroot /path/to/prebuilt-kernel-bundle'
```

Wrapper behavior:

1. Verifies the bundle before build (layout + required assets + checksum when available).
2. Runs a single payload-mode build with `CLOCK8002_PREBUILT_KERNEL=1`.
3. Fails closed on missing/invalid bundle.

Expected prebuilt bundle layout:

```text
<bundle>/
  Image
  dtbs/                    # or dtb/
    *.dtb
  overlays/
    *.dtbo
    overlay_map.dtb
    bcm2712d0.dtbo
  modules/                 # either modules/<release>/... or modules/lib/modules/<release>/...
    ...
  SHA256SUMS               # optional but recommended; verified when present
```

Canonical prebuilt bundle storage (cm5):

```text
/srv/clock8002/prebuilt-kernel-bundles/
  bundle-<id>/
    Image
    dtbs/ or dtb/
    overlays/
    modules/
    SHA256SUMS
    BUNDLE.PROVENANCE
  current -> bundle-<id>
```

Bundle management commands:

```bash
# Verify a candidate bundle before promotion.
buildroot-external/scripts/verify-prebuilt-kernel-bundle.sh /path/to/prebuilt-kernel-bundle

# Promote candidate into canonical store and repoint current.
sudo buildroot-external/scripts/promote-prebuilt-kernel-bundle.sh \
  /path/to/prebuilt-kernel-bundle \
  bundle-<id> \
  /srv/clock8002/prebuilt-kernel-bundles

# Verify active bundle in canonical location.
buildroot-external/scripts/verify-prebuilt-kernel-bundle.sh \
  /srv/clock8002/prebuilt-kernel-bundles/current
```

Operational requirements when using prebuilt payload:

1. Keep the active validation bundle promoted under `/srv/clock8002/prebuilt-kernel-bundles/current`.
2. Record bundle provenance in HANDOFF before flash/test.

### Full image build

```bash
ssh pi@cm5.local 'cd ~/buildroot && ~/clock8002-root-ram/buildroot-external/scripts/build-with-kernel-fallback.sh ~/buildroot'
```

Monitor progress:

```bash
ssh pi@cm5.local 'L=$(ls -1t /tmp/br-payload-*.log 2>/dev/null | head -n1); [ -n "$L" ] && tail -f "$L" || echo "No payload build log yet"'
```

Quoting-safe status probe (recommended for ad-hoc checks):

```bash
tools/buildroot/cm5-build-status.sh --session <session-name>
```

This helper avoids fragile inline SSH quoting and reports state, elapsed time, active make process, last package stamp, strict kernel-compile markers, and a log tail.

### Partial rebuilds (when to use each)

| Change type | Command |
|---|---|
| sdl-clock / alsa-ltc code | `CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make clock8002-dirclean && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make` |
| Kernel config (`linux.config`) | Not used for payload-validated images (kernel comes from prebuilt payload bundle). |
| Defconfig changes | `make clock8002_rpi5_defconfig` then payload wrapper command |
| `rpi-firmware` / `config.txt` | `CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make rpi-firmware-dirclean && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make` |
| Rootfs overlay only | `CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make` |

### Release build (password only, no SSH key)

Release builds must use `make clean` — not `clock8002-dirclean` — to wipe `output/target/` completely. `clock8002-dirclean` only cleans the package build dir; stale files (including SSH keys from prior dev builds) persist in `output/target/` across partial rebuilds. `post-build.sh` also actively purges any stale key, but `make clean` ensures a provably clean rootfs.

```bash
ssh pi@cm5.local 'cd ~/clock8002-root-ram && git fetch --tags origin && git checkout vX.X.X && cd ~/buildroot && make clean && ~/clock8002-root-ram/buildroot-external/scripts/build-with-kernel-fallback.sh ~/buildroot'
```

> Replace `vX.X.X` with the release tag. `make clean` is slower (full rebuild of all packages) but required for release integrity.

### Dev build (SSH key + password)

Bakes your personal SSH public key into the image for passwordless login. `BR2_PICLOCKKEY` must be set at build time.

**One-time setup on cm5** — store your key in a local untracked file:

```bash
ssh pi@cm5.local 'echo "export BR2_PICLOCKKEY=\"ssh-ed25519 AAAA... you@host\"" > ~/buildroot-keys.env && chmod 600 ~/buildroot-keys.env'
```

**Dev build command:**

```bash
ssh pi@cm5.local 'source ~/buildroot-keys.env && cd ~/clock8002-root-ram && git pull --ff-only && cd ~/buildroot && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current ~/clock8002-root-ram/buildroot-external/scripts/build-with-kernel-fallback.sh ~/buildroot'
```

> Dev builds can use `clock8002-dirclean` for speed. The SSH key is re-injected on every build by `post-build.sh` when `BR2_PICLOCKKEY` is set.

### Defconfig notes

The committed file `configs/clock8002_rpi5_defconfig.sample` is **not** the live `.config` on cm5. After adding packages to the sample you must also patch the live `.config`:

```bash
# Example: adding a package
sed -i 's/# BR2_PACKAGE_FOO is not set/BR2_PACKAGE_FOO=y/' ~/buildroot/.config
make olddefconfig
```

Always verify: `grep BR2_PACKAGE_<name> ~/buildroot/.config`

## Flashing an Image

### Copy image to Mac

```bash
scp pi@cm5.local:~/buildroot/output/images/sdcard.img \
    /Users/yourname/Desktop/piClock-<7-char-commit>-sdcard.img
```

Image naming convention: `piClock-<7-char-commit-hash>-sdcard.img`

### Flash to SD card (run manually — never automated)

First identify the correct disk:

```bash
diskutil list external physical
```

Then flash (replace `diskN` with the actual disk number):

```bash
diskutil unmountDisk /dev/diskN
sudo dd if=/Users/jp/Desktop/piClock-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress
diskutil eject /dev/diskN
```

## EEPROM Provisioning (Pi 5)

New Pi 5 units (or units previously running Trixie) need a one-time EEPROM config change to set `BOOT_ORDER=0xf1` (SD-only boot) and suppress the red PCIe probe screen.

The boot-partition `pieeprom.upd`/`recovery.bin` approach was removed in `65af133` — it caused a red screen loop on units with newer firmware (Jan 2026+). Use the manual approach instead:

```bash
# SSH into the running unit
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piClock.local

# Write config, build patched firmware blob, apply
printf '[all]\nBOOT_UART=1\nBOOT_ORDER=0xf1\n' > /tmp/eeprom.cfg
BLOB=$(ls /usr/lib/firmware/raspberrypi/bootloader-2712/default/pieeprom-*.bin | sort | tail -1)
rpi-eeprom-config --config /tmp/eeprom.cfg --out /tmp/custom.bin "$BLOB"
rpi-eeprom-update -d -f /tmp/custom.bin
reboot
```

This is a one-time operation per unit. After reboot, boot is clean with no red screen.

To verify the current EEPROM settings at any time:

```bash
rpi-eeprom-config
```

Expected output should include `BOOT_ORDER=0xf1`.

## SSH Access

Default credentials on all images:

| | Value |
|---|---|
| User | `root` |
| Password | `clockworkadmin` |
| Root login | Enabled (`PermitRootLogin yes`) |

SSH with key (dev builds with `BR2_PICLOCKKEY` set):

```bash
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piClock.local
```

SSH with password (release images):

```bash
ssh root@piClock.local
```

SCP note: use `-o IdentitiesOnly=yes -i ~/.ssh/id_ed25519` to avoid "too many authentication failures" if SSH agent has multiple keys loaded.

## Deploying a Binary (Without Reflash)

To test a single binary change on the running unit without rebuilding the full image:

```bash
# Cross-compile on Mac (from v4/)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOFLAGS=-mod=vendor go build -o /tmp/sdl3-clock-linux-arm64 ./cmd/sdl3-clock

# Stop service on unit, copy binary, then restart
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piClock.local '/etc/init.d/S99clock stop'
scp -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 /tmp/sdl3-clock-linux-arm64 root@piClock.local:/opt/clock8002/sdl3-clock
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piClock.local '/etc/init.d/S99clock start'
```

No `install.sh` exists on Buildroot — deploy binaries directly.

## BusyBox Compatibility Notes

The Buildroot target uses BusyBox — not GNU coreutils. Key differences:

| Operation | GNU (Trixie) | BusyBox (Buildroot) |
|---|---|---|
| Shell | `bash` | `sh` (no bash) |
| tar decompress | `tar -xzf` | `gzip -d -c foo.tar.gz \| tar x` (no `-z`) |
| sha256sum | `--ignore-missing` | not supported |
| hostname | `hostname -I` | `ip -4 -o addr show scope global` |
| ssh/scp user | `pi@` | `root@` |

## Service Management

```sh
# Clock
/etc/init.d/S99clock start
/etc/init.d/S99clock stop
/etc/init.d/S99clock restart
ps | grep sdl3-clock        # status

# LTC decoder
/etc/init.d/S99alsa-ltc start
/etc/init.d/S99alsa-ltc stop
/etc/init.d/S99alsa-ltc restart
ps | grep alsa-ltc          # status

# OLED daemon
/etc/init.d/S98oled start
/etc/init.d/S98oled stop
ps | grep oled_daemon       # status
```

Log file location (root user): `/root/.config/clock-8001/clock.log`

## Config Files

Config files live on the boot partition under `/boot/piclock/`:

| File | Purpose |
|---|---|
| `/boot/piclock/clock.ini` | Main clock config |
| `/boot/piclock/network.ini` | Network / Wi-Fi config |
| `/boot/piclock/oled.ini` | OLED hardware config — enable/disable, I2C port, I2C address, rotation |
| `/boot/piclock/authorized_keys` | SSH public keys for passwordless root login (optional) |

`/opt/clock8002/clock.ini` and `/opt/clock8002/oled/oled.ini` are symlinks into `/boot/piclock/`, so edits survive service restarts.

At boot, `S03copy_clock_files` copies `/boot/piclock/authorized_keys` to `/root/.ssh/authorized_keys` (mode 600) if it exists. To enable passwordless SSH, place your public key(s) in that file on the FAT boot partition before first boot — no SD card re-flash required.

### SSH key provisioning — dev vs production

| Build type | How to build | SSH access |
|---|---|---|
| **Production** | `make` (no `BR2_PICLOCKKEY`) | Password only (`clockworkadmin`) — no key baked in |
| **Dev** | `BR2_PICLOCKKEY='ssh-ed25519 ...' make` | Key baked into image at build time |
| **Field provisioning** | Either build type | Drop `authorized_keys` on `/boot/piclock/` — applied at every boot |

Production images ship with **no embedded SSH key**. Field SSH key provisioning via `/boot/piclock/authorized_keys` works on both build types without reflashing.

## Known Issues / Open Work

| Issue | Description |
|---|---|
| [#28](https://github.com/jpkelly/clock8002/issues/28) | Post-merge validation: Trixie regression test, broadcast.go leak fix, boot splash, Wi-Fi AP test |
| [#29](https://github.com/jpkelly/clock8002/issues/29) | Mesa 25.0.7 + host-xz manual patches — not in git, must be re-applied after clean checkout |
| [#30](https://github.com/jpkelly/clock8002/issues/30) | SSH: hardcoded dev key replaced with `BR2_PICLOCKKEY` env var; default root password set |

## Test Units

| Hostname | IP | RAM | Notes |
|---|---|---|---|
| `piClock.local` | 10.0.0.184 | 2GB (actually 8GB — Rev 1.1 D0) | Primary dev/test unit |
| `piclockT.local` | 10.0.0.162 | 1GB | Stability test unit |
