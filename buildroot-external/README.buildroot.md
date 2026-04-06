# clock8002 Buildroot Image

This document covers building, flashing, and maintaining the Buildroot-based SD card image for clock8002 on Raspberry Pi 5. This is separate from the Debian/Trixie install.sh deployment path described in the main README.

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

`sdl-clock` and `alsa-ltc` are built directly from `~/clock8002/v4` on the build host — not from a tarball download.

## Build Host Setup

**Build host:** `pi@pi5start.local`  
**Buildroot version:** 2025.02  
**Buildroot path:** `~/buildroot`  
**External tree:** `~/clock8002/buildroot-external` (via `BR2_EXTERNAL`)

### Manual Build-Host Patches (not in git)

Two changes must be applied manually to `~/buildroot/` after any clean Buildroot checkout. These are tracked in issue #29.

#### 1. Mesa 25.0.7 upgrade

Mesa 24.0.9 (Buildroot 2025.02 default) has a V3D GLES2 alpha blending bug that makes text rendering invisible on Pi 5. Mesa 25.0.7 fixes it.

Changes to apply in `~/buildroot/package/mesa3d/`:

- `mesa3d.mk`: `version = 24.0.9` → `25.0.7`
- `mesa3d.hash`: update sha256/sha512 for new tarball
- `mesa3d.mk`: add `host-python-pyyaml` to dependencies
- `mesa3d.mk`: remove deprecated meson options: `gallium-omx`, `dri3`, `gallium-vc4-neon`
- `patches/`: move 4 incompatible patches to `patches-backup/` (OpenCL/uClibc/ARM32 NEON — not needed for glibc aarch64)

**Failure mode if missing:** image builds successfully but text rendering is invisible on display.

#### 2. host-xz libtool workaround

host-xz 5.6.4 has a libtool bug where `acl_cv_wl` is set to `""` instead of `"-Wl,"`.

In `~/buildroot/package/xz/xz.mk`, line 69, add `acl_cv_wl="-Wl,"` to the configure env:

```makefile
CXX="$(HOSTCXX_NOCCACHE)" acl_cv_wl="-Wl,"
```

**Failure mode if missing:** build fails during host-xz compilation.

---

> **Note:** A script to automate re-applying these patches is tracked in issue #29.

## Building an Image

Before any build, always sync the clock8002 source on the build host:

```bash
ssh pi@pi5start.local 'cd ~/clock8002 && git pull --ff-only'
```

### Full image build

```bash
ssh pi@pi5start.local 'cd ~/buildroot && make clock8002-dirclean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$?'
```

Monitor progress:

```bash
ssh pi@pi5start.local 'tail -f /tmp/br-build.log'
```

### Partial rebuilds (when to use each)

| Change type | Command |
|---|---|
| sdl-clock / alsa-ltc code | `make clock8002-dirclean && make` |
| Kernel config (`linux.config`) | `make linux-dirclean && make` |
| Defconfig changes | `make clock8002_rpi5_defconfig` then `make` |
| `rpi-firmware` / `config.txt` | `make rpi-firmware-dirclean && make` (or just `make` — post-image.sh force-syncs config.txt) |
| Rootfs overlay only | `make` |

### Release build (password only, no SSH key)

This is the standard release build. No SSH key is baked into the image — users log in with the default root password.

```bash
ssh pi@pi5start.local 'cd ~/clock8002 && git pull --ff-only && cd ~/buildroot && make clock8002-dirclean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$?'
```

### Dev build (SSH key + password)

Bakes your personal SSH public key into the image for passwordless login. `BR2_PICLOCKKEY` must be set at build time.

**One-time setup on pi5start** — store your key in a local untracked file:

```bash
ssh pi@pi5start.local 'echo "export BR2_PICLOCKKEY=\"ssh-ed25519 AAAA... you@host\"" > ~/buildroot-keys.env && chmod 600 ~/buildroot-keys.env'
```

**Dev build command:**

```bash
ssh pi@pi5start.local 'source ~/buildroot-keys.env && cd ~/clock8002 && git pull --ff-only && cd ~/buildroot && make clock8002-dirclean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$?'
```

The key is injected into `/root/.ssh/authorized_keys` in the image. The root password is also set, so both auth methods work on dev images.

### Defconfig notes

The committed file `configs/clock8002_rpi5_defconfig.sample` is **not** the live `.config` on pi5start. After adding packages to the sample you must also patch the live `.config`:

```bash
# Example: adding a package
sed -i 's/# BR2_PACKAGE_FOO is not set/BR2_PACKAGE_FOO=y/' ~/buildroot/.config
make olddefconfig
```

Always verify: `grep BR2_PACKAGE_<name> ~/buildroot/.config`

## Flashing an Image

### Copy image to Mac

```bash
scp pi@pi5start.local:~/buildroot/output/images/sdcard.img \
    ~/Desktop/piclockBR-<7-char-commit>-sdcard.img
```

Image naming convention: `piclockBR-<7-char-commit-hash>-sdcard.img`

### Flash to SD card (run manually — never automated)

First identify the correct disk:

```bash
diskutil list external physical
```

Then flash (replace `diskN` with the actual disk number):

```bash
diskutil unmountDisk /dev/diskN
sudo dd if=/Users/jp/Desktop/piclockBR-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress
diskutil eject /dev/diskN
```

## EEPROM Provisioning (Pi 5)

New Pi 5 units (or units previously running Trixie) need a one-time EEPROM config change to set `BOOT_ORDER=0xf1` (SD-only boot) and suppress the red PCIe probe screen.

The boot-partition `pieeprom.upd`/`recovery.bin` approach was removed in `65af133` — it caused a red screen loop on units with newer firmware (Jan 2026+). Use the manual approach instead:

```bash
# SSH into the running unit
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piclockBR.local

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
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piclockBR.local
```

SSH with password (release images):

```bash
ssh root@piclockBR.local
```

SCP note: use `-o IdentitiesOnly=yes -i ~/.ssh/id_ed25519` to avoid "too many authentication failures" if SSH agent has multiple keys loaded.

## Deploying a Binary (Without Reflash)

To test a single binary change on the running unit without rebuilding the full image:

```bash
# Build on pi5start, copy binary to unit, restart service
systemctl stop clock8002
scp <binary> root@piclockBR.local:/opt/clock8002/sdl-clock
ssh root@piclockBR.local 'systemctl start clock8002'
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

```bash
systemctl start|stop|restart|status clock8002
systemctl start|stop|restart|status alsa-ltc
systemctl start|stop|restart|status oled_daemon
journalctl -u clock8002 -f
```

Log file location (root user): `/root/.config/clock-8001/clock.log`

## Config Files

Config files live on the boot partition under `/boot/piclock/`:

| File | Purpose |
|---|---|
| `/boot/piclock/clock.ini` | Main clock config |
| `/boot/piclock/network.ini` | Network / Wi-Fi config |
| `/boot/piclock/oled.ini` | OLED hardware config — enable/disable, I2C port, I2C address, rotation |

`/opt/clock8002/clock.ini` and `/opt/clock8002/oled/oled.ini` are symlinks into `/boot/piclock/`, so edits survive service restarts.

## Known Issues / Open Work

| Issue | Description |
|---|---|
| [#28](https://github.com/jpkelly/clock8002/issues/28) | Post-merge validation: Trixie regression test, broadcast.go leak fix, boot splash, Wi-Fi AP test |
| [#29](https://github.com/jpkelly/clock8002/issues/29) | Mesa 25.0.7 + host-xz manual patches — not in git, must be re-applied after clean checkout |
| [#30](https://github.com/jpkelly/clock8002/issues/30) | SSH: hardcoded dev key replaced with `BR2_PICLOCKKEY` env var; default root password set |

## Test Units

| Hostname | IP | RAM | Notes |
|---|---|---|---|
| `piclockBR.local` | 10.0.0.184 | 2GB (actually 8GB — Rev 1.1 D0) | Primary dev/test unit |
| `piclockT.local` | 10.0.0.162 | 1GB | Stability test unit |
