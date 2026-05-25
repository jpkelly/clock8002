# Clock8002 Handoff

## Hard Rules (MUST follow)

1. **Prebuilt kernel is the default.** `CLOCK8002_PREBUILT_KERNEL=1` is automatic. To compile a custom kernel, you MUST explicitly set `CLOCK8002_PREBUILT_KERNEL=0`. There is no other mode.
2. **Never reuse an output directory across commits.** Each build gets a fresh `O=/home/pi/output-root-ram-<commit>-<timestamp>`.
3. **Verify branch and commit before building.** Run `git branch --show-current && git log --oneline -1` on cm5 before every build.
4. **Always build inside a screen session.** Never run long builds directly in an SSH one-liner.

## Build Workflow Docs (Use These First)

- Policy: `docs/build-policy.md`
- Manifest template: `docs/build-manifest-template.md`

### How to use
1. Read `docs/build-policy.md` before starting a candidate build.
2. Run the build using one source tree and one commit set only.
3. Fill `docs/build-manifest-template.md` for the build before flashing.
4. Only promote to known-good after hardware validation (boot, services, LTC).
5. If build inputs or outputs are not recorded in a manifest, treat the image as non-reproducible.

## CM5 Enforcement Gate

Use this before any build on cm5.

1. **One active working copy only**
  - Use `/home/pi/clock8002-root-ram` as the canonical dev/RC checkout.
  - Do not build from a second cm5 checkout path unless the workflow explicitly says so.

2. **Pinned inputs only**
  - Build from an explicit commit SHA or tag, not a moving branch tip.
  - Record the exact source SHA(s) and kernel bundle path in the manifest.

3. **Preflight must pass**
  - `git -C /home/pi/clock8002-root-ram branch --show-current`
  - `git -C /home/pi/clock8002-root-ram status --short`
  - `git -C /home/pi/clock8002-root-ram worktree list`
  - Fail if the working tree is dirty, the branch is wrong, or another worktree is attached to the target branch.

4. **Manifest required for any candidate image**
  - If the image might be flashed, tested, or promoted, create/update the manifest before sharing it.
  - Manifest must be stored with the image basename and referenced in HANDOFF.

5. **Reference unit rule**
  - `/home/pi/clock8002-root-ram` is the active build tree.
  - `root@192.168.8.245` is the primary validation target.
  - `root@192.168.8.246` is reference/research only.

## Commonly Missed Items

1. **Filename labels are not provenance**
  - Always trust manifest + hashes, not the commit string in an image filename.

2. **Mixed source trees are invalid**
  - Never combine app source from one checkout with buildroot-external from another.

3. **Release vs dev are different flows**
  - Dev/RC uses `feature/root-ram` and the canonical cm5 working copy.
  - Release uses a ref-verified clean clone.

4. **No branch switching during active build**
  - Wait for the exit marker before any checkout or ref movement.

5. **Every promoteable image needs a manifest**
  - No manifest means no candidate / no known-good.

## Root-Ram To Master Cutover Checklist (Concise)

Use this only after feature/root-ram is functionally complete.

1. **Freeze feature work**
  - Stop adding new features on `feature/root-ram`.
  - Fix only blockers/regressions.

2. **Pick release-candidate commit**
  - Select one commit SHA on `feature/root-ram` as RC.
  - Build from that SHA using fresh output dir + manifest.

3. **Run final hardware validation on .245**
  - Boot success.
  - `clock8002`, `alsa-ltc`, `oled-daemon` running.
  - LTC decode confirmed.
  - Required network/serial features confirmed.

4. **Record known-good evidence**
  - Complete manifest with image + runtime hashes.
  - Add short validation result block to HANDOFF.

5. **Prepare rollback point on master**
  - Tag current `master` before promotion.
  - Keep rollback tag documented in HANDOFF.

6. **Promote to master**
  - Preferred: fast-forward `master` to RC commit.
  - If needed: normal merge commit (no history rewrite).

7. **Retag and document release baseline**
  - Tag promoted commit as new baseline/release tag.
  - Update HANDOFF latest tag + commit + status.

8. **Switch default workflow to master**
  - Update instructions/docs so normal builds use `master`.
  - Keep `feature/root-ram` only for hotfix overlap, then retire.

## Current State (2026-05-25 — SSH key persistence fix + clean rebuild in progress)

### Active dev baseline
- **Branch**: `feature/root-ram`
- **HEAD**: `215b93a` — pushed to GitHub (`fix: unbind fbcon from fb0 before splash so console text cannot overwrite it`)
- **Last tag**: `working-2026-05-24-ltc-broadcast` at `14a485b`
- **Kernel bundle**: `bundle-245-6.12.41-v8-20260509-161234` (prebuilt, `CLOCK8002_PREBUILT_KERNEL=1`)

### Validated on .245 (2026-05-24)
- Wi-Fi AP (hostapd 2.11) — `piClock-ap` SSID broadcasts ✅
- DHCP via dnsmasq (192.168.50.10-100/24) ✅
- OLED Wi-Fi icon (`is_ap_active` via `/boot/iw`) ✅
- SDL overlay "AP SSID: piClock-ap" visible in info panel ✅
- LTC broadcast via alsa-ltc confirmed working ✅ (fixed in `14a485b`)
- SSH, boot, power button all working ✅
- vtcon1 unbind + splash repaint in `clock_pokemon.sh` — live on device ✅ (from `215b93a`)
- `--info-timer 0` suppressing startup overlay when splash enabled — live on device ✅ (from `f0cd63d`)

### Working tree — uncommitted changes (pending rebuild)
These changes are in the repo source but NOT yet baked into any flashed image.
They require a Buildroot rebuild to take effect — runtime root is fresh tmpfs each boot and `/etc/init.d/` changes on-device are lost on reboot.
Note: `buildroot-external/board/clock8002-rpi5/config.txt` has `initramfs rootfs.cpio.gz followkernel` **active**, so rootfs-overlay changes DO reach the running system after a rebuild.

| File | Change |
|------|--------|
| `buildroot-external/board/clock8002-rpi5/cmdline.txt` | Added `quiet loglevel=0` |
| `golden-working-card/boot/cmdline.txt` | Added `quiet loglevel=0` |
| `rootfs-overlay/etc/init.d/S03copy_clock_files` | Unbinds vtcon1 + paints splash right after /boot mount |
| `golden-working-card/etc/init.d/S03copy_clock_files` | Same — golden copy |
| `golden-working-card/piclock/setup.sh` | Unbinds vtcon1 before dd write of bootsplash.raw |
| `rootfs-overlay/etc/init.d/S00splash` | **NEW** — unbinds vtcon1 at S00 time (very early, before any init text) |
| `golden-working-card/etc/init.d/S00splash` | **NEW** — same, golden copy |

### Firmware splash research result (2026-05-24)
**RPi5 EEPROM bootloader has NO custom splash PNG support.** There is no `splash.png` or equivalent mechanism in the firmware. `disable_splash=1` in `config.txt` suppresses the rainbow only. The current dd-to-fb0 approach is the correct userspace solution for this platform.

### Bootsplash architecture (how it works)
- `bootsplash.raw` (RGB565, 1920×1080) generated at build time via ffmpeg, staged to `/boot/piclock/` on FAT
- `splash_enabled=true` in `/boot/piclock/piclock.ini` enables the feature
- **Currently**: vtcon1 unbind + dd write happens in `setup.sh` (late — at S99clock time via clock_pokemon.sh)
- **With pending changes**: vtcon1 unbind at S00 time (very early) → splash painted at S03 time (as soon as /boot mounts)
- **Alternative not yet implemented**: `console=ttyAMA0` in `cmdline.txt` would redirect all console output to UART, leaving the display completely clear until the clock app starts — cleanest option

### Open items
1. **Bootsplash text suppression** — uncommitted changes above need to be committed and built into an image.
   - Option A (S00splash + S03 approach): commit the working tree, rebuild, flash
   - Option B (console redirect): change `console=tty1` → `console=ttyAMA0,115200` in `cmdline.txt` — simpler, no init text at all on display
   - Both options require a rebuild; no further device-side testing can validate them until then.
2. **Manifest** — commits since `51703f6` need a build manifest.
3. **network.ini default** — `gateway=192.168.8.1` set live on device; golden-working-card default still has it commented out.

### Commits since last tag (working-2026-05-24-ltc-broadcast / 14a485b)
- `a21212f` — docs: clean up README for master promotion
- `f0cd63d` — fix: suppress boot text and startup overlay when bootsplash is enabled
- `215b93a` — fix: unbind fbcon from fb0 before splash so console text cannot overwrite it (HEAD)

## Current Checkpoint (2026-05-24 late - rollback baseline confirmed)

### TL;DR
- Repro image from `output-repro-245-20260524-062711` booted but did not match expected runtime behavior (user-observed LTC regression and boot visual mismatch).
- Trusted rollback card on `.245` and reference unit `.246` were used as baseline truth.
- Baseline runtime hashes (working) are:
  - `sdl-clock`: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
  - `alsa-ltc`: `c78c3fc8094dd701a9f63465641525998812db9c56be68f703173178eb830417`
- Baseline LTC launcher command (working):
  - `/root/alsa-ltc plughw:2,0 255.255.255.255 1245`
- Baseline boot profile from rollback/reference:
  - `cmdline.txt`: `logo.nologo consoleblank=0 console=tty1`
  - golden `config.txt` uses commented `#initramfs rootfs.cpio.gz` form (no active external initramfs line).

### Repo correction committed
- Commit `004a1fc` (branch `feature/root-ram`, pushed to origin):
  - `buildroot-external/board/clock8002-rpi5/golden-working-card/boot/config.txt`
  - aligned to rollback baseline by restoring commented initramfs line.

### Next build target
1. Build a fresh image from `004a1fc` with standard payload mode (`CLOCK8002_PREBUILT_KERNEL=1`) and fresh output dir.
2. Flash and validate on `.245` first.
3. Validate parity checklist:
  - single `sdl-clock` + single `alsa-ltc` process set
  - LTC decode working end-to-end
  - runtime hashes match expected baseline or known intended delta.

## Known-Good State (2026-05-23)

### Verified working commit
- **Commit:** `51703f6`
- **Tag:** `working-2026-05-23-powerbutton-ltc`
- **Branch:** `feature/root-ram`
- **Status:** Clean boot, LTC decoding, OLED, SSH, power button — all confirmed working on live unit

### Build from this tag (copy exactly)
```bash
ssh pi@cm5.local 'sh' <<'REMOTE'
cd ~/clock8002 && git checkout working-2026-05-23-powerbutton-ltc && git log --oneline -1
cd ~/buildroot
OUTPUT="output-root-ram-51703f6-$(date +%Y%m%d-%H%M%S)"
screen -S br-build-51703f6 -dm bash -lc "{
  CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/${OUTPUT} BR2_EXTERNAL=/home/pi/clock8002/buildroot-external clock8002_rpi5_defconfig &&
  CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/${OUTPUT} BR2_EXTERNAL=/home/pi/clock8002/buildroot-external clock8002-dirclean &&
  CLOCK8002_PREBUILT_KERNEL=1 BR2_PICLOCKKEY='$(cat ~/.ssh/id_rsa.pub)' make O=/home/pi/${OUTPUT} BR2_EXTERNAL=/home/pi/clock8002/buildroot-external;
} > /tmp/br-build-51703f6.log 2>&1; echo BR_BUILD_EXIT:\$? > /tmp/br-build-51703f6.exit"
echo "Build started: ${OUTPUT}"
REMOTE
```

### Why this specific command
- Source dir `~/clock8002` has the defconfig pointing to `/home/pi/clock8002/v4` — this is what the working image was built from
- `clock8002-dirclean` forces rebuild of Go binaries and oled-daemon
- `CLOCK8002_PREBUILT_KERNEL=1` uses the approved kernel bundle (`bundle-245-6.12.41-v8-20260509-161234`)
- Fresh output dir prevents stale-cache corruption

### Monitor
```bash
ssh pi@cm5.local 'screen -ls; tail -f /tmp/br-build-51703f6.log'
```

### Check completion
```bash
ssh pi@cm5.local 'cat /tmp/br-build-51703f6.exit'
```

### Important notes
- **Do NOT build from `ba7209b`** — it is broken (prebuilt kernel default change, untested)
- **Do NOT use `~/clock8002-root-ram`** for Buildroot source dir — the defconfig still points to `~/clock8002/v4`
- **Do NOT delete `output-root-ram-51703f6-retry-20260523-155431`** — this is the reference working build dir

## Current Checkpoint (2026-05-23 — commit 62266c9, build with CLOCK8002_PREBUILT_KERNEL=0)

### TL;DR
- **Commit 62266c9** (`CONFIG_SND=y` + `CONFIG_SND_USB_AUDIO=y`): fragment fix committed and pushed.
- **First 62266c9 build FAILED silently**: `post-image.sh` replaced our built kernel with golden prebuilt `3062308d`. Image flashed to .245 was golden kernel, not custom.
- **Root cause**: `CLOCK8002_PREBUILT_KERNEL` defaults to `1` (enabled). Build must use `CLOCK8002_PREBUILT_KERNEL=0`.
- **Corrected rebuild**: `br-build-62266c9-real` running now with `CLOCK8002_PREBUILT_KERNEL=0`.
- **Post-image fix** (`41a18fd`): When `CLOCK8002_PREBUILT_KERNEL=0`, runtime binaries are copied from fresh build output instead of stale golden copies.
- `.245` currently running golden kernel (from accidental flash), LTC working.

### Build history
| Commit | Config fix | Prebuilt flag | Result | Image hash |
|--------|-----------|---------------|--------|------------|
| c942a36 | SND_USB_AUDIO=y only | default (1) | Kernel =m (syncconfig reverted) | 542d2a58... |
| 62266c9 | SND=y + SND_USB_AUDIO=y | default (1) | Kernel compiled correctly, but post-image.sh swapped in golden prebuilt | 3062308d... (golden) |
| 62266c9-real | SND=y + SND_USB_AUDIO=y | **0** | Build in progress | TBD |

### cm5 build status
- **Screen**: `br-build-62266c9-real`
- **Log**: `/tmp/br-build-62266c9-real.log`
- **Exit marker**: `/tmp/br-build-62266c9-real.exit`
- **Command**: `linux-dirclean && CLOCK8002_PREBUILT_KERNEL=0 make`
- **Stage**: Kernel compile (lib/, fs/, drivers/ — ~95% done)
- **ETA**: ~5 min

Monitor:
```
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa pi@cm5.local 'screen -ls; tail -f /tmp/br-build-62266c9-real.log'
```
Check done:
```
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa pi@cm5.local 'cat /tmp/br-build-62266c9-real.exit'
```

### Post-build verification
1. `sha256sum images/Image` → must NOT be `3062308d...` (golden hash)
2. `grep "CONFIG_SND=" build/linux-custom/.config` → `=y`
3. `grep "CONFIG_SND_USB_AUDIO=" build/linux-custom/.config` → `=y`
4. Flash .245, verify LTC decodes with custom kernel

### Key findings this session
- `bcm2712_defconfig` sets `CONFIG_SND=m`; fragment override must include `CONFIG_SND=y`
- `post-image.sh` has `CLOCK8002_PREBUILT_KERNEL` guard — defaults to 1 (prebuilt injection)
- Golden prebuilt bundle (`/srv/clock8002/prebuilt-kernel-bundles/current/`) has **zero `.ko` files** — fully monolithic
- Old runtime binaries (sdl-clock, alsa-ltc) from golden card overwrite freshly built ones in `post-image.sh`
- Fix `41a18fd`: When `CLOCK8002_PREBUILT_KERNEL=0`, copy runtime binaries from `${BUILD_DIR}/clock8002-prototype/`
- `.245` is the correct test unit; `.246` is a separate unit with different config

## Previous Checkpoint (2026-05-22 - Issue #44 incremental queue prepared; cm5 build still running)

### TL;DR
- Unit at `192.168.8.245` is currently running build identity `74439b3` (binary hash-anchored) after flashing `piClock-fb6d5d4-sdcard.img`.
- User-observed behavior: `alsa-ltc` is running stably on the current boot.
- RAM-root headroom is strong on the live unit: ~1.6 GB `MemAvailable`, no swap, `/tmp` tmpfs at 4%.
- A separate full Buildroot image build is actively running on cm5 from commit `337aa9c` (session `br-root-ram-20260522-090437`).
- Local Issue #44 changes are queued in source (not yet built/flashed): power button wiring, machine-id path move, and network.ini backend-aligned dual-mode apply path.

### Live unit state (.245)
- Host: `sdl-clock` (`root@192.168.8.245`)
- Running/boot binary hashes:
  - `/root/sdl-clock`: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
  - `/boot/sdl-clock`: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
- Hash maps to source commit `74439b3` (previously documented in this handoff).
- `alsa-ltc` process observed: `/root/alsa-ltc plughw:2,0 255.255.255.255 1245`

### RAM-root memory headroom (.245)
- `MemTotal`: `2028180 kB`
- `MemAvailable`: `1677568 kB` (~1.6 GB available)
- Swap: not configured (`SwapTotal=0`)
- Top RSS sample:
  - `sdl-clock` ~52 MB
  - `oled-daemon` ~25 MB
  - `clock-bridge` ~6 MB
  - `alsa-ltc` ~2.7 MB
- tmpfs usage sample:
  - `/tmp`: `43.4M / 990.3M` (4%)
  - `/run`: near zero

### cm5 build status (in progress)
- Build host: `pi@cm5.local`
- Active session: `br-root-ram-20260522-090437`
- Build commit: `337aa9c` (`defconfig: add BR2_PACKAGE_HOST_DTC=y`)
- Clone path: `/tmp/clock8002-root-ram-build-20260522-090437`
- Output path: `/home/pi/output-root-ram-20260522-090437`
- Log: `/tmp/br-root-ram-20260522-090437.log`
- Exit marker: `/tmp/br-root-ram-20260522-090437.exit`
- Current observed stage: Linux kernel compile (`CC fs/proc/*`, `CC io_uring/*`, `CC crypto/asymmetric_keys/*`).

### Local queued source changes (not yet built/flashed)
- Issue #44: power button support in golden runtime path
  - Added `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S04power-button`
  - Added `buildroot-external/board/clock8002-rpi5/golden-working-card/root/power-button.sh`
  - Wired in `buildroot-external/board/clock8002-rpi5/post-build.sh`
- Issue #44: machine-id persistence path
  - `buildroot-external/board/clock8002-rpi5/rootfs-overlay/etc/init.d/S12machine-id`
  - Store moved from `/boot/piclock/machine-id` to `/boot/.piclock-machine-id`
- Issue #44: network + WiFi + ini path standardization
  - Removed legacy `/boot/interfaces` copy path and `ifup eth0:1` startup dependency
  - Added backend-aligned non-NM apply path for `network.ini` with explicit `dhcp`, `static`, and `dual` modes
  - Updated `v4/network.ini.default` comments for `dual` and `ap_country`
- Tracker file: `NEXT-BUILD-CHANGES.md`

### Notes
- The first launch attempt failed due to invalid target `clock8002-dirclean` in this invocation path.
- The second launch required defconfig bootstrap for the fresh output directory, then resumed normally.

## Previous Checkpoint (2026-05-14 - Option B: mdev blacklist + external initramfs)

### TL;DR
- **Option B implemented** in commit `744a7a6` — re-enables external initramfs while fixing mdev timing.
- Full `make clean && make` rebuild running on cm5 (screen `br-full-744a7a6`). Image not yet transferred.
- Goal: alsa-ltc works reliably AND all overlay/feature changes now take effect normally on each build.

### What changed (commit `744a7a6`)
- **New**: `rootfs-overlay/etc/modprobe.d/snd-usb-audio.conf`
  - `blacklist snd_usb_audio` — prevents mdev from auto-loading via `$MODALIAS` at ~1.5s
  - Explicit `modprobe snd_usb_audio` in `alsa-ltc_pokemon.sh` still works (blacklist only blocks alias-based loading)
- **Changed**: `config.txt` — `initramfs rootfs.cpio.gz followkernel` uncommented
  - Our Buildroot rootfs.cpio.gz now actually loads at boot (was commented out, blocking all overlay changes)

### Why this matters for feature development
- With `initramfs rootfs.cpio.gz followkernel` previously commented out, **no overlay changes ever reached the running system** — only FAT-resident files via `setup.sh` could change behaviour
- Option B restores the standard Buildroot workflow: edit overlay → `make` → flash → test
- Both prebuilt-kernel path (`CLOCK8002_PREBUILT_KERNEL=1`) and future compiled-kernel path inherit the fix automatically

### Build in progress
- Screen session: `br-full-744a7a6` on cm5
- Command: `make clean && make` (full rebuild, no shortcuts)
- Log: `/tmp/br-full-744a7a6.log`
- Exit file: `/tmp/br-full-744a7a6.exit`
- Started: ~17:04 UTC 2026-05-14; expect 30-60 min
- Monitor: `ssh pi@10.0.0.101 'tail -f /tmp/br-full-744a7a6.log'`
- Check done: `ssh pi@10.0.0.101 'cat /tmp/br-full-744a7a6.exit'`

### Post-build steps (once `BR_BUILD_EXIT:0`)
1. Transfer image: `scp pi@10.0.0.101:/home/pi/output-root-ram-payload-20260509-165344/images/sdcard.img /Users/jp/Desktop/piClock-744a7a6-sdcard.img`
2. Verify config.txt in image has `initramfs rootfs.cpio.gz followkernel` (not commented)
3. Flash to `/dev/disk6` (verify disk number first: `diskutil list external physical`)
4. Boot and confirm:
   - alsa-ltc running past 30s without bandwidth errors (`dmesg | grep -i "bandwidth\|alsa\|usb"`)
   - `ps | grep alsa` shows `/root/alsa-ltc plughw:2,0` running
   - Clock and OLED working as before

### Device / build state
- Live image: `piClock-2b9e641-sdcard.img` — still on device, unchanged
- HEAD: `744a7a6` on `feature/root-ram`
- cm5 worktree: `/home/pi/clock8002-root-ram` at `744a7a6`
- Output dir: `/home/pi/output-root-ram-payload-20260509-165344`

## Previous Checkpoint (2026-05-14 - alsa-ltc root cause determined)

### TL;DR
- Root cause of alsa-ltc reliability failure on Buildroot-built kernels: **identified**.
- Current image (`piClock-2b9e641`) continues to work — no code changes this session.
- Two concrete options documented to enable building the kernel ourselves.

### Root cause: mdev `$MODALIAS` timing in Buildroot 2025.11 rootfs

The prebuilt kernel (`bundle-245-6.12.41-v8-20260509-161234`, 73,177,600 bytes) has an **embedded initramfs** baked in, compiled by Buildroot 2021.11 on January 14, 2026. That embedded rootfs has an old `mdev.conf` that does **not** auto-load USB modules via `$MODALIAS`. Result: `snd_usb_audio` only loads at ~17s via the explicit `modprobe` in `alsa-ltc_pokemon.sh`, well after the USB host controller is settled.

Our Buildroot 2025.11 `mdev.conf` **does** have `$MODALIAS` auto-load rules. So:
- **With `initramfs rootfs.cpio.gz followkernel` re-enabled**: Pi firmware appends our fresh rootfs on top of the embedded one. mdev immediately loads `snd_usb_audio` at ~1.5s (USB device enumeration). This causes `Not enough bandwidth for altsetting 0` at ~30s when alsa-ltc opens the device. **This is why re-enabling external initramfs broke alsa-ltc.**
- **With a freshly-built Buildroot kernel (no embedded initramfs, external cpio)**: Same mdev timing problem — snd_usb_audio loads too early via `$MODALIAS`.

### Golden system facts (for reference)
- Host: `root@10.0.0.162`, hostname `sdl-clock`
- Kernel: `Linux sdl-clock 6.12.41-v8 #3 SMP PREEMPT Wed Jan 14 11:49:24 UTC 2026 aarch64`
- Compiler: `aarch64-linux-gcc.br_real (Buildroot 2021.11-18033-g83947c7bb6) 14.3.0`
- Image size: 73,177,600 bytes — identical to prebuilt bundle Image
- alsa-ltc running clean: `/root/alsa-ltc plughw:2,0 255.255.255.255 1245`
- ALSA cards: 0=vc4-hdmi-0, 1=vc4-hdmi-1, 2=USB Audio Device (C-Media 0d8c:0014)
- No `/proc/config.gz` on golden (no IKCONFIG_PROC)

### Paths to building the kernel ourselves

**Option A — Embed rootfs in new kernel (`CONFIG_INITRAMFS_SOURCE`)**
- Enable `BR2_LINUX_KERNEL=y` in defconfig; set `CONFIG_INITRAMFS_SOURCE` in `linux.config` to the Buildroot target dir.
- Buildroot bakes the rootfs into the kernel Image at build time.
- mdev sees the same old-style rootfs it does today — no timing regression.
- Requires full kernel build (slow; not yet tested).

**Option B — Fix mdev timing in external initramfs** _(lower risk, no kernel rebuild)_
- Add a custom mdev.conf fragment to `buildroot-external/board/clock8002-rpi5/rootfs-overlay/etc/mdev.conf` that **blocks** `$MODALIAS` auto-load for `snd_usb_audio` and related sound modules.
- Re-enable `initramfs rootfs.cpio.gz followkernel` in `config.txt`.
- `snd_usb_audio` then only loads via the explicit `modprobe` in `alsa-ltc_pokemon.sh` at the correct time.
- Works with both the prebuilt kernel and a freshly-built one.
- **Pending user approval before implementing.**

### Device / build state
- Live image: `piClock-2b9e641-sdcard.img` — unchanged, working
- HEAD: `2b9e641` on `feature/root-ram`
- No new commits this session

## Current Checkpoint (2026-05-13 - 2b9e641 confirmed; bootsplash working)

### TL;DR
- `piClock-2b9e641-sdcard.img` flashed 2026-05-10, confirmed working. ✅
- **Bootsplash**: CONFIRMED WORKING via raw RGB565 `dd` to `/dev/fb0` from `setup.sh`. ✅
- **Known issue**: init script console text overwrites splash before SDL3 clock takes over. Deferred — not blocking.
  - Best fix when ready: move `dd` write to just before clock loop in `clock_pokemon.sh` so it fires after init text settles.
  - Alternative: redirect console off display (`console=ttyAMA0` in `cmdline.txt`).
- **piclock.ini**: FAT config file at `/boot/piclock/piclock.ini`; toggle `splash_enabled=true|false`.
- Build: `bootsplash.raw` (RGB565, 1920×1080) generated at build time via `ffmpeg`, staged to `piclock/` on FAT.
- Root cause of previous splash attempts failing: embedded initramfs (not our Buildroot rootfs.cpio) boots — `S05bootsplash` and `/opt/clock8002/` from cpio are unreachable.
- No code changes since 2026-05-10. Holding on splash cosmetic fix.

### Commit chain (HEAD → oldest)
- `2b9e641` — bootsplash: implement via raw fb0 dd from setup.sh (**HEAD**)
- `53085e6` — piclock.ini: default splash_enabled=true
- `0f5ce12` — piclock.ini: add splash_enabled toggle; S05bootsplash reads it at boot
- `cf66b9d` — HANDOFF.md update
- `3f1ce4c` — copilot-instructions: document clock8002-dirclean requirement

### Live device state
- Image: `piClock-2b9e641-sdcard.img` (flashed 2026-05-10)
- Splash: under test (splash_enabled=true in piclock.ini)
- OLED: logo + `ram-root` version string confirmed ✅ (from prior image)
- Passwordless SSH from Mac: confirmed ✅

### Key architecture note
- The Pi boots from the **prebuilt kernel's embedded initramfs** — NOT from `rootfs.cpio.gz` on FAT.
- `rootfs.cpio.gz` is generated but NOT loaded (`initramfs` line in `config.txt` is commented out by design).
- All runtime customisation must flow through **FAT partition** (`/boot/piclock/`) via `setup.sh`.
- `setup.sh` is sourced by `clock_pokemon.sh` before the clock loop starts.
- `S05bootsplash` and `/opt/clock8002/` in the cpio are unreachable at runtime on this boot model.

### piclock.ini feature toggles
- Location: `/boot/piclock/piclock.ini` (FAT — editable from any OS)
- `splash_enabled=true|false` — show bootsplash on boot

### Build infra
- Output dir: `/home/pi/output-root-ram-payload-20260509-165344` on cm5
- cm5 source repos: `~/clock8002-root-ram` (BR2_EXTERNAL) and `/tmp/clock8002-build-initramfs-20260510` (CLOCK8002_SOURCE_DIR)
- Standard build command (always use dirclean):
  ```
  screen -S br-build-<commit> -dm bash -lc "{ CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-root-ram-payload-20260509-165344 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot clock8002-dirclean && CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-root-ram-payload-20260509-165344 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot BR2_PICLOCKKEY=\"$(cat ~/.ssh/id_rsa.pub)\"; } > /tmp/br-build-<commit>.log 2>&1; echo BR_BUILD_EXIT:\$? > /tmp/br-build-<commit>.exit"
  ```

### Flash workflow
- Transfer: `scp pi@cm5.local:/home/pi/output-root-ram-payload-20260509-165344/images/sdcard.img /Users/jp/Desktop/piClock-<commit>-sdcard.img`
- Flash (user runs): `diskutil unmountDisk /dev/disk6 && sudo dd if=/Users/jp/Desktop/piClock-<commit>-sdcard.img of=/dev/rdisk6 bs=4m status=progress && diskutil eject /dev/disk6`

---

## Previous Checkpoint (2026-05-10 - 3f1ce4c flashed and confirmed working)

### TL;DR
- `piClock-3f1ce4c-sdcard.img` flashed and all changes confirmed working. ✅
- OLED splash: logo + version string `ram-root` confirmed appearing. ✅
- Mac passwordless SSH confirmed working (`jp@Sapporo.local` key baked in). ✅
- Root cause of previous missing version (on `113da10` image): incremental build skipped `clock8002-dirclean`, so stale oled-daemon (pre-`a296051` regex fix) ended up on FAT.
- Fix baked in: build command now always runs `clock8002-dirclean` before `make`. Documented in `copilot-instructions.md`.

### Commit chain (HEAD → oldest)
- `a478096` — HANDOFF.md update (**HEAD**)
- `3f1ce4c` — copilot-instructions: document clock8002-dirclean requirement for root-ram builds
- `113da10` — HANDOFF.md update
- `a296051` — oled_daemon: broaden gitTag regex to match non-semver tags
- `5dd5c53` — oled, build scripts, authorized_keys: Mac key baked in + OLED version fix

### Live device state
- Image: `piClock-3f1ce4c-sdcard.img` (flashed 2026-05-10)
- OLED: logo + `ram-root` version string confirmed ✅
- Passwordless SSH from Mac: confirmed ✅
- `/dev/i2c-1` present, `oled.ini`: `i2c_port=1`, `i2c_address=0x3c`, `rotation=2`

### Build infra
- Output dir: `/home/pi/output-root-ram-payload-20260509-165344` on cm5
- cm5 worktree: `/home/pi/clock8002-root-ram`
- Standard build command (always use dirclean):
  ```
  screen -S br-build-<commit> -dm bash -lc "{ CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-root-ram-payload-20260509-165344 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot clock8002-dirclean && CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-root-ram-payload-20260509-165344 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot BR2_PICLOCKKEY=\"$(cat ~/.ssh/id_rsa.pub)\"; } > /tmp/br-build-<commit>.log 2>&1; echo BR_BUILD_EXIT:\$? > /tmp/br-build-<commit>.exit"
  ```

### Flash workflow note
- User flashes SD card manually. Copilot: transfer image to Desktop and provide `dd` command only — do not run `dd`.

---

## Previous Checkpoint (2026-05-10 - 5dd5c53 flashed; Mac key + OLED version fix)

### TL;DR
- `piClock-b5b8416-sdcard.img` is on the Desktop, verified. Device powered off. Ready to flash to `/dev/disk6`.
- OLED version display fixed in source (`oled_daemon.py` `get_build_version()`) and baked into b5b8416 oled-daemon binary via PyInstaller in `clock8002.mk`.
- Mac passwordless SSH fixed live on running device. **Not baked into b5b8416 image** — will need re-adding after flash (see below).
- `golden-working-card/piclock/authorized_keys` in repo still has only cm5's key. Needs updating.

### Flash target
- Image: `/Users/jp/Desktop/piClock-b5b8416-sdcard.img`
- Disk: `/dev/disk6` (31.9 GB external, FAT `piClock` at `disk6s1`)
- Flash command (run manually):
  ```
  diskutil unmountDisk /dev/disk6 && sudo dd if=/Users/jp/Desktop/piClock-b5b8416-sdcard.img of=/dev/rdisk6 bs=4m status=progress && diskutil eject /dev/disk6
  ```

### What b5b8416 contains
- All OLED assets on FAT: `oled-daemon` (17329928B), `piclockLogo.bin` (1024B), `DejaVuSans.ttf` (756072B)
- `dtparam=i2c_arm=on` in `config.txt` → `/dev/i2c-1` available at boot
- `setup.sh` on FAT runs at boot: installs authorized_keys, starts oled-daemon
- `oled_daemon.py` `get_build_version()` fixed to use `SDL_CLOCK_PATH` (`/root/sdl-clock`) and Python byte-scan fallback — version should appear on OLED splash
- authorized_keys on FAT contains **cm5's key only** (built with `BR2_PICLOCKKEY=$(cat ~/.ssh/id_rsa.pub)` on cm5)

### Post-flash steps
1. Flash image, boot device
2. Verify `/boot/oled-daemon`, `/boot/piclockLogo.bin`, `/boot/DejaVuSans.ttf` present
3. Verify `/dev/i2c-1` present; oled-daemon in process list
4. Verify OLED shows splash with version number in lower right
5. **Re-add Mac key** (password login required first):
   ```sh
   ssh root@192.168.8.245  # password: clockworkadmin
   mount -o remount,rw /boot
   echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDYCpnx51J6CmS3VhfPxE+xlnb5zu48Nh5+hOXiQYXxNUJVd8uCf4NP7RWYda+cC72kVvKhTKWfG8ZYD7pyoR97YpoJSU7RPzZDs2VFh/us5dBwFL42+rU58VZrx5arN5gMl0h2WmCJlNjKXI8b2CcOJVomhdYDDfzPFNrpovawmmzrgJgOShehLwIOlvGt0OwZBzl12ucpOslm88YNMlDTHgQ2TEpVSdeeCd9N+KW+hAT48bKjeg3vrEbXUDaCpkeXsMARTBtIq+LYTYKXwDedLDMJkqsR6oaKFtHe76R9RP/ZCB+9nBKWv/NtCFNZ2daa8XOXgiuLWCzK5JD90Xohi2ObLEL98sAVX0ra55UMxq73Baspzjdgy0lsCwqSjKQqwD2AY38Oz9uXk0jZve2ugrAYvCJfkFsFe6KfYKhxWGlG8ebSbmKm4001l1lJwvY/k7UFo74mCHLuEDNENm43jdznbK81EDBpaxN1+y47gLgunWtry0vc6M9uurpOhKM= jp@Sapporo.local' >> /boot/piclock/authorized_keys
   echo 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDYCpnx51J6CmS3VhfPxE+xlnb5zu48Nh5+hOXiQYXxNUJVd8uCf4NP7RWYda+cC72kVvKhTKWfG8ZYD7pyoR97YpoJSU7RPzZDs2VFh/us5dBwFL42+rU58VZrx5arN5gMl0h2WmCJlNjKXI8b2CcOJVomhdYDDfzPFNrpovawmmzrgJgOShehLwIOlvGt0OwZBzl12ucpOslm88YNMlDTHgQ2TEpVSdeeCd9N+KW+hAT48bKjeg3vrEbXUDaCpkeXsMARTBtIq+LYTYKXwDedLDMJkqsR6oaKFtHe76R9RP/ZCB+9nBKWv/NtCFNZ2daa8XOXgiuLWCzK5JD90Xohi2ObLEL98sAVX0ra55UMxq73Baspzjdgy0lsCwqSjKQqwD2AY38Oz9uXk0jZve2ugrAYvCJfkFsFe6KfYKhxWGlG8ebSbmKm4001l1lJwvY/k7UFo74mCHLuEDNENm43jdznbK81EDBpaxN1+y47gLgunWtry0vc6M9uurpOhKM= jp@Sapporo.local' >> /root/.ssh/authorized_keys
   ```
6. **Persist Mac key in repo**: update `buildroot-external/board/clock8002-rpi5/golden-working-card/piclock/authorized_keys` to include Mac key so future builds don't require the live fix

### Known remaining work (not yet done)
- `golden-working-card/piclock/authorized_keys` still cm5-only → future builds still need live Mac key fix
- Version display unconfirmed on hardware (needs post-flash boot test)

### Build infra
- Output dir: `/home/pi/output-root-ram-payload-20260509-165344` on cm5
- BR2_EXTERNAL clone: `/tmp/clock8002-build-initramfs-20260510` (at `b5b8416`)
- .config patched: `BR2_PACKAGE_CLOCK8002_SOURCE_DIR` and `BR2_EXTERNAL_CLOCK8002_PATH` point to initramfs clone

---

## Previous Checkpoint (2026-05-10 - OLED asset staging fix + b5b8416 image)

### TL;DR
- Root cause of blank OLED on `8f6ac6c` image: `oled-daemon` and `piclockLogo.bin` were staged to `BINARIES_DIR` by `post-image.sh` but had no `mcopy` calls to inject them into the FAT. `DejaVuSans.ttf` was not staged at all.
- All three issues fixed in `b5b8416`. Build succeeded. FAT verified. Image on Desktop.

### Commit chain
- `3985557` — oled-daemon: build PyInstaller binary, ship on FAT, enable i2c_arm
- `8f6ac6c` — post-image.sh: inject BR2_PICLOCKKEY into piclock/authorized_keys on FAT
- `b5b8416` — post-image.sh: fix OLED asset staging to FAT (**HEAD**)

### What was fixed in b5b8416
- `BOOT_RUNTIME_FILES` now includes `DejaVuSans.ttf` — staged from `golden-working-card/root/` and mcopy'd to FAT root via existing loop.
- New `mcopy` loop after BOOT_RUNTIME_FILES loop injects `oled-daemon` and `piclockLogo.bin` to FAT root:
  ```sh
  for oled_asset in oled-daemon piclockLogo.bin; do
      [ -f "${BINARIES_DIR}/${oled_asset}" ] || continue
      MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
          "${BINARIES_DIR}/${oled_asset}" ::
  done
  ```

### Build and artifact status
- Build: `BR_BUILD_EXIT:0` for `b5b8416` (dirclean rebuild)
- Output dir: `/home/pi/output-root-ram-payload-20260509-165344` on cm5
- BR2_EXTERNAL clone: `/tmp/clock8002-build-initramfs-20260510`
- FAT verified via `mdir`:
  - `oled-daemon` — 17329928 bytes
  - `piclockLogo.bin` — 1024 bytes
  - `DejaVuSans.ttf` — 756072 bytes
- Local image: `/Users/jp/Desktop/piClock-b5b8416-sdcard.img`
- Flash target: `/dev/disk6` — user flashing manually

### OLED daemon known-good test result (on 8f6ac6c image before flash)
- After manual deploy of built binary to live device (`dd` over SSH stdin):
  - `modprobe i2c-dev` creates `/dev/i2c-1` (from `dtparam=i2c_arm=on` in config.txt)
  - `oled-daemon` with `i2c_port=1` and logo at `/boot/piclockLogo.bin` runs cleanly: `EXIT:0`, no tracebacks
  - OLED daemon binary is confirmed functional on the hardware
- Note: on the 8f6ac6c image the daemon only ran after manual file deploy; b5b8416 image has all assets on FAT from build time.

### setup.sh / authorized_keys mechanism status
- `setup.sh` is on FAT at `/boot/piclock/setup.sh`, confirmed present on device
- `clock_pokemon.sh` delegates to `setup.sh` if present (committed `e2fde39`)
- `BR2_PICLOCKKEY` at build time = cm5's `~/.ssh/id_rsa.pub` (injected into `piclock/authorized_keys` on FAT)
- Mac's SSH key is **not** in `authorized_keys` — password login required from Mac until Mac key is added to FAT's `authorized_keys`

### Stale .config path risk (known issue)
- After `clock8002-dirclean`, Buildroot tried to rsync from the old deleted clone path.
- Root cause: `BR2_PACKAGE_CLOCK8002_SOURCE_DIR` and `BR2_EXTERNAL_CLOCK8002_PATH` in the output dir's `.config` were hardcoded to `/tmp/clock8002-build-payload-rerun-20260509-184110`.
- Fix applied: `sed -i` on cm5 to point both to `/tmp/clock8002-build-initramfs-20260510`.
- Risk: this will recur if the output dir is reused after the source clone path changes. Always patch `.config` or use explicit `BR2_EXTERNAL=` override.

### Next steps
1. Boot b5b8416 image, confirm:
   - `/boot/oled-daemon`, `/boot/piclockLogo.bin`, `/boot/DejaVuSans.ttf` present
   - `/dev/i2c-1` present after boot (dtparam takes effect)
   - `oled-daemon` running in process list (started by `setup.sh`)
   - OLED display shows splash/info screen
2. Add Mac's public key to `/boot/piclock/authorized_keys` on FAT if passwordless SSH from Mac is required
3. Update HANDOFF.md after successful boot verification



## Current Checkpoint (2026-05-10 - issue #44 authorized_keys: definitive root cause)

### Definitive root cause — kernel-embedded initramfs

- `feature/root-ram` builds use `CLOCK8002_PREBUILT_KERNEL=1`. The prebuilt kernel has an **old initramfs baked into it** predating the authorized_keys commits.
- The sdcard.img produced by these builds has a **single FAT32 partition** only — no squashfs, no separate rootfs partition.
- `rootfs.cpio.gz` in the output `images/` dir is generated fresh each build (with the correct S03copy_clock_files), but `initramfs rootfs.cpio.gz followkernel` is **commented out** in config.txt.
- The external cpio is therefore **never loaded**. The device boots from the kernel-embedded initramfs — the old version without the authorized_keys block.
- All incremental rebuilds, target-dir patches, and squashfs stamp clears were irrelevant — they never changed what the device boots.

### What IS correct
- Both source files have the authorized_keys block committed (`bba6bc0`, `8ad295e`).
- The cm5 target/ has the correct block. post-build.sh ran. rootfs.cpio.gz was regenerated.
- Manual key auth works (confirmed on live unit): placing `/boot/piclock/authorized_keys` on the FAT partition and manually copying it to `/root/.ssh/authorized_keys` (mode 700/600) produces passwordless SSH via Dropbear.
- LTC is working on `cf519f7` image: `alsa-ltc plughw:2,0` is running, card 2 is present, no bandwidth errors.

### Rejected approaches
- Re-enabling `initramfs rootfs.cpio.gz followkernel` in config.txt: previously caused USB audio bandwidth failure at ~30s (`Not enough bandwidth for altsetting 0`). **Rejected.**
- Full kernel rebuild (drop `CLOCK8002_PREBUILT_KERNEL=1`): **Rejected.**

### Proposed path forward (pending user approval)
- Add the authorized_keys install block to a **FAT-resident runtime script** (e.g. `golden-working-card/clock_pokemon.sh` or `alsa-ltc_pokemon.sh`).
- These scripts are copied from the FAT partition by the kernel-embedded init at boot, then executed as root. `/boot/piclock/` is accessible at that point.
- Only `boot.vfat` changes — no kernel rebuild required.
- Status: **proposed, not yet approved.**

### Device state at session end
- Unit: `root@192.168.8.245` (hostname `sdl-clock`), running `piClock-cf519f7-authkeys-sdcard.img`
- Flashed image on Desktop: `/Users/jp/Desktop/piClock-cf519f7-authkeys-sdcard.img`
- SD card: `/dev/disk6`
- LTC running, USB audio card present, clock and clock-bridge running
- `/root/.ssh/` does NOT exist on fresh boot — passwordless SSH requires password

## Hard Rule (2026-05-09 - payload kernel + modules mandatory for test validation)

- For `feature/root-ram` test-image validation, the kernel and modules must come from the same prebuilt payload bundle.
- Required payload set is matched and complete: `Image`, `dtbs/` (or `dtb/`) + `overlays/`, and `modules/` (or `modules/lib/modules/`).
- Compile-kernel Buildroot image runs are non-compliant for this validation objective and must not be used for sign-off.
- Exception policy: only bypass this rule if the user explicitly overrides it in that session.
- Historical Mode A / Mode B notes below are archival context and are superseded by this hard rule.

## Current Checkpoint (2026-05-09 - issue #44 authorized_keys field provisioning fix)

### Problem confirmed on test image
- Placing `/boot/piclock/authorized_keys` on the boot partition did not enable passwordless SSH login.
- Root cause: Buildroot `post-build.sh` copies selected init scripts from `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/` into the target rootfs.
- The golden `S03copy_clock_files` script did not include the `/boot/piclock/authorized_keys` import block, even though the overlay copy did.

### Fix applied (local worktree)
- Added authorized key import to:
  - `buildroot-external/board/clock8002-rpi5/golden-working-card/etc/init.d/S03copy_clock_files`
- Added behavior at boot:
  - if `/boot/piclock/authorized_keys` exists, copy to `/root/.ssh/authorized_keys`
  - enforce `/root/.ssh` mode `700` and `authorized_keys` mode `600`

### Build status after fix
- Incremental payload test build launched on cm5 in screen session:
  - `br-build-root-ram-authkeys-20260509-214652`
- Result:
  - `PREP_RC:0`
  - `CONFIG_RC:0`
  - `DIRCLEAN_RC:0`
  - `BR_BUILD_EXIT:0`
- Artifact:
  - cm5 image: `/home/pi/output-root-ram-payload-20260509-165344/images/sdcard.img`
  - local copy refreshed: `/Users/jp/Desktop/piClock-4e7b636-sdcard.img` (timestamp `2026-05-09 21:48`)

### Validation state
- Runtime on-device verification of passwordless login from `/boot/piclock/authorized_keys` is pending after flash/boot of the refreshed image.

## Current Checkpoint (2026-05-09 - remove legacy /boot/clock.ini from future images)

### Decision
- To avoid user confusion from dual config files, legacy `/boot/clock.ini` must not be shipped in new images.
- Intended single boot-partition config path remains `/boot/piclock/clock.ini`.

### Implemented changes (feature/root-ram)
- `buildroot-external/board/clock8002-rpi5/post-image.sh` now excludes `clock.ini` in both `BOOT_SOURCE_DIR` copy loops.
- `buildroot-external/board/clock8002-rpi5/golden-working-card/boot/clock.ini` was removed so it cannot be staged into future artifacts.

### Runtime context from live target
- Live probe on `192.168.8.245` showed current booted card running `/root/sdl-clock -C /boot/clock.ini`.
- Cause: `/boot/clock_cmd.sh` override path on that flashed unit.
- Scope: this fix affects future builds/images only; already-flashed cards keep their existing boot partition files.

### Build state at capture
- Branch: `feature/root-ram`
- Active payload build session: `br-build-root-ram-payload-20260509-165344`
- Remote state: `RUNNING`, elapsed `5291s`, last package stamp `nano-8.2/.stamp_patched`
- Strict kernel compile marker scan: empty in sampled window.

## Current Checkpoint (2026-05-09 - payload incremental build in progress)

### Snapshot (local + cm5)
- Local repo snapshot at capture time:
  - `git branch --show-current` -> `feature/root-ram`
  - `git status --short` -> clean working tree
- Active payload build session on cm5:
  - session: `br-build-root-ram-payload-20260509-165344`
  - output dir: `/home/pi/output-root-ram-payload-20260509-165344`
  - state: `RUNNING`
  - elapsed at sample: `1263s`
  - last package stamp: `glibc-2.41-5-gcb7f20653724029be89224ed3a35d627cc5b4163/.stamp_configured`
- Kernel compile guardrail signal:
  - strict kernel compile marker scan returned no lines in the sampled window
  - build tail shows glibc userspace compile/configure activity

### Build cadence decision
- Development work should use incremental payload-mode builds to reduce turnaround time.
- Reserve full clean rebuilds for release-candidate prep unless the user explicitly overrides.

## Current Checkpoint (2026-05-09 - root-ram image build + flash verification)

### Build and artifact status
- Build session `br-build-root-ram-20260509-162611` on cm5 completed successfully:
  - `PATCH_RC:0`
  - `DIRCLEAN_RC:0`
  - `BR_BUILD_EXIT:0`
- Build source identity captured in the build log:
  - `REPO_BRANCH: feature/root-ram`
  - `REPO_HEAD: 74439b3`
- Generated image on cm5:
  - `/home/pi/buildroot/output/images/sdcard.img`
  - sha256: `be06a3026389ef2f06ed418229ff7a260620496d9611f41ff230651ea4a443eb`

### Transfer and flash prep
- Local artifact copied and named using commit convention:
  - `/Users/jp/Desktop/piClock-74439b3-sdcard.img`
- Hash parity confirmed between cm5 image and local Desktop copy (`SHA_MATCH:yes`).
- Existing Desktop image with same name was preserved as backup before overwrite:
  - `/Users/jp/Desktop/piClock-74439b3-sdcard.bak-20260509-163001.img`
- Flash target discovery snapshot at prep time:
  - external SD media detected as `/dev/disk6` (`piClock` FAT partition present).

### Runtime outcome on target
- User confirmed: flashed image booted and LTC is running.
- This validates the deployed image path at commit `74439b3` as currently bootable with active LTC on test hardware.

## Current Checkpoint (2026-05-09 - .246 parity reset and clean rebuild relaunch)

### What we locked
- Golden `.246` is now the authoritative runtime baseline for recovery.
- We identified the exact app build identity running on Golden:
  - `vcs.revision=74439b3a431824d96752eb21053f394a9a00a319`
  - `vcs.modified=false`
  - build tag includes `gitTag=ram-root`
- Kernel/runtime signatures from Golden were captured for parity:
  - `uname -r`: `6.12.41-v8`
  - `/boot/Image` sha256: `3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80`
  - `/boot/alsa-ltc` sha256: `c78c3fc8094dd701a9f63465641525998812db9c56be68f703173178eb830417`
  - `/boot/sdl-clock` sha256: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
  - `/boot/clock-bridge` sha256: `09c4bf22e4956a172153f636a8f311bb73c404f8aa8a67ebe89ec419eb7a75dd`
  - `/boot/config.txt` sha256: `fef6e7e51193d05897d410f2b8bfa9186b36dd801788c7d1a6cdd494d7318664`
  - `/boot/clock.ini` sha256: `fbd6e41e9010ee25ecd8f2ce4ba0dbfc6a06cbb83b8cc6b31320be09f852495f`
- Golden live process state confirms LTC stack is active together (`alsa-ltc`, `clock-bridge`, `sdl-clock`).

### CM5 drift found
- `/home/pi/clock8002` was not at parity target (`feature/squashfs-readonly`, head `fb6d5d4754b1`, dirty).
- `/home/pi/clock8002-root-ram` was on the right commit line (`feature/root-ram`, head `74439b3a4318`) but dirty.
- Conclusion: do not trust either dirty tree for parity rebuilds.

### Recovery action in progress
- A clean source-of-truth clone was created on cm5 and pinned to the Golden app commit:
  - clone: `/tmp/clock8002-recover-74439b3-20260509-153727`
  - commit: `74439b3a431824d96752eb21053f394a9a00a319`
  - status: clean (`CLONE_STATUS_COUNT:0`)
- Build relaunched from that clean clone in `screen`:
  - session: `br-recover-74439b3-20260509-153727`
  - output: `/home/pi/output-root-ram-recover-20260509-153727`
  - log: `/tmp/br-recover-74439b3-20260509-153727.log`
  - exit marker: `/tmp/br-recover-74439b3-20260509-153727.exit`
- Required startup monitor policy executed: 10-second post-launch watch completed with no immediate `error/fatal/failed` signatures.

### Operator monitor commands
1. Live log:
   - `ssh pi@cm5.local 'screen -ls; tail -f /tmp/br-recover-74439b3-20260509-153727.log'`
2. Completion:
   - `ssh pi@cm5.local 'cat /tmp/br-recover-74439b3-20260509-153727.exit'`
3. Error scan:
   - `ssh pi@cm5.local 'grep -niE "error:|\*\*\* .*Error|fatal:|failed|permission denied|no space left|memory exhausted" /tmp/br-recover-74439b3-20260509-153727.log | tail -n 120 || true'`

## Current Checkpoint (2026-05-09 - payload evaluation and immediate LTC pivot policy)

### Decision summary
- Evaluated Clock-8001 GitLab `images_64b` payload as a Plan B candidate.
- Result: not directly usable under current fallback hooks because kernel modules are missing.
- Canonical working bundle location remains:
  - `/srv/clock8002/prebuilt-kernel-bundles/<bundle-id>/`
  - `/srv/clock8002/prebuilt-kernel-bundles/current`
- Operational policy confirmed: if a new build fails LTC reliability at runtime, pivot immediately to bundled kernel+modules using a forced Plan B build flow (manual override), rather than waiting for compile-failure auto-gating.

### Evidence recorded
- Raw payload verify failed: `BUNDLE_STATUS:missing_dtbs` (layout mismatch for current verifier expectations).
- Normalized candidate (`Image` + `dtbs/` + `overlays/`) verify failed: `BUNDLE_STATUS:missing_modules`.
- Payload image observed:
  - SHA256: `6f1a835f7070dbb7b4e94fb2bb2f9407b53d1df018cc260fbf1123cdcfd9b846`
  - kernel hint: `Linux 6.12.41-v8`, 4K pages
- No `modules` tree or `.ko*` files found in payload inventory.

### Scope boundary
- Investigation into sourcing matching modules from Clock-8001 payload path is intentionally paused by user direction.
- Keep payload as a boot-assets reference only until a complete matched kernel bundle is available.

## Current Checkpoint (2026-05-09 - post-boot checklist moved to issue tracking)

### Tracking
- Created GitHub issue: `#44` "Post-boot feature checklist (track one-by-one after bootable image)".
- Applied issue metadata:
  - labels: `buildroot`, `enhancement`
  - milestone: `v0.2.0-beta`

### Locked decisions
- Keep build-environment indicator in Web UI.
- OLED build-environment indicator is dropped; OLED role remains WiFi indicator.
- Power button support is required and must be restored in the post-boot sequence.
- `machine-id` persistence remains in scope while `/var/lib` is tmpfs; re-evaluate if runtime storage model changes.

### Intent
- The checklist is now issue-driven and executed one feature at a time only after a reliably bootable image is available.
- No branch closure actions were taken as part of this tracking update.

## Current Checkpoint (2026-05-09 - Plan B locked as fallback-only)

### Decision
- Plan B is approved, but only as a fallback path.
- Default path remains: compile kernel in the Buildroot environment.
- Plan B activation condition: if we cannot compile a working kernel in the build environment, use the known-good prebuilt kernel from the Golden build instead of compiling kernel sources.

### Plan B definition (exact scope)
- Prebuilt kernel bundle must be injected as a matched set:
  - kernel image (`Image`)
  - device trees and overlays
  - kernel modules
- All three must come from the same known-good Golden kernel release.

### Activation gate (must all hold)
1. Normal kernel compile path is attempted first.
2. Kernel compile fails or cannot produce a working kernel.
3. Prebuilt kernel bundle checksum/version checks pass.
4. Build provenance is explicitly marked as fallback-kernel output.

### Current repo readiness (implemented)
- Fallback hooks are now implemented in:
  - `buildroot-external/board/clock8002-rpi5/post-build.sh`
  - `buildroot-external/board/clock8002-rpi5/post-image.sh`
- Bundle management scripts are now implemented in:
  - `buildroot-external/scripts/verify-prebuilt-kernel-bundle.sh`
  - `buildroot-external/scripts/promote-prebuilt-kernel-bundle.sh`
- Canonical Plan B bundle store on cm5:
  - `/srv/clock8002/prebuilt-kernel-bundles/<bundle-id>/`
  - `/srv/clock8002/prebuilt-kernel-bundles/current` symlink used as the default `CLOCK8002_PREBUILT_KERNEL_BUNDLE` path.
- Current repo-staged Golden payload still includes runtime userspace assets only; a complete prebuilt kernel bundle (Image + dtbs/overlays + modules) must be supplied externally when Mode B is invoked.
- Default behavior remains payload-mode with prebuilt kernel bundle; kernel-from-source is opt-in only.
- Build mode selection documentation now lives in:
  - `buildroot-external/README.buildroot.md` -> "Build Mode Selection (A vs B)"

## Current Checkpoint (2026-05-09 - corrected 4K parity gate + root-ram source rebind + new build launch)

### TL;DR
- User correction accepted: parity gate now treats 4K pages as the expected baseline (not 16K).
- The prior no-go run was stopped:
  - session: `/tmp/br-resume-20260509-114304` (screen session `br-resume-20260509-114304` terminated)
- Buildroot was re-bound away from stale `/tmp/clock8002-kpin-*` sources and now points to the feature worktree:
  - BR2 external path: `/home/pi/clock8002-root-ram/buildroot-external`
  - app source dir: `/home/pi/clock8002-root-ram/v4`
  - active source branch/head: `feature/root-ram` @ `74439b3`
- Kernel naming normalization applied for clarity:
  - `CONFIG_LOCALVERSION="-v8"` (removed `-v8-4k` suffix)
- Corrected prebuild parity gate state on cm5 now passes (`READY:yes`):
  - `BR2_ARM64_PAGE_SIZE_4K=y`
  - `# BR2_ARM64_PAGE_SIZE_16K is not set`
  - `BR2_LINUX_KERNEL_CUSTOM_TARBALL_LOCATION=...590178d58b730e981099fdcb405053a000e79820...`
  - `BR2_LINUX_KERNEL_PATCH=""`
  - `CONFIG_LOCALVERSION="-v8"`
- New build launched from corrected inputs:
  - session: `br-rootram-20260509-123446`
  - log: `/tmp/br-rootram-20260509-123446.log`
  - exit marker: `/tmp/br-rootram-20260509-123446.exit`
  - required 10s startup monitor completed; no immediate `error:` / `fatal:` / `failed` / `memory exhausted` signatures in startup scan.

### Current state
- Build is running in `host-gcc-final` stages at last sample.
- cm5 repository `/home/pi/clock8002` remains on `feature/squashfs-readonly` with local dirt, but the active build inputs are now explicitly sourced from `/home/pi/clock8002-root-ram`.

### Operator monitor commands (single-command copy friendly)
1. Live log:
   - `ssh pi@cm5.local 'L=$(ls -1t /tmp/br-rootram-*.log 2>/dev/null | head -n1); tail -f "$L"'`
2. Completion state:
   - `ssh pi@cm5.local 'E=$(ls -1t /tmp/br-rootram-*.exit 2>/dev/null | head -n1); if [ -n "$E" ]; then cat "$E"; else echo BUILD_STATE:RUNNING; fi'`
3. Error scan:
   - `ssh pi@cm5.local 'L=$(ls -1t /tmp/br-rootram-*.log 2>/dev/null | head -n1); grep -niE "error:|\*\*\* .*Error|fatal:|failed|memory exhausted" "$L" | tail -n 120 || true'`

## Current Checkpoint (2026-05-09 - cm5 Buildroot recovery: host-bison bypass + monitored resume)

### TL;DR
- Active Buildroot run is now in the standard cm5 tree:
  - build root: `/home/pi/buildroot`
  - session: `br-resume-20260509-114304`
  - log: `/tmp/br-resume-20260509-114304.log`
  - exit marker: `/tmp/br-resume-20260509-114304.exit`
- Repeated `host-bison-3.8.2` build failures were encountered in sequence (`M4`, package/version defines, `wctype_t`, `strverscmp`, `dupfd`, `FLAG_LOCALIZED`).
- Even after compile fixes, the package-built binary remained unusable (`src/bison: memory exhausted` on `--version`/`--help`), so continuing to patch that build artifact was not reliable.
- Recovery action on cm5:
  - installed Debian `bison` package (`/usr/bin/bison`, `/usr/bin/yacc`)
  - copied working tools into Buildroot host bin:
    - `/home/pi/buildroot/output/host/bin/bison`
    - `/home/pi/buildroot/output/host/bin/yacc`
  - marked host-bison package as completed for this working tree:
    - `/home/pi/buildroot/output/build/host-bison-3.8.2/.stamp_built`
    - `/home/pi/buildroot/output/build/host-bison-3.8.2/.stamp_host_installed`
- Required startup check was executed on the relaunched run:
  - 30s monitor of `/tmp/br-resume-20260509-114304.log`
  - result: no immediate `error:` lines during the window
  - build progressed through `host-gawk`, `host-mpc`, and into `host-gcc-initial`

### Current state
- `br-resume-20260509-114304` is still running.
- Latest sampled tail contains configure/build progress only (no current fatal/error lines).

### Operator monitor commands
1. Live monitor:
   - `ssh pi@cm5.local 'screen -ls; tail -f /tmp/br-resume-20260509-114304.log'`
2. Completion check:
   - `ssh pi@cm5.local 'cat /tmp/br-resume-20260509-114304.exit'`
3. Error-only view:
   - `ssh pi@cm5.local 'grep -niE "error:|\*\*\* .*Error|fatal:|failed" /tmp/br-resume-20260509-114304.log | tail -n 120'`

### Important caveat
- This unblocks the active cm5 working tree, but the host-bison fallback is operational debt. A clean/reproducible fix should be added to Buildroot/package patching later so a fresh tree can build without manual stamp/tool injection.

## Prior Checkpoint (2026-05-09 - reflashed A/B proof + build-level kernel delta isolation)

### TL;DR
- Reflashed failing image on test unit `192.168.8.245` booted as expected with failing kernel line:
  - `uname`: `6.12.41-v8-4k #4`
  - `/boot/Image` sha256: `c657863dc1a45f713c1e762c2c615e98d6e90e3a276ad44e65e1f8f458e5b99e`
- Controlled baseline manual trigger on `.245` (autostart gate temporarily disabled):
  - command: `/root/alsa-ltc hw:CARD=Device,DEV=0 255.255.255.255 1245`
  - result: exited early with `rc=1`
  - kernel delta: `usb_set_interface failed (-110)` count increased by `+1`
  - userspace symptom: `cannot set parameters (Connection timed out)`
- Swapped only kernel image + matching modules from Golden `.246` onto `.245` and rebooted:
  - new `/boot/Image` sha256 on `.245`: `3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80`
  - runtime kernel on `.245`: `6.12.41-v8 #3`
- Post-swap manual trigger (same command, 30s hold):
  - process remained running full interval (`rc=143` only because it was intentionally terminated)
  - kernel delta: no new `usb_set_interface failed` lines (`+0`)
  - LTC decode loop stable (continuous dot output)
- Startup behavior restored after controlled test:
  - `/boot/enable_ltc` re-enabled
  - `S99alsa-ltc` manually started
  - `alsa-ltc_pokemon.sh` + `alsa-ltc` confirmed running

### Host identity guardrail (important)
- During reboot probes, `.local` hostnames were ambiguous and resolved to Golden.
- Use direct IPs for proof runs:
  - test unit: `192.168.8.245`
  - golden reference: `192.168.8.246`

### Build-level delta isolation (config + patch lineage)
- Failing 4k kernel lineage from cm5 Buildroot output (`/home/pi/output-root-ram-goldencopy-20260509-000403`):
  - `BR2_LINUX_KERNEL_CUSTOM_TARBALL=y`
  - `BR2_LINUX_KERNEL_CUSTOM_TARBALL_LOCATION="$(call github,raspberrypi,linux,b1b490bae0bb8ad62d1f028ed0dcbbe7395a964d)/linux-b1b490bae0bb8ad62d1f028ed0dcbbe7395a964d.tar.gz"`
  - `BR2_LINUX_KERNEL_DEFCONFIG="bcm2712"`
  - `BR2_LINUX_KERNEL_CONFIG_FRAGMENT_FILES="$(BR2_EXTERNAL_CLOCK8002_PATH)/board/clock8002-rpi5/linux.config"`
- Kernel local patch stack for failing build appears empty:
  - `/home/pi/output-root-ram-goldencopy-20260509-000403/build/linux-custom/.applied_patches_list` has `0` lines
- Failing 4k compile provenance:
  - `UTS_RELEASE: 6.12.41-v8-4k`
  - `LINUX_COMPILE_BY/HOST: pi@cm5`
  - compiler string includes `Buildroot 2025.11`
- Extracted failing 4k config confirms key USB/xHCI settings:
  - `CONFIG_ARM64_4K_PAGES=y`
  - `CONFIG_USB_XHCI_HCD=y`
  - `CONFIG_USB_XHCI_PCI=y`
  - `CONFIG_USB_XHCI_PLATFORM=y`
  - `CONFIG_SND_USB_AUDIO=m`
  - `CONFIG_USB_DEFAULT_PERSIST=y`
  - `CONFIG_USB_AUTOSUSPEND_DELAY=2`
  - `CONFIG_IOMMU_DEFAULT_DMA_STRICT=y`
- Working v8 config could not be extracted from the Golden image (`extract-ikconfig: Cannot find kernel config`), and Golden does not expose `/proc/config.gz` or equivalent installed config files.

### Tight kernel error-path mapping for `-110`
- Exact log string source for observed failure:
  - `sound/usb/endpoint.c` in `endpoint_set_interface()` logs:
    - `"%d:%d: usb_set_interface failed (%d)"`
- Call path for failure at stream/interface activation:
  - `endpoint_set_interface()` -> `usb_set_interface()` -> USB core control transfer
- Timeout translation site:
  - `drivers/usb/core/message.c` (`usb_start_wait_urb`):
    - completion timeout returns `-ETIMEDOUT` (observed as `-110`)
- Retry behavior nuance:
  - ALSA USB endpoint path retries only `-EPROTO` (bounded retries)
  - `-ETIMEDOUT` is not retried there, so the error becomes an immediate hard failure at hw_params/stream start.

### Ranked suspect list (post-isolation)
1. Kernel/xHCI control-transfer timeout behavior specific to the `6.12.41-v8-4k` line (highest confidence from same-unit A/B inversion).
2. USB runtime power/LPM interaction during interface altsetting transition on the failing line.
3. Defconfig/platform interaction (`bcm2712` + 4k lineage) affecting timing/resource behavior in the USB audio start path.
4. Userspace retry/restart policy amplifies impact but is not the primary trigger (proven by baseline swap test).

### Recommended next diagnostics (if deeper proof is needed)
1. On a fresh failing 4k boot, capture precise pre/post trigger dmesg windows around each manual run with focused xHCI verbosity.
2. Add dynamic debug for relevant USB core + ALSA USB endpoint paths to capture return propagation to `endpoint_set_interface()`.
3. Keep same command/test cadence used in this proof to preserve comparability.

## Previous Checkpoint (2026-05-09 - deterministic manual-trigger failure with enhanced USB/xHCI forensics)

### TL;DR
- Startup loop remains disabled by policy: `/boot/enable_ltc` is absent, `S99alsa-ltc` does not autostart LTC, and no `alsa-ltc_pokemon.sh` loop process is active.
- Enhanced monitor is running and writing second-by-second snapshots plus per-error forensic bundles:
  - active monitor dir: `/boot/piclock/usb-monitor/failed-usb-live-20260509-170631`
  - recent events: `event-20260509-170809-n0005` through `event-20260509-170824-n0008`
- Controlled trigger loop result (manual `/root/alsa-ltc plughw:CARD=Device,DEV=0 255.255.255.255 1245`, 4 consecutive runs):
  - 4/4 runs failed with exit code 1 at ALSA hw params set stage
  - each run produced exactly one new kernel error: `usb 1-1.3: 2:1: usb_set_interface failed (-110)`
  - cumulative error count increased from 4 to 8 in lockstep with run count
- During this failure window, the USB audio device remained enumerated and ALSA-visible (`card0=Device`), and no host-controller death/disconnect cascade occurred.
- Low-level endpoint context snapshots across n0005->n0008 show progressive degradation:
  - control endpoint state transitioned `running -> stopped`
  - isochronous IN endpoint transitioned `running -> disabled`
  - slot remained configured and USB topology remained present

### Current interpretation
- Proximate cause is now strongly established: stream/interface activation on the C-Media USB device times out in `usb_set_interface` (`-110`).
- Continuous retry loops can amplify damage/rate of failure, but are not the primary trigger; a single manual run reproduces the same failure signature.
- Current data narrows the issue to the kernel/xHCI handling path on this image line, but does not yet provide line-level kernel proof (usbmon/USB trace events unavailable on this build).

### Current test posture
1. Keep startup script disabled (manual testing only).
2. Keep enhanced monitor running for event-bundle collection.
3. Use controlled manual triggers for reproducible, low-noise evidence.

## Earlier Checkpoint (2026-05-09 - Phase 1 diff: ALSA card ordering hypothesis, now superseded as primary root cause)

### TL;DR
- Userspace between Golden (`root@192.168.8.246`, LTC works) and Buildroot replica (`root@192.168.8.245`, LTC fails) is **byte-identical**: same sha256 for `alsa-ltc`, `sdl-clock`, `clock-bridge`, every `*_cmd.sh` / `*_pokemon.sh`, `/root/clock.ini`, fonts, and every `S03copy_*` / `S99*` init script. The Buildroot copy-based pivot is correctly delivering Golden's userspace.
- Two systemic differences remain:
  1. **Kernel build differs.** Golden runs `6.12.41-v8` (16 KiB-page, 73 MB Image, sha `3062308d…`). Buildroot replica runs `6.12.41-v8-4k` (4 KiB-page, 132 MB Image, sha `c657863d…`). cm5 build output's `Image` hash matches the flashed unit, so flash is faithful — the build is just selecting the `-v8-4k` kernel variant.
  2. **ALSA card index ordering is reversed.** Golden: card0=vc4hdmi0, card1=vc4hdmi1, **card2=USB-Audio C-Media**. Failed replica: **card0=USB-Audio C-Media**, card1=vc4hdmi0, card2=vc4hdmi1. `alsa-ltc_cmd.sh` hardcodes `plughw:2,0`, which on Golden points at the C-Media USB ADC (the actual LTC source) but on the replica points at vc4-hdmi playback — `arecord` therefore cannot capture LTC. The hifiberry-dacplusadcpro overlay loads on both units but produces no card on either; the C-Media USB ADC is the real LTC source on both.
- Snapshots and full three-column diff captured in `/tmp/clock8002-phase1/`:
  - `golden-246.txt`, `failed-245.txt`, `cm5.txt`
  - `PHASE1_DIFF.md` (sections A–G + root-cause + Strategy A/B/C/D menu)

### Strategy menu (no build changes yet — awaiting user choice)
- **B (recommended):** change `plughw:2,0` → `plughw:CARD=Device,DEV=0` in `buildroot-external/board/clock8002-rpi5/golden-working-card/root/alsa-ltc_cmd.sh`. One-line, kernel-independent, smallest diff.
- **C:** pin card indices via `/etc/modprobe.d/alsa-base.conf` (`options snd-usb-audio index=2`, `options snd_bcm2835 index=0`).
- **A:** force Buildroot to build the `-v8` (16 KiB-page) Pi5 kernel to match Golden. Most invasive.
- **D:** rebuild `alsa-ltc` as a Buildroot package — already proven unnecessary (binary is byte-identical).

### Suggested read-only validation before any build change
On both live units:
```sh
arecord -L | grep -A1 -i 'CARD=Device'
arecord -D plughw:CARD=Device,DEV=0 -d 1 -f S16_LE -r 48000 /tmp/probe.wav && echo OK
```
If both succeed, Strategy B is confirmed viable.

### Live units (both password `clockworkadmin`)
- Golden: `root@192.168.8.246` (`sdl-clock` hostname). LTC works.
- Failed: `root@192.168.8.245` (`piClock` hostname). LTC fails as described above.

## Previous Checkpoint (2026-05-09 - fuller Golden `/boot` image still not an exact Golden replica)

### Local repository status
- Branch: `feature/root-ram`
- HEAD: `74439b3`
- The working tree now includes a copy-based Buildroot pivot on top of earlier root-ram/parity changes.
- Important local delta for this checkpoint:
  - New copied working-card payload under `buildroot-external/board/clock8002-rpi5/golden-working-card/`
  - Build glue changes in `buildroot-external/board/clock8002-rpi5/post-build.sh` and `buildroot-external/board/clock8002-rpi5/post-image.sh`
  - Earlier parity/runtime edits remain present in the working tree and were overlaid into the remote build clone as part of this run.

### What This Build Is
- This is still a normal Buildroot image build, not a raw SD-card clone.
- Buildroot continues to supply the base userspace, package graph, kernel, and image construction.
- The new part is that the runtime payload is copied from the captured working card instead of being reconstructed from the newer overlay assumptions.
- The image shape remains the current root-ram model in this branch: single FAT boot partition plus kernel-embedded initramfs.

### Copy-Based Payload Pivot
- New source-of-truth payload directory:
  - `buildroot-external/board/clock8002-rpi5/golden-working-card/`
- Contents copied from the working card snapshot include:
  - boot payload files (`clock.ini`, `config.txt`, `cmdline.txt`, `interfaces`, `ntp.conf`, `enable_*` flags)
  - selected init scripts (`S03copy_*`, `S99alsa-ltc`, `S99clock`, `S99clock_bridge`)
  - persistent `/root` runtime payload (`sdl-clock`, `alsa-ltc`, `clock-bridge`, wrapper scripts, fonts)
- Buildroot glue behavior changed accordingly:
  - `post-build.sh` now copies the golden rootfs payload into `TARGET_DIR`
  - `post-build.sh` removes the synthetic `/root` tmpfs path and drops the old `S02setup-root` / `S98oled` behavior for this build path
  - `post-image.sh` now syncs the golden boot payload into `BINARIES_DIR` and boot.vfat on every build

### Active cm5 Build
- Build host: `pi@cm5.local`
- Buildroot tree: `/home/pi/buildroot-2025.11`
- Fresh source clone for this run: `/tmp/clock8002-root-ram-goldencopy-20260509-000403`
- Remote clone was overlaid with the local uncommitted working tree before the build was launched.
- Output tree: `/home/pi/output-root-ram-goldencopy-20260509-000403`
- `screen` session: `br-root-ram-goldencopy-20260509-000403`
- Initial build log: `/tmp/br-root-ram-goldencopy-20260509-000403.log`
- Initial exit marker: `/tmp/br-root-ram-goldencopy-20260509-000403.exit`
- Targeted rerun log: `/tmp/br-root-ram-goldencopy-20260509-000403-rerun.log`
- Targeted rerun exit marker: `/tmp/br-root-ram-goldencopy-20260509-000403-rerun.exit`
- Second incremental rerun log: `/tmp/br-root-ram-goldencopy-20260509-000403-rerun2.log`
- Second incremental rerun exit marker: `/tmp/br-root-ram-goldencopy-20260509-000403-rerun2.exit`
- Current observed state:
  - initial full build reached `post-image.sh` and failed during genimage boot image creation
  - root cause was a CRLF `kernel=Image^M` line in the copied golden `config.txt`, which corrupted the generated `genimage.cfg`
  - local fix in `buildroot-external/board/clock8002-rpi5/post-image.sh` strips `\r` from the extracted kernel filename before generating the genimage file list
  - first rerun of the existing output tree completed successfully with `BR_BUILD_EXIT:0`
  - live Golden-vs-replica comparison then showed the earlier copy-based image still was not a literal Golden `/boot` clone because the FAT root lacked Golden runtime payload files (`alsa-ltc`, `sdl-clock`, `clock-bridge`, related launcher scripts), firmware root files (`start*.elf`, `fixup*.dat`, `bootcode.bin`), and `voices/`
  - `post-image.sh` was then expanded to inject the captured Golden runtime payload from `golden-working-card/root`, the Buildroot `rpi-firmware` root files, and the repo `v4/voices/` tree into `boot.vfat`
  - second incremental rerun completed successfully with `BR_BUILD_EXIT:0`
  - the resulting image `/Users/jp/Desktop/piClock-74439b3-goldencopy-fullboot-sdcard.img` was flashed and tested
  - user-reported result: it failed again and is still not an exact copy of the Golden build
  - no fresh post-flash live diff was captured in this turn, so the remaining mismatch is still unresolved
- Produced artifacts:
  - cm5 image: `/home/pi/output-root-ram-goldencopy-20260509-000403/images/sdcard.img`
  - local copy from first rerun: `/Users/jp/Desktop/piClock-74439b3-goldencopy-sdcard.img`
  - first rerun sha256: `760d9c3110289a376f5ac097648f52f3af5add822461244d5630fc612a2676bc`
  - local copy from second rerun: `/Users/jp/Desktop/piClock-74439b3-goldencopy-fullboot-sdcard.img`
  - second rerun sha256: `1d4014e89ae8d2b6f083e916204b841de6639da5256eabc6190da63f2364619a`
  - verified `boot.vfat` from the second rerun now contains the previously missing Golden-style FAT-root payload entries including `alsa-ltc`, `sdl-clock`, `clock-bridge`, `start.elf`, `start4.elf`, `fixup.dat`, `fixup4.dat`, `bootcode.bin`, and `voices/`
- Host patch note:
  - `apply-build-host-patches.sh` completed and warned that current `rpi-firmware` version/hash content in the cm5 Buildroot tree was unexpected, but the launch continued and did not fail at that step

### Next actions
1. Capture an exact live diff between Golden `.246` and the newly failed flashed image before making more build changes, starting with `/boot`, copied `/root` payload, service state, and ALSA enumeration.
2. Determine whether the remaining mismatch is payload content, boot sequence, or hardware-driven device ordering rather than missing FAT-root files.
3. Only then adjust build glue or image composition again; the fuller `/boot` staging by itself was insufficient.

## Current Checkpoint (2026-05-09 - live unit on `ram-root` SDL3 binary)

### Local repository status
- Branch: `feature/root-ram`
- HEAD: `74439b3`
- Live-unit validation in this checkpoint stayed on the current branch and did not require repository source edits beyond recording the outcome here.

### Live test unit state
- Unit access:
  - primary SSH: `root@10.0.0.192`
  - secondary/static alias: `192.168.8.245`
- The unit was moved from alias `.244` to `.245` by updating both `/boot/interfaces` and `/etc/network/interfaces`, then applying the alias live on `eth0:1`.
- `/boot/config.txt` on the unit was missing the repo UART overlay lines; the following were added and then validated after reboot:
  - `dtoverlay=uart2`
  - `dtoverlay=uart3`
  - `dtparam=uart0=on`
- After reboot the unit exposed `/dev/ttyAMA0`, `/dev/ttyAMA2`, `/dev/ttyAMA3`, and `/dev/ttyAMA10`.

### Clock/runtime validation
- Repo default `v4/clock.ini.default` is now active on the unit and stable.
- Current key lines in `/boot/clock.ini`:
  - `app-version=ram-root`
  - `Format12h=true`
  - `source1.text=Limitimer`
  - `source1.counter=5`
  - `limitimer-mode=receive`
  - `limitimer-serial=/dev/ttyAMA3`
- Web config remains enabled and reachable on both:
  - `http://10.0.0.192/`
  - `http://192.168.8.245/`
  - auth: `admin` / `clockwork`
- `feature/squashfs-readonly` was searched for the AM/PM / 12-hour text clock work; the relevant SDL3 formatting code is already present on `feature/root-ram` too. The missing behavior on the unit was due to running the old exact `v1.3.5` binary, not a missing code path on `feature/root-ram`.
- Final live binary deployed on the unit:
  - source branch: `feature/root-ram`
  - source commit: `74439b3`
  - cm5 build command shape: `make GIT_TAG=ram-root sdl3-clock`
  - installed hash: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
- Post-install restart validated successfully; live processes are `/root/clock_pokemon.sh start` and `/root/sdl-clock -C /boot/clock.ini`.

### Intermediate findings worth preserving
- BusyBox/Buildroot target did not support modern SFTP-based `scp`; the reliable transfer path was SSH streaming via the local machine with `sshpass`, or legacy `scp -O` where auth permits.
- Foreground `ssh` runs of `sdl-clock` can fail with `Failed to initialize SDL: No available video device`; normal service startup under KMS/DRM is the valid runtime check.
- A full-clone tagged build from `feature/squashfs-readonly` improved version stamping over the shallow-clone fallback, but the final user-facing runtime label needed to read `ram-root`, so the live deployment returned to `feature/root-ram` with explicit `GIT_TAG=ram-root`.

### Open follow-ups
1. The background Buildroot image build on cm5 from `/tmp/clock8002-root-ram-build-current` was not re-polled during the live-unit validation work; refresh its status before using that image path again.
2. If branch test builds should always present a friendly runtime label, decide whether to keep passing `GIT_TAG=ram-root` explicitly or create a real Git tag for that line of work.

## Current Checkpoint (2026-05-08 - SDL3-only `.244` parity build running on cm5)

### Local repository status
- Branch: `feature/root-ram`
- HEAD: `74439b3`
- Relevant uncommitted parity/runtime edits in the working tree:
  - Board/runtime payload: `buildroot-external/board/clock8002-rpi5/{cmdline.txt,config.txt,post-image.sh,interfaces,enable_clock,enable_ltc}`
  - Rootfs overlay init/watchdog scripts: `buildroot-external/board/clock8002-rpi5/rootfs-overlay/etc/init.d/{S02setup-root,S03copy_alsa-ltc_files,S03copy_clock_files,S99alsa-ltc,S99clock}`
  - Rootfs overlay command wrappers: `buildroot-external/board/clock8002-rpi5/rootfs-overlay/opt/clock8002/{alsa-ltc_cmd.sh,alsa-ltc_pokemon.sh,clock_cmd.sh,clock_pokemon.sh}`
  - Build/package plumbing: `buildroot-external/configs/clock8002_rpi5_defconfig.sample`, `buildroot-external/external.mk`, `buildroot-external/package/clock8002/{Config.in,clock8002.mk}`, `buildroot-external/scripts/apply-build-host-patches.sh`
- Change intent:
  - Keep the `.244`-style boot/runtime behavior (`/boot` payload, `/root` tmpfs seeding, enable-file gating, explicit `plughw:2,0` for LTC) while preserving the current SDL3 clock implementation.
  - `sdl-clock` remains only a deployed compatibility name; the binary itself is built from `v4/cmd/sdl3-clock` and installed alongside `sdl3-clock`.
  - Do not use SDL2 and do not revive the old standalone `clock-bridge` or anything from `v3`.

### Key decisions locked in
- The working `.244` unit's deployed `sdl-clock` binary is SDL3-based, not SDL2-based.
- The active code path stays in `v4` only; `v4/cmd/sdl3-clock` is the source of truth for the deployed clock binary.
- The bridge behavior already exists inside the `v4` engine/OSC path, so no separate `clock-bridge` service is being carried forward into the Buildroot image.
- Active Go path is already current enough for this work: `v4/go.mod` declares `go 1.24.0`, and Buildroot `2025.11` on cm5 provides Go `1.24.2`.

### cm5 Buildroot 2025.11 build state
- Dedicated Buildroot tree on cm5: `/home/pi/buildroot-2025.11`
- Fresh source clone used for this run: `/tmp/clock8002-root-ram-build-current`
- Output tree: `/home/pi/output-root-ram-sdl3alias-20260508`
- Active `screen` session: `br-root-ram-sdl3alias-20260508`
- Build log: `/tmp/br-root-ram-sdl3alias-20260508.log`
- Exit marker: `/tmp/br-root-ram-sdl3alias-20260508.exit`
- Relaunch sequence already completed:
  - Synced the current local uncommitted patchset into the fresh cm5 clone.
  - Added the previously missing `enable_clock` and `enable_ltc` sentinel files to the repo so the `.244`-style boot gates are explicit and can be staged into FAT.
  - Made `buildroot-external/scripts/apply-build-host-patches.sh` version-aware so the earlier Mesa `25.2.7` hash mismatch does not recur on Buildroot `2025.11`.
  - Copied the sample defconfig into the cm5 Buildroot tree and rewired `BR2_PACKAGE_CLOCK8002_SOURCE_DIR` to the fresh clone's `v4` path before launching the build in `screen`.
- Current observed state:
  - `BUILD_RUNNING`
  - Latest observed log tail shows the build progressing through toolchain/binutils compilation rather than failing in the earlier Mesa download step.

### Next actions
1. Wait for `/tmp/br-root-ram-sdl3alias-20260508.exit` and confirm `BR_BUILD_EXIT:0` or inspect the first failing step if it exits non-zero.
2. If the build succeeds, copy the resulting `sdcard.img` to Desktop with the final image name based on the deployed commit hash and prepare the flash command.
3. Boot the new image on the test unit and rerun LTC/USB validation against the `.244` baseline behavior.

## Current Checkpoint (2026-05-06 - root-ram USB stabilization + rebuild in progress)

### Local repository status
- Branch: `feature/root-ram`
- HEAD: `74439b3`
- Working tree currently has uncommitted edits in three files:
  - `buildroot-external/board/clock8002-rpi5/rootfs-overlay/etc/init.d/S99alsa-ltc`
  - `buildroot-external/board/clock8002-rpi5/rootfs-overlay/opt/clock8002/alsa-ltc_pokemon.sh`
  - `buildroot-external/configs/clock8002_rpi5_defconfig.sample`
- Change intent:
  - Harden LTC service lifecycle (deterministic stop/start, pid/lock handling, single-instance guard).
  - Pin kernel tarball from 6.12.41 (`b1b490ba...`) to 6.12.47 candidate (`359f37f0...`).

### Live runtime validation completed on test unit (`root@piClock.local`)
- 25-minute restart/soak (`/tmp/alsa-soak-20260506-115902.log`) completed with `SOAK_EXIT:1`.
- Result split:
  - Process multiplication issue was eliminated during soak (`wd=1 cmd=1 ltc=1` for the run, except brief transient recovery minute 23 where `cmd/ltc=0`).
  - USB stability issue remains: `usb_set_interface failed (-110)` started growing at minute 19, ending `err_delta=52`.
- Conclusion: lifecycle fix is valid, but kernel/USB path still needs the new-image test.

### Active cm5 Buildroot build (in progress)
- Build host: `pi@10.0.0.101` (`cm5.local` fallback)
- `screen` session: `br-root-ram-74439b3-k641247-usbfix2`
- Output dir: `/home/pi/buildroot/output-root-ram-74439b3-k641247-usbfix2`
- Log: `/tmp/br-root-ram-74439b3-k641247-usbfix2.log`
- Exit marker: `/tmp/br-root-ram-74439b3-k641247-usbfix2.exit`
- Monitor:
  - `ssh pi@10.0.0.101 'screen -ls; tail -f /tmp/br-root-ram-74439b3-k641247-usbfix2.log'`
  - `ssh pi@10.0.0.101 '[ -f /tmp/br-root-ram-74439b3-k641247-usbfix2.exit ] && cat /tmp/br-root-ram-74439b3-k641247-usbfix2.exit || echo BUILD_RUNNING'`

### Next actions after build completion
1. Confirm `BR_BUILD_EXIT:0` and image at `.../images/sdcard.img`.
2. Copy image to Desktop and flash test SD card.
3. Re-run the same 25-minute restart/soak on the rebuilt image to validate whether 6.12.47 removes `usb_set_interface -110` growth.

## Current Checkpoint (2026-05-01 - root-ram parity prep)

### Local repository status
- Branch: `feature/root-ram`
- Working tree has uncommitted edits in six files:
  - `buildroot-external/configs/clock8002_rpi5_defconfig.sample`
  - `buildroot-external/board/clock8002-rpi5/config.txt`
  - `buildroot-external/board/clock8002-rpi5/cmdline.txt`
  - `buildroot-external/board/clock8002-rpi5/genimage.cfg.in`
  - `buildroot-external/board/clock8002-rpi5/linux.config`
  - `buildroot-external/board/clock8002-rpi5/post-image.sh`
- Root mode was pivoted to match the 3rd-party `.244` behavior:
  - `BR2_TARGET_ROOTFS_INITRAMFS=y` set in defconfig sample.
  - `config.txt` keeps `initramfs rootfs.cpio.gz followkernel` commented (not active).
  - `cmdline.txt` no longer forces squashfs `root=/dev/mmcblk0p2`.
  - `genimage.cfg.in` remains FAT-only (no rootfs partition).
  - `post-image.sh` no longer injects `rootfs.cpio*` into FAT payload.
  - `linux.config` keeps initrd/gzip support (`CONFIG_BLK_DEV_INITRD`, `CONFIG_RD_GZIP`).

### cm5 runtime/build capacity snapshot
- Active detached build: `screen` session `br-pre-squashfs` (output path `/home/pi/buildroot/output-pre-squashfs`).
- Under load snapshot: CPU saturated; `MemAvailable` ~5.6-5.8 GiB.
- Swap is intentionally small and currently the limiting factor:
  - `/etc/dphys-swapfile` has `CONF_SWAPSIZE=200`
  - `/var/swap` is ~200 MiB and was fully used during active build.
- User decision: hold off starting the root-ram build for now.

### Next actions when resumed
1. Start root-ram build only with fully isolated clone, output dir, screen session, and log/exit files.
2. Consider increasing cm5 swap to 1-2 GiB before any concurrent full Buildroot builds.

## Current Checkpoint (2026-05-01)

### Repository + build status
- Branch: `feature/squashfs-readonly`
- Pushed HEAD: `e9d2f9c`
- Logo packaging fix commit includes `buildroot-external/package/clock8002/clock8002.mk` install-line correction for `piclockLogo.bin`.
- Incremental Buildroot rebuild completed on cm5 with source at `/tmp/clock8002-clean-build` (HEAD `e9d2f9c`), `BR_BUILD_EXIT:0`.
- New image artifact transferred to Desktop: `/Users/jp/Desktop/piClock-e9d2f9c-sdcard.img`.
- Image checksum: `b5b3b2e86a71e06f55703c1acaa2438b24a1126f1c831cba014785257f7a107e`.
- Per user request, additional incremental rebuilds are paused until LTC issue work proceeds.

### Runtime checkpoint (target)
- During live diagnostics, repeated `sdl3-clock` exit(1) loops were traced to Limitimer serial init failing when config referenced `/dev/ttyAMA3` and that device node was absent on the running profile.
- Temporary mitigation (`limitimer-mode=off`) restored stable clock process.
- Per user direction, Limitimer was re-enabled for clock #1:
  - `source1.text=Limitimer`
  - `source1.timer=true`
  - `source1.counter=1`
  - `limitimer-mode=receive`
- After restart, `sdl3-clock` remained up (stable PID check passed).
- USB audio instability remains a separate open track (`usb_set_interface failed (-110)`, `cannot set freq 44100`).

### Defaults changed in repo
- `v4/network.ini.default` now defaults to static networking:
  - `mode=static`
  - `address=192.168.8.245`
  - `netmask=24` (CIDR for `255.255.255.0`)

## Active Investigation: Rootfs Mode vs USB Stability (2026-04-30)

### Verified facts
- piclock (.245) uses two partitions: `/dev/mmcblk0p1` mounted at `/boot` (vfat) and `/dev/mmcblk0p2` as root (`squashfs`, read-only).
- 3rd-party unit (.244) shows no squashfs or overlay mounts at runtime; root is mounted as `rootfs` (RAM-root style), with `/boot` on `/dev/mmcblk0p1` (vfat).
- USB failure signature on .245 remains `usb_set_interface failed (-110)`, reproducible with manual `arecord` and manual `alsa-ltc` while watchdog is disabled.
- Kernel page size on .245 is 4 KB at runtime even when uname suffix said `-v8-16k`.

### Config updates prepared on `feature/squashfs-readonly`
- `buildroot-external/board/clock8002-rpi5/linux.config`
  - `CONFIG_LOCALVERSION="-v8-4k"`
  - `CONFIG_IKCONFIG=y`
  - `CONFIG_IKCONFIG_PROC=y`
- `buildroot-external/configs/clock8002_rpi5_defconfig.sample`
  - Kernel tarball pin changed to `b1b490bae0bb8ad62d1f028ed0dcbbe7395a964d` (rpi 6.12.41 line).
- Note: an initial full-SHA typo caused 404 download failures on codeload; corrected and rebuild relaunched.

### Build status (cm5)
- Detached session: `3854895.br-kernel-rebuild`
- Build source: clean clone at `/tmp/clock8002-clean-build` (feature/squashfs-readonly, HEAD `fb6d5d4`) to avoid the dirty `~/clock8002` tree.
- Effective build settings logged:
  - `BR2_LINUX_KERNEL_CUSTOM_TARBALL_LOCATION="$(call github,raspberrypi,linux,b1b490bae0bb8ad62d1f028ed0dcbbe7395a964d)/linux-b1b490bae0bb8ad62d1f028ed0dcbbe7395a964d.tar.gz"`
  - `BR2_PACKAGE_CLOCK8002_SOURCE_DIR="/tmp/clock8002-clean-build/v4"`
- Monitor commands:
  - `ssh pi@cm5.local 'tail -f /tmp/br-kernel-rebuild-driver.log'`
  - `ssh pi@cm5.local 'tail -f /tmp/br-build.log'`
- Completion check:
  - `ssh pi@cm5.local 'cat /tmp/br-build.exit'`

### 3rd-party parity checklist (workboard)

| Parity axis | 3rd-party baseline | Current state on .245 | Action | Status |
|---|---|---|---|---|
| Kernel line | rpi 6.12.41 (`b1b490ba...`) | 6.12.64 currently deployed | Rebuild with 6.12.41 pin and validate uname + behavior | In progress |
| Page-size clarity | 4 KB runtime pages | 4 KB runtime pages, previous uname suffix was misleading | Keep 4 KB; publish `-v8-4k` localversion in next image | In progress |
| Runtime kernel config visibility | N/A | `/proc/config.gz` unavailable on current image | Enable IKCONFIG + IKCONFIG_PROC | In progress |
| Rootfs execution model | RAM-root style at runtime | SquashFS partition root (`/dev/mmcblk0p2`) | Run controlled A/B (squashfs vs RAM-root) | Planned |
| Partition model | single visible FAT partition | dual partitions (FAT + squashfs root) | Change only if A/B shows material USB benefit | Gated |
| Init/service shape | BusyBox init + watchdog scripts | BusyBox init + watchdog scripts | Keep constant during A/B | Aligned |
| USB symptom parity | Stable under test sequence | Reproducible `usb_set_interface -110` | Isolate root-cause variable via A/B controls | Open |

### Issue tracking recommendation
- Yes: this is worth a dedicated issue because the work spans multiple images, controlled experiments, and decision gates.
- Suggested issue title: `Parity investigation: align Buildroot image behavior with 3rd-party RAM-root reference`.
- Tracking issue created: **#43** (`Parity investigation: align Buildroot behavior with 3rd-party RAM-root reference`).
- Suggested acceptance criteria:
  - A/B results recorded with identical kernel/DT/service controls.
  - Clear decision: proceed with RAM-root migration or stop and focus kernel/USB path.
  - Final summary includes exact image provenance and observed USB outcome for each run.

### RAM-root A/B investigation plan (time-boxed)
1. Objective
  - Determine whether rootfs mode (squashfs partition vs RAM-root) materially changes USB LTC stability.
2. Hypothesis
  - Squashfs is not the primary root cause, but rootfs mode could be a secondary contributor via boot/runtime timing.
3. Decision gate
  - Continue RAM-root work only if A/B results show a clear and repeatable stability improvement.
  - If no material delta, stop RAM-root effort and focus on kernel/USB stack.
4. Controlled variables (must stay identical across A and B)
  - Kernel commit, DTB/overlays, cmdline USB-related flags, ALSA command line, service startup order, test hardware/cabling/power.
5. Experimental variable (only change)
  - Root mode:
    - A: current squashfs partition root (`/dev/mmcblk0p2`).
    - B: RAM-root model (initramfs/rootfs in memory) with persistent config still on `/boot/piclock`.
6. Test protocol per image
  - Cold boot, confirm services, run baseline idle window, run manual `arecord` test, run manual `alsa-ltc -v` test, capture dmesg and counters.
7. Metrics to record
  - `usb_set_interface`/xHCI errors, service liveness, `VmRSS`, `VmSwap`, system swap, temperature/throttle, and whether failure reproduces.
8. Success criteria
  - RAM-root must reduce or eliminate reproducible USB failures under the same procedure.
9. Run log template

```
Run ID:
Date/Time:
Image Provenance (commit, kernel SHA, root mode):
Hardware/Power/Cabling:
Commands Executed:
Observed Errors (if any):
Metrics Snapshot:
Outcome (pass/fail/inconclusive):
Next Action:
```

## Active Investigation: LTC dropouts on piclockBR (2026-04-21 → 2026-04-22)

**Symptom:** alsa-ltc logs `[gap] no LTC decoded for Nms, peak_during_gap=P` events
on piclockBR (192.168.8.246, Buildroot image c4847b2). All observed gaps show
`peak≈32750-32767` (full-scale) and duration 2-5s — classified INVALID_LTC (audio
present but biphase unreadable). Source is TouchDesigner → USB-C analog →
C-Media CM108 USB audio (`plughw:0,0`).

**Pi-side evidence (kernel dmesg, when in degraded state):**
```
usb 1-1.1: 2:1: cannot set freq 44100 to ep 0x82
usb 1-1.1: 2:0: usb_set_interface failed (-110)
```
Errors arrive on a fixed **17.92s cadence** — kernel periodically retries sample
rate renegotiation with the CM108 and times out.

**Key test result:** Stopping alsa-ltc for 60s → 0 new errors. Restarting
alsa-ltc → 0 new errors for 60s after. The 18s cadence only runs once the card
is in a degraded state; a clean `alsa-ltc` restart resets the endpoint and
clears it. Once in the bad state, each failed renegotiation can stall audio
capture 1-5s → INVALID_LTC gap.

**Escalation event (21:20 UTC):** On the pre-enhanced run, alsa-ltc hit 10 EIO
errors and exited. Watchdog restart attempts failed with `cannot set parameters
(Connection timed out)`. USB stack fully wedged: `can't set config #1, error
-110`; unbind/rebind of `1-1.1` and parent hub `1-1` failed; `lsusb` hung.
Required warm reboot to recover (matches the documented "ribbon cable" failure
mode in `/memories/repo/clock8002-stability-tests.md`).

**Reboot recovery (21:53 UTC):** Card re-enumerated cleanly as card 0, alsa-ltc
started by S99 init. Monitor re-armed.

**2026-04-22 reflash — new SD card with image `piclockBR-8234252-gerry-sdcard.img`:**
- Fresh card booted; hostname bug diagnosed (`/opt/clock8002/piclock-network.sh`
  missing exec bit in overlay → `Permission denied` at boot → network.ini
  silently skipped). Fix committed as **`58c6d17`** (`git update-index --chmod=+x`)
  and pushed. Hot-patched live unit first; Buildroot image rebuilt for 58c6d17.
- On **cold boot** of 8234252: USB xHCI HC died at 147s (hard failure) on that
  run. Subsequent cold boot is clean; unit stable at 192.168.8.246 as
  `piClockBR` (user renamed hostname in network.ini to avoid mDNS collision
  with another `piclock.local` on LAN).
- USB audio now on **card 1** (HDMI takes card 0/2 on this image); `1-1.1`
  still. alsa-ltc PCM RUNNING, hw_ptr advancing, no errors.

**Monitoring in place (ltcmon v3, 2026-04-22 ~15:59 UTC on new image):**
- Script: `/root/ltcmon.sh` on piclockBR (repo copy: `tools/ltcmon.sh`). Rebuilt
  from the session-memory spec — original file did not survive the new SD card.
- Log: `/tmp/ltcmon.log`. Auto-detects USB audio card + usbdev path, so it
  works regardless of card index.
- Watches:
  - `tail -F /tmp/alsa-ltc.log` (foreground) — classifies `[gap]` events
    (SILENCE / INVALID_LTC / PARTIAL), `[APP_ERR]`. **NOTE:** on this image the
    S99 init does not redirect alsa-ltc stdout, so `/tmp/alsa-ltc.log` is empty
    until alsa-ltc is restarted under a redirect. Gap classification dormant
    until then; all other signals still captured.
  - `dmesg` cursor-based poll @5s — KERN/FREQ_FAIL, IFACE_FAIL, XRUN,
    DISCONNECT, RESET, HC_DIED, DMA_PAUSE
  - `/proc/asound/card<N>/pcm0c/sub0/status` @1s — PCM state transitions
    (PCM/STATE), hw_ptr stall detection (PCM/STALL)
  - `/sys/bus/usb/devices/1-1.1/urbnum` @1s — URB stall detection (URB/STALL)
    when alsa-ltc is running
  - `/sys/bus/usb/devices/1-1.1/power/runtime_status` @1s —
    USB/SUSPEND / USB/RESUME transitions (definitive autosuspend fingerprint)
  - 10s HEALTH snapshot: pid, state, pcm, hw_ptr, urbn, rs, susp_dms (ms
    suspended this interval), xhci_d (xHCI interrupt delta), load, temp,
    cumulative freq/iface error counts

**Baseline (healthy):** hw_ptr +442K/10s, urbn +10K/10s, xhci_d ≈10022,
rs=active, susp_dms=0, no errors.

**Autosuspend status:** `autosuspend=2s` is configured on the CM108 endpoint,
but `runtime_suspended_time=0` since boot — device has never suspended while
alsa-ltc is actively reading. Autosuspend could only fire during an
error-recovery window when alsa-ltc momentarily pauses URB submission. The
new USB/SUSPEND tag will definitively show whether autosuspend is the
trigger when the bug next occurs.

**Commands:**
```bash
# Skip heartbeats, see events
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa root@192.168.8.246 \
  'grep -v heartbeat /tmp/ltcmon.log | tail -40'

# Restart monitor (hard-kills strays, clears log)
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa root@192.168.8.246 '
for p in $(ps | grep ltcmon | grep -v grep | awk "{print \$1}"); do kill -9 $p; done
rm -f /tmp/ltcmon.log
nohup /root/ltcmon.sh > /tmp/ltcmon.log 2>&1 &'
```

**Open questions / next steps:**
- Does autosuspend trigger the degraded state? (Monitor v3 will answer: if
  `[USB/SUSPEND]` precedes `[KERN/FREQ_FAIL]` cluster → confirmed; if
  `rs=active` throughout → ruled out)
- Is PCM stall or URB stall the first symptom? (differentiates ALSA buffer
  starve vs. USB URB submission freeze)
- Candidate mitigations (NOT yet attempted — change-control gate):
  - Disable USB autosuspend on the CM108 at runtime:
    `echo -1 > /sys/bus/usb/devices/1-1.1/power/autosuspend` +
    `echo on > /sys/bus/usb/devices/1-1.1/power/control`
  - Watchdog-level PCM reopen on repeated gap events
  - Different USB port / different audio adapter
- Repo rule: no code changes until user explicitly approves.

**Session note saved:** `/memories/session/ltc-dropout-investigation.md`

**Production image policy (2026-04-22):** Diagnostics (ltcmon, verbose
logging) stay OUT of production images — deploy on demand from `tools/`.
See `/memories/repo/clock8002-production-image-policy.md`.

**Source3 "Playback" not displaying (2026-04-22) — resolved:** Mitti was
still configured to send OSC to the previous unit's IP. IP changed with the
new SD card. Not a code/config bug on the clock side. Fix: update Mitti OSC
destination to `192.168.8.246:1245`.

---

## feature/squashfs-readonly Branch (2026-04-29)
### Goal
Convert the Buildroot rootfs from ext4 (read-write) to SquashFS (read-only) with
tmpfs covering all runtime write paths. All writes go to RAM; SD card rootfs
partition is never written after first flash.

**Current status (2026-04-29): Phase 2 config changes committed as `f84f49e`.
Build complete (`BR_BUILD_EXIT:0`). Image `piClock-f84f49e-sdcard.img` (343 MB) on Desktop.
SHA-256: `aa5b01b9707fbeab46bdc0a087f6cd170971681ac507491604516f8c2e80da6b`.
cm5 restored to `master`. SD card detected at `/dev/disk6`. Ready to flash and run Phase 9 tests.
See Issue #41 for full 10-phase plan.**

### Commits on this branch (HEAD: `f84f49e`)
- `be4198e`: buildroot: /root tmpfs — move launcher scripts to /opt, add S02setup-root
- `58e9b0a`: squashfs-readonly: fix /root tmpfs permissions for sshd StrictModes
- `f351372`: buildroot: fix S02setup-root mkdir/chmod on same line
- `45d637f`: buildroot: fix network boot — move /root tmpfs from overlay fstab to post-build.sh
- `16a99a7`: buildroot: fix piclockLogo.bin not found on OLED splash (root tmpfs)
- `f4679e0`: clock8002.mk: remove install to $(TARGET_DIR)/root (build host path issue)
- `f84f49e`: buildroot: squashfs Phase 2 — defconfig, kernel, genimage, SSH key persistence

### Phase 1 write-path audit (2026-04-29) — complete
All runtime write paths classified. The only real problem was SSH host keys
(would regenerate on every reboot with overlayfs). Fix: `S49sshd-keys` init
script persists keys to `/boot/piclock/ssh/` on first gen, restores each boot.

All other write paths (NM connections, /var/lib, /etc shadow/passwd/group,
/var/lib/seedrng, /crond.reboot) are acceptable on overlayfs upper tmpfs.

### Phase 2 config changes applied (2026-04-29)
- `defconfig.sample`: ext2 → squashfs, no REMOUNT_RW
- `linux.config`: +CONFIG_SQUASHFS, +CONFIG_SQUASHFS_ZSTD, +CONFIG_OVERLAY_FS
- `genimage.cfg.in`: rootfs.ext4 → rootfs.squashfs
- `cmdline.txt`: rootfstype=ext4 → rootfstype=squashfs
- `post-build.sh`: +S49sshd-keys chmod, HostKey redirect, /var/lib tmpfs + NM connections tmpfs fstab entries
- `S49sshd-keys`: new init script — generates/restores SSH host keys from FAT

### Write-path tmpfs coverage (post Phase 2)
| Path | Mechanism |
|---|---|
| `/root` | separate tmpfs (fstab, pre-existing) |
| `/tmp`, `/run` | separate tmpfs (Buildroot default) |
| `/var/lib` | new tmpfs (post-build.sh fstab) |
| `/etc/NetworkManager/system-connections` | new tmpfs (post-build.sh fstab) |
| `/etc/ssh/ssh_host_*` | S49sshd-keys copies to `/root/ssh/` before sshd |
| `/etc/shadow`, `/etc/passwd`, `/etc/group`, etc. | overlayfs upper tmpfs (squashfs lower is ro) |
| `/boot/piclock/clock.ini` | FAT (stays writable) |

### Buildroot image status
- ✅ OLED logo fix live-tested and confirmed working on `piClock-f4679e0-sdcard.img`
- ✅ Network fixed (`45d637f`), confirmed working
- ✅ First squashfs build complete (`f84f49e`): `BR_BUILD_EXIT:0`, image `piClock-f84f49e-sdcard.img` on Desktop
- ✅ machine-id fix committed (`1fe6357`): S12machine-id init script generates/persists machine-id to FAT
- ✅ machine-id build complete — image `piClock-1fe6357-sdcard.img` (343 MB) on Desktop, flashed and live

### Phase 9 clean-boot test results (2026-04-29) — image `piClock-1fe6357-sdcard.img`
Unit: `root@piClock.local` / `192.168.8.246`

| Check | Result |
|---|---|
| rootfs is squashfs (ro) | ✅ |
| `/etc` fully read-only (not even writable by root) | ✅ stronger than overlayfs approach |
| tmpfs: `/tmp`, `/run`, `/root`, `/var/lib`, `/etc/NetworkManager/system-connections` | ✅ |
| FAT persistence across reboot (`/boot/piclock/phase9-test`) | ✅ |
| machine-id stable across reboot | ✅ `e8d290a3...` same before and after |
| SSH host key unchanged across reboot | ✅ fingerprint `SHA256:YKeSTc5J...` |
| sdl3-clock running after reboot | ✅ |
| alsa-ltc running after reboot | ✅ |
| oled_daemon running after reboot | ✅ |
| network.ini static IP + ap_enabled | ✅ user-verified |
| Config symlinks resolve to FAT | ✅ `clock.ini` → `/boot/piclock/clock.ini`, `oled.ini` → `/boot/piclock/oled.ini` |
| Web UI config save to FAT | ✅ `ToDHideSeconds=false` written to `/boot/piclock/clock.ini` |
| authorized_keys provisioning | ✅ `/boot/piclock/authorized_keys` → `/root/.ssh/authorized_keys` at boot |
| clock log writes (`/root/.config/clock-8001/clock.log`) | ✅ 16 KB, actively written to tmpfs |

**Key finding:** `/etc` is directly on squashfs — no overlayfs upper layer. This is stronger isolation than the S02overlayfs approach originally planned in Issue #41. `touch /etc/...` returns `Read-only file system` even as root.

**Config symlink note:** `/opt/clock8002/clock.ini` does not exist on this image. The clock binary uses `~/.config/clock-8001/clock.ini`, which S02setup-root recreates as a symlink to `/boot/piclock/clock.ini` at every boot.

### Status
- ✅ Phase 1 audit complete
- ✅ Phase 2 config changes committed (`f84f49e`)
- ✅ Phase 2 build complete
- ✅ machine-id fix committed and built (`1fe6357`)
- ✅ Phase 9 clean-boot test — **ALL 14 CHECKS PASSED** (2026-04-29)
- ✅ Confirmatory tmpfs-reset test — **PASSED** (2026-04-29): `ToDHideSeconds=false` persisted in FAT after reboot
- ✅ 12h mode — **VERIFIED WORKING** (2026-04-30): was broken by build race (background `make &` + immediate branch switch); fixed with synchronous build. CRITICAL rule added to copilot-instructions.md.
- ✅ AM/PM label rendering — working with `textR.W -= textR.H * 0.6` reservation (`aa09bb8`)
- ⏳ draw4TextClocks size tweak — H 240→210, spacing unchanged at 265 (`f669f5b`) — flashed, awaiting visual check
- ⏳ Pending items (see below) — none are functional blockers

### To Do List
1. ✅ **Confirmatory test** — `ToDHideSeconds=false` written via web UI, survived reboot in FAT clock.ini; in-memory state reset on reboot. Tmpfs reset works as designed.
2. **Dev-deploy behaviour note** — `/opt/clock8002` is squashfs (ro); `cp sdl3-clock /opt/...` will fail on the live unit unlike ext4. Dev deploys require reflash. Document this change in Issue #41.
3. **Issue #41 update** — record the per-path tmpfs approach as the resolved implementation (vs `S02overlayfs` full overlayfs originally planned), and close out the checklist.
4. **tmpfs simplification follow-up** — evaluate removing the `/root` tmpfs + `S02setup-root` layer after the reproducible baseline is settled; the current conclusion is that it is redundant, but it stays as a cleanup task for later.
5. **Issue #44 queued image validation** — run queued-change image build/flash on `.245`, then validate: power button shutdown path, machine-id persistence (`/boot/.piclock-machine-id`) across reboots, network.ini modes (`dhcp`/`static`/`dual`) + AP options, and UART1 (`/dev/ttyAMA1`) cue path.
6. **Issue #44 closeout notes** — record validation outcomes in Issue #44 and summarize final state in HANDOFF.

### Next steps
1. ✅ ~~Run confirmatory tmpfs-reset test~~ — DONE
2. ⏳ Update Issue #41 with per-path tmpfs rationale + dev-deploy note
3. ⏳ Merge `feature/squashfs-readonly` to `master` and cut next release

### What changed vs master
- `rootfs-overlay/etc/fstab` — deleted entirely. `post-build.sh` appends
  `tmpfs /root tmpfs mode=0700,noatime 0 0` to preserve Buildroot-generated fstab.
- `S02setup-root` — new init script (runs at S02, before S03/S04): copies 5 launcher
  scripts from `/opt/clock8002/` to `/root/` AND copies `piclockLogo.bin` from
  `/opt/clock8002/` to `/root/` (so oled_daemon.py can find it). Recreates
  `/root/.config/clock-8001/clock.ini → /boot/piclock/clock.ini` symlink.
- All 5 launcher scripts moved from `rootfs-overlay/root/` to `rootfs-overlay/opt/clock8002/`
  so they are accessible from `/opt` (not hidden by the tmpfs mount)
- `S01power-button` → `S04power-button` — renamed so it runs after S02 populates `/root/`
- `post-build.sh` — chmod lists updated to match new names/paths; appends both
  `/boot` FAT entry and `/root` tmpfs entry to fstab
- `clock8002.mk` — installs `piclockLogo.bin` to `/opt/clock8002/` (not `/root/`,
  since `/root` is tmpfs at runtime). S02setup-root copies to `/root/` at boot.
- **Nothing network-related changed** — S43, S45, NM configs, network.ini,
  piclock-network.sh are all identical to master

### Boot sequence
```
mount -a (fstab)  → /root tmpfs mounted drwx------ (mode=0700)
S02setup-root     → chmod 700 /root; copies scripts; creates clock.ini symlink
S03copy_*         → /boot overrides if present
S04power-button   → /root/power-button.sh now exists, launches cleanly
S43piclock-network-prep → DHCP mode: removes any stale static NM config; exits
S45piclock-network      → waits for NM; runs piclock-network.sh (WiFi AP + hostname)
```

### Key bugs found and fixed
1. **sshd StrictModes rejection**: default tmpfs mount mode is `drwxrwxrwt`
   (world-writable). sshd refuses key auth when home dir is world-writable.
   Fix: `mode=0700` in fstab. Belt-and-suspenders: `chmod 700 /root` in S02.
2. **S02setup-root mkdir/chmod on same line**: tab-separated `mkdir -p /root`
   and `chmod 700 /root` on one line — shell treats them as one command with
   extra args. chmod never ran. Fixed at `f351372`.

### Live test result (2026-04-26) — manual apply on v1.3.5
- v1.3.5 reflashed to piClock (192.168.8.245)
- Feature changes manually applied over password SSH
- Rebooted — clean boot, key auth working, all services up:
  `power-button`, `oled_daemon`, `alsa-ltc`, `sdl3-clock`
- `/root` permissions: `drwx------`, tmpfs mounted `mode=700`
- clock.ini symlink recreated at `/root/.config/clock-8001/clock.ini`
- SSH authorized_keys in `/boot/piclock/authorized_keys`

### Buildroot image status
- Dev builds from `af37c54` and `f351372` **both failed clean-boot** — unit
  unreachable on network. Root cause: unknown.
- The manual-apply approach (v1.3.5 base + feature changes over SSH) works
  correctly and has passed two clean-boot tests.
- **piClock (192.168.8.245) is currently running v1.3.5 + manually applied
  feature changes** (commit `f351372` equivalent). SSH key auth working.

### Network failure — ROOT CAUSE FOUND AND FIXED (commit `45d637f`)

**Root cause:** `rootfs-overlay/etc/fstab` was a complete file replacement.
Buildroot overlays replace the entire target file — not append to it. The
overlay contained only one line (`tmpfs /root tmpfs mode=0700,noatime 0 0`),
which wiped the Buildroot-generated entries for `/proc`, `/sys`, `/run`,
`/tmp`, `/dev/pts`, and `/dev/shm` from every baked image. NetworkManager
requires sysfs to enumerate network interfaces — without `/sys` mounted,
`eth0` was invisible and the unit had no IP on every clean flash.

The running card was unaffected because it was built from `master` (which has
no overlay fstab), and the `/root tmpfs` line was manually appended on top of
the existing full fstab.

**Fix (`45d637f`):**
- Deleted `rootfs-overlay/etc/fstab` entirely
- Updated `post-build.sh` to append `tmpfs /root mode=0700,noatime` alongside
  the existing `/boot` append — Buildroot-generated fstab is now preserved intact

### Build in progress
- Full rebuild (`make clean` + `make`) started on cm5 at 2026-04-26 ~20:12 UTC
- Running in `screen -S piclock-build` on cm5 — survives disconnection
- Script: `/tmp/piclock-build.sh` on cm5
- Log: `/tmp/br-build.log` on cm5 — tail with:
  ```bash
  ssh pi@cm5.local 'tail -f /tmp/br-build.log'
  ```
- Completion marker: `BUILD_COMPLETE` in log
- Script auto-restores cm5 to `master` on completion
- Build includes `BR2_PICLOCKKEY` — dev image with SSH key baked in

### Status
- ✅ Root cause identified and fixed (`45d637f`)
- ✅ Code committed and pushed to `feature/squashfs-readonly` at `45d637f`
- ✅ Live-tested (manual apply on v1.3.5 base) — all services up, key auth working
- ⏳ Clean-build image in progress on cm5 (full `make clean`)
- ❌ Clean-boot test of baked image — pending build completion

### Next steps
1. Wait for `BUILD_COMPLETE` in `/tmp/br-build.log`
2. Verify cm5 restored to master: `ssh pi@cm5.local 'cd ~/clock8002 && git branch --show-current'`
3. Transfer image: `scp pi@cm5.local:~/buildroot/output/images/sdcard.img ~/Desktop/piClock-45d637f-sdcard.img`
4. Flash and do clean-boot test — verify network, key auth, all services, `/root` tmpfs
5. If clean-boot passes: merge to master and cut v1.4.0

---

## Current State

- Repository: jpkelly/clock8002
- Active release line: v1.x
- **Latest release: v1.3.5** (2026-04-26) — `master` branch (Buildroot/SDL3). Commit: `5477158`
- Latest Trixie release: **v1.3.1** — archived on `trixie` branch
- **Active branch: `master`** (Buildroot) — branch rename complete 2026-04-26
- **feature/squashfs-readonly**: HEAD `1fe6357` — Phase 9 clean-boot test **COMPLETE** (all 14 checks passed, 2026-04-29). Image `piClock-1fe6357-sdcard.img` live at `192.168.8.246`. Remaining: Issue #41 phases 3–8/10.
- **Active monitoring**: ltcmon + alsa-ltc logging live on piClock (192.168.8.245). alsa-ltc stable; watching for USB/LTC dropout recurrence.
- **piClock test unit** (192.168.8.245): flashed `piClock-f4679e0-sdcard.img`. OLED logo confirmed working. Network (static) working. All services up.
- Recent UI fixes on buildroot branch (2026-04-22), deployed to piclockBR:
  - `013be8d`: cue fullscreen clipping fix (text4 uses per-face logical size, vertical swap)
  - `e585ba0`: web-config save page refresh delay 1s → 3s (fix stale view race)
  - `70d0695`: 3-line text clock wider row gaps (stride 365→380, heights reverted to 300/100) — supersedes `06cc775` which grew row heights
  - `8001c05`: font fields in web config become `<select>` dropdowns; pass `--font-path=/opt/clock8002` in service/install.sh/clock_cmd.sh overlay
  - `5a18694`: preserve FontPath across config save; extract font walk into `collectFonts()` helper (fixes dropdown going empty after first save)
  - All five UI fixes above **visually verified on piclockBR display 2026-04-22**: dropdown populates and repopulates after save; saved NumberFont renders on text faces; 3-line spacing correct; save redirect timing correct; cue fullscreen no longer clipped.
- **Scope:** SDL3 + font dropdown fixes are `buildroot` branch only. Trixie variants (master) still on SDL2 `sdl-clock`; will inherit these on next buildroot → master merge + v1.x release.
- Reboot + sdl3-clock deploy lesson captured in memory (`/memories/repo/clock8002-sdl3-clock-deploy-rules.md`): `S99clock stop` doesn't kill the pokemon watchdog; multiple deploy cycles stack watchdogs that race for port :80; sdl3-clock can hang in D-state (unkillable) — reboot first.
  - OLED logo + stats: **working at boot**
  - sdl3-clock HDMI: **working at boot**
  - WiFi AP (piClock-ap): **working**
  - Power button shutdown: **working at boot** (stable `/dev/input/by-path/` symlink)
  - Network config from `network.ini`: **working** (static IP, hostname, AP — all verified after reboot)
  - alsa-ltc: **stable** — 0 USB errors, 0 restarts (PID stable 15+ min), LTC decoding to display confirmed
  - `authorized_keys` from `/boot/piclock/authorized_keys`: **tested and working** — key-based SSH confirmed
  - Static IP from `network.ini`: **working on first boot** — `S43piclock-network-prep` writes NM connection file before NM starts, no DHCP race
  - Build host: pi@cm5.local (10.0.0.101)
- **3rd party reference unit (192.168.8.246 / 10.0.0.131)**: `root` / `clockworkadmin`. BusyBox init. alsa-ltc fixed (was using `-` for device, now uses `plughw:${ALSA_CARD:-2},0`). LTC rolling on display. 0 USB errors.
- **2GB Pi 5 board #1** (piclockTG.local): fresh Trixie 6.12.47, v1.3.1 gerry. EEPROM downgraded to 2025-05-08. Monitor running.
- **2GB Pi 5 board #2** (piclockTD.local): fresh Trixie 6.12.47, v1.3.1 default. Stable. EEPROM 2025-05-08.
- `buildroot-prototype` branch: fully merged into master

## Second Display Work (2026-04-25, buildroot branch)

### Commits
- `fc4c6eb`: second-display: mirror mode + PerfectCue icon mode for Buildroot
- `af37c54`: second-display: fix freeze on config save with second display active

### What was implemented
- **Mirror mode** (`CueSecondDisplay=false`, default): real-time DRM dumb buffer copy of SDL3 renderer output to spare HDMI connector. ABGR8888→XRGB8888 byte-swap for correct colours. Letterbox/pillarbox scaling for mismatched resolutions.
- **Cue icon mode** (`CueSecondDisplay=true`): fullscreen PerfectCue icons (right/left/blank) on second HDMI, toggled via web config. Black between cues.
- New files: `drm_mirror_linux.go`, `drm_mirror_other.go`, `drm_cue_linux.go`, `drm_cue_other.go`, `second_display_probe.go`

### Key bugs found and fixed
1. **sysfs false-positive HDMI detection**: `/sys/class/drm/card1-HDMI-A-2/status` always returns `connected` on Pi 5. Old code called `DROP_MASTER` on SDL's DRM fd → SDL lost display → hung system (bricked unit 3 times). Fix: use DRM ioctl (`findSpareHDMIConnector`) which only returns a connector if physically connected.
2. **Unsafe pointer arithmetic**: `surface + 5*8` assumed SDL3 Surface struct layout. Fix: use `surface.Pixels()`.
3. **R/B colour swap**: SDL3 `ReadPixels` returns ABGR8888 (R,G,B,A in memory) but DRM XRGB8888 expects B,G,R,X. Fix: detect format and swap bytes 0 and 2.
4. **S99clock stop() was a no-op**: `clock_pokemon.sh stop()` contained only `true`. Fix: kill by PID file + process name. `S99clock start` now saves PID.
5. **DRM freeze on config save**: `probeSecondDisplayOutput()` called `initDRMMirror()` on every config reload (not just first-time init). Second call did DROP_MASTER/SET_MASTER again → frozen display. Fix: guard with `isDRMMirrorActive()` — in-place mode switch, no DRM teardown.

### Soak test status (2026-04-25 ~19:35 PDT / 2026-04-26 02:35 UTC)
- Unit: 192.168.8.245, commit `af37c54`, ~5 min uptime at start
- 0h baseline: RSS=109,560 kB, VmSwap=0, RAMfree=~688 MB, swap=0
- 36 samples to date: RSS flat (normal jitter ±400 kB), both processes alive, no swap
- Monitor: on-device `/tmp/soak.sh` (nohup), logging to `/tmp/soak.log`
- **Decision gate**: hold release tag until 24h checkpoint shows flat RSS/swap

### Release build status
- Full `make clean && make` running on cm5 in screen session `br-release`
- Log: `/tmp/br-release.log` on cm5
- Check: `ssh pi@cm5.local 'grep BR_EXIT /tmp/br-release.log'` (empty until done)
- Monitor: `ssh pi@cm5.local 'tail -5 /tmp/br-release.log'`
- This will be the release image (no SSH key baked in)

### After soak test passes
1. Check soak log: `ssh root@192.168.8.245 'tail -20 /tmp/soak.log'`
2. Update HANDOFF.md with final soak metrics
3. Cut new version tag (v1.x — check last tag with `git tag | sort -V | tail -5`)
4. Both default and Gerry release tarballs required (per release policy)
5. Update README quick-install commands to new version
6. Transfer release image to desktop and provide flash command

---

## SDL3 Migration Status (branch: buildroot)

### Current state (2026-04-20)
- Branch renamed `feature/sdl3-migration` → `buildroot`; tagged **v1.3.2**
- Branch HEAD: **`c1fc28a`** (pre-tag) → tagged commit includes sdl3-clock feature parity work
- All boot issues fixed: logo, clock, WiFi AP, power button, network config
- **Test unit** (192.168.8.246): live-deployed, all features confirmed working from cold boot
- `piclockBR-c1fc28a-sdcard.img` built on cm5, transferred to Mac Desktop

### Recent commits (session 2026-04-20 — power button + network + BusyBox compat)
- `06d3715`: buildroot: add power button shutdown handler for BusyBox init
- `ab7e80e`: power-button: use nohup to detach handler from init session
- `720d883`: power-button: wait for /dev/input/event0 before reading events
- `855151c`: HANDOFF: add power button, document BusyBox boot-timing pattern
- `c1fc28a`: power-button: use stable by-path symlink instead of hardcoded event0
- `d62f6dd`: piclock-network: BusyBox compatibility for Buildroot

### Prior commits (session 2026-04-19/20 — Buildroot boot fixes)
- `934e43a`: oled: fix SyntaxWarning on regex string literal
- `c05d2c7`: oled: fix INI_PATH to find clock.ini on Buildroot
- `ba63500`: oled: blank display on SIGTERM/SIGINT for clean shutdown
- `ff229c0`: clock_pokemon: unbind fbcon before launching sdl3-clock
- `a9feaf7`: buildroot: add fbcon=map:10 to cmdline to prevent fbcon holding DRM master
- `5609f6d`: fix boot: use absolute paths for logo, export HOME=/root in watchdog

### BusyBox init boot-timing pattern (unified root cause)
All boot failures on BusyBox init share the same root cause: **rcS runs S## scripts
before devices, environment, or services are ready**. Unlike systemd (device units,
After= dependencies), BusyBox rcS is a sequential loop with no dependency tracking.

| Issue | Root Cause | Fix |
|-------|-----------|-----|
| OLED logo black at boot | `HOME` unset → `~/piclockLogo.bin` → `/piclockLogo.bin` | Absolute path `/root/piclockLogo.bin` with fallback |
| sdl3-clock crash-loop at boot | `HOME` unset → fonts not found; fbcon held DRM master | `export HOME=/root` + `fbcon=map:10` + device wait |
| Power button not working at boot | `/dev/input/event0` not yet created by eudev | Wait loop (up to 30s) for device node |
| OLED blank user/pass/port | `INI_PATH` hardcoded to Trixie path | Added `/boot/piclock/clock.ini` Buildroot fallback |
| WiFi AP not broadcasting | `piclock-network.sh` ran before wlan0 ready | 3-retry loop with 5s delay |
| SyntaxWarning in oled_daemon | Unescaped `\.` in non-raw string | Changed to `r"..."` |

**Fix pattern**: Always poll/wait for the required resource (device node, env var,
network interface) before using it. Never assume it exists at S## script time.

### Prior commits (session 2026-04-18)
- `96f45ac`: fix ifeq-in-recipe; enable WiFi AP in gerry network.ini
- `8bc6c5e`: add S98oled BusyBox init script for oled_daemon
- `8ad295e`: install authorized_keys from /boot/piclock/ at boot
- `af98d5e`: fix OLED (i2c-dev module, sdl3-clock path) and WiFi AP (S45piclock-network)
- `2b81719`: add CONFIG_I2C_DEV=m; wait for NM before configuring network

### USB audio root cause — BusyBox init / pokemon watchdog approach
- **3rd party unit** (10.0.0.131): kernel `6.12.41-v8` commit `590178d5`, BusyBox inittab — **zero retire_capture_urb errors**
- **Our prior approach**: systemd ExecStartPre with `rmmod`+`modprobe snd_usb_audio` on every alsa-ltc restart — triggers RP1 xHCI endpoint lock → EIO reads → `usb_set_interface -110`
- **New approach** (commit `e247328`): load `snd_usb_audio` once at boot via S11modules, then restart only the userspace `alsa-ltc` process (pokemon watchdog) — never cycles the driver
- **Page size**: Both our old build and 3rd party are 4K pages. `-16k` in kernel CONFIG_LOCALVERSION is misleading — not the issue. DO NOT revisit this.

### Changes in `e247328`
- `defconfig.sample`: kernel commit `359f37f0` → `590178d58b730e981099fdcb405053a000e79820`, `BR2_INIT_SYSTEMD` → `BR2_INIT_BUSYBOX`
- `clock8002.mk`: removed all systemd service file installs
- New rootfs overlay `etc/init.d/`: S11modules, S03copy_alsa-ltc_files, S03copy_clock_files, S99alsa-ltc, S99clock
- New rootfs overlay `root/`: alsa-ltc_pokemon.sh, alsa-ltc_cmd.sh, clock_pokemon.sh, clock_cmd.sh
- `post-build.sh`: fully rewritten — removed all systemd-specific logic
- Deleted orphaned: `etc/systemd/system/piclock-network.service`, `usr/lib/systemd/system/alsa-ltc.service`, `etc/udev/rules.d/99-usb-audio-power.rules`

### What was ported to sdl3-clock (vs sdl-clock master)
- `AppVersion` field + config stamping (data.go, config.ini.go, main.go)
- `CueSecondDisplay` option skeleton (option exists, no DRM implementation)
- `clock.ini.default`: removed hardcoded `app-version`, changed `Face=quad` → `Face=text`

### Features NOT yet ported to sdl3-clock
| # | Feature | Status |
|---|---------|--------|
| 1 | **Quad face** — 4-source text clock | Not ported |
| 2 | **Dual face** — 2-source text clock | Not ported |
| 3 | **DRM mirror** — second HDMI via KMS/DRM dumb buffer | ✅ Done (`fc4c6eb`) |
| 4 | **PerfectCue icon on HDMI-2** — cue icon mode | ✅ Done (`fc4c6eb`) |
| 5 | **Configurable PerfectCue geometry** — cue-pos-x/y, cue-size | Not ported |
| 6 | **Sync Time web API** — `/api/settime` + RTC sync | Not ported |
| 7 | **Cue test API** — `/api/cue` | Not ported |
| 8 | **Atomic config import** — temp+validate+rename | Not ported |
| 9 | **SDL resource leak fixes** — destroyTextClock/Audio on hot-reload | Not ported |
| 10 | **Row 4 color/alpha** — configurable color + alpha for text clock row 4 | Not ported |
| 11 | **Network-aware info overlay** — shows eth0/wlan0 IPs and WiFi AP SSID on 'I' overlay | Not ported |
| 12 | **Config version display** — "Loaded config version" in web UI + `/export` download link | Not ported |
| 13 | **Symlink-safe config path** — `resolvedConfigPath()` resolves symlinks before import/save | Not ported |
| 14 | **Web UI teal theme** — color scheme changed from pink/magenta to teal (`#006D88`); tab, heading, and form colors updated | Not ported |

### Session 2026-04-20 — alsa-ltc stability + authorized_keys test

**alsa-ltc stability confirmed** (both our Buildroot unit and 3rd party):
- Root cause of prior crash loops: `alsa-ltc_cmd.sh` used `-` (auto-detect) which fails to open CM108 on BusyBox kernels, causing a continuous crash/restart loop that floods VL805 with `usb_set_interface -110` errors
- Our fix (already in image via pokemon watchdog): `ALSA_CARD` detected dynamically from `/proc/asound/cards` → used as `plughw:${ALSA_CARD:-2},0`
- Applied same fix to 3rd party unit (`/root/alsa-ltc_cmd.sh` + `/boot/piclock/authorized_keys` installed)
- Result on both units: 0 USB errors, stable PID over 15+ min, LTC decoding confirmed on display

**authorized_keys feature tested**:
- `S03copy_clock_files` already implements: copies `/boot/piclock/authorized_keys` → `/root/.ssh/authorized_keys` (700/600 perms) at every boot
- Tested live: wrote key to `/boot/piclock/authorized_keys`, manually triggered copy, passwordless SSH confirmed working
- FAT boot partition makes this accessible from any OS without ext4 support

### Still pending
- Remaining sdl3-clock feature gap items (see table above — items 1–14)
- No blocking issues on the Buildroot platform
- Port features 1–14 from table above (DRM mirror/cue most complex)

### Branch rule
- `v4/` Trixie files are off-limits for Buildroot-only fixes on this branch
- Buildroot-specific changes go in `buildroot-external/` only

### Key credentials / commands
- piclockBR SSH: `sshpass -p 'clockworkadmin' ssh -o StrictHostKeyChecking=no root@10.0.0.128`
- 3rd party unit: `sshpass -p 'clockworkadmin' ssh -o StrictHostKeyChecking=no root@10.0.0.131`
- Monitor build: `ssh pi@cm5.local 'tail -f /tmp/br-build.log'`
- Binary deploy (no reflash): `scp pi@cm5.local:~/buildroot/output/target/opt/clock8002/sdl3-clock root@10.0.0.128:/opt/clock8002/sdl3-clock`
- Image transfer after build: `scp pi@cm5.local:~/buildroot/output/images/sdcard.img ~/Desktop/piclockBR-ab5139d-sdcard.img`

---

### Active Investigation: VL805 xHCI crash on piclockTG (2026-04-14)

**Symptom:** VL805 xHCI controller dies with `Host System Error` / `HC died` at random intervals (~55min, ~6h50m). USB hub and audio device disappear. alsa-ltc crash-loops. Requires hard power cycle to recover.

**Root cause finding:** Two differentiators from piclockTD (stable):
1. `[4.9s] Undervoltage detected!` in TG dmesg on every boot — TD has none
2. TG was running EEPROM firmware **2025-12-08** (`2226a853`); TD has **2025-05-08** (`69471177`)

**Key observation:** After downgrading TG EEPROM to 2025-05-08, the undervoltage event **disappeared** on the same PSU and hardware. This implicates the newer firmware as the primary driver — it likely draws more current during VL805 init (different ASPM / power sequencing), pushing the PSU below threshold and triggering the PCIe bus fault.

**Action taken (2026-04-14):**
- Hard power cycled TG to recover USB
- Downgraded TG EEPROM: `sudo rpi-eeprom-update -d -f /lib/firmware/raspberrypi/bootloader-2712/stable/pieeprom-2025-05-08.bin`
- Rebooted — firmware confirmed `69471177`, no undervoltage, throttle `0x0`, USB healthy
- 1-minute monitor running (`~/monitor.sh` → `~/monitor.log`)

**Decision gates:**
- 6h: check TG monitor log for usb-hub=0 / usb-audio=0 / usb-errors
- 24h: final determination
- If TG survives → firmware was the cause → document and close
- If TG still fails → PSU is the cause → swap PSU between TD and TG

**Monitor restart (after any reboot):** `nohup bash /home/pi/monitor.sh > /dev/null 2>&1 & echo "PID=$!"`

**Bug (`install.sh`):** `~/.config/clock-8001/` was created as root when running `sudo bash install.sh` on a fresh system — `sdl-clock` (running as `User=pi`) could not open the log file and exited immediately on every restart attempt. Only affected fresh installs where the directory didn't already exist.

**Fix (`16302b4`):** Added `chown "${INSTALL_USER}:${INSTALL_USER}" "${CONFIG_DIR}"` after `mkdir -p`. One line.

**Process note:** v1.3.0 was released without catching this because soak test units had the directory from a prior deploy. Fresh-install smoke test step added to RELEASING.md (step 7) to prevent recurrence.

### VL805 xHCI crash — EEPROM firmware implicated (Issue #39, re-opened 2026-04-14)

**Prior resolution (2026-04-12):** Reflashed piclockTG with fresh Trixie — appeared stable. Both boards soaking.

**Re-occurrence (2026-04-14):** piclockTG failed again — VL805 xHCI `HC died` at ~55min. Root cause investigation found new differentiator: TG had EEPROM firmware **2025-12-08** (`2226a853`) vs TD's **2025-05-08** (`69471177`). Pre-downgrade: TG showed `Undervoltage detected!` at boot on same PSU. Post-downgrade: undervoltage gone, throttle `0x0`. EEPROM downgraded to 2025-05-08 and soak restarted. See "Active Investigation" section above.

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
12. `second_display_probe.go` — cue mode now uses `initDRMMirror()` + `updateCueDRMBuffer(off)`. `syncSecondDisplayCueDisplay()` calls `updateCueDRMBuffer(desired)` — renders icon via `renderCueVisualImage()` and writes XRGB8888 directly into the dumb buffer. No `fbi` binary or `/dev/fb0` required. Web GUI toggle (PerfectCue section) switches modes live without restart. Verified working on piclockBR at `a5929ef`.

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
- **DRM mirror working** (`2ee57fa`): Root cause found — `findHDMI1Connector()` was hardcoded to target HDMI-A-1, but SDL already renders there. Fix: `findSpareHDMIConnector()` scans all connected HDMI outputs, identifies SDL's CRTC by highest fb_id, picks the spare. Both displays now show the clock on piclockBR. DRM state confirmed: plane-2→crtc-92 (SDL, fb=685) + plane-3→crtc-104 (mirror, fb=682).
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
  - `ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone --depth 1 --branch BRANCH https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && git checkout v1.x.y && make release-all GIT_TAG=v1.x.y'`
- Deploy release tarball to piclock.local from local Mac relay:
  - `scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-*-default-linux-arm64.tar.gz /tmp/`
  - `scp /tmp/clock8002-*-default-linux-arm64.tar.gz pi@piclock.local:/tmp/`
  - `ssh pi@piclock.local 'set -e; sudo systemctl stop clock8002.service alsa-ltc.service oled_daemon.service || true; sudo systemctl kill clock8002.service alsa-ltc.service oled_daemon.service || true; mkdir -p /tmp/clock8002-install && rm -rf /tmp/clock8002-install/clock8002-*-default-linux-arm64; tar xzf /tmp/clock8002-*-default-linux-arm64.tar.gz -C /tmp/clock8002-install; cd /tmp/clock8002-install/clock8002-*-default-linux-arm64; sudo bash install.sh > /tmp/clock8002-install-v1.x.y.log 2>&1; echo INSTALL_EXIT:$?; sudo systemctl start clock8002.service alsa-ltc.service oled_daemon.service'`
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

## Branch Promotion Decision (2026-04-26)

**Decision:** Promote `buildroot` → `master`. SDL3/Buildroot is the primary deployment path going forward. SDL2/Trixie is archived as `trixie`.

**Rationale:**
- Zero external users (0 stars, 0 forks, all issues/PRs from owner only)
- Buildroot image is the active production path; no new units are deployed on Trixie
- SDL3 port complete and soak-tested; SDL2 path has no ongoing development
- Code divergence (118 vs 420 commits from common ancestor) makes cross-branch merging impractical

**Branch operations completed (2026-04-26):**
1. ✅ Changed default branch to `master` (was `buildroot`)
2. ✅ Renamed `master` → `trixie`
3. ✅ Renamed `buildroot` → `master`
4. ✅ Local tracking updated
5. ✅ `.github/copilot-instructions.md` — branch references verified correct

**Tracking issue:** [#40](https://github.com/jpkelly/clock8002/issues/40)

**Docs already committed (this session):**
- `README.md` on `buildroot` — rewritten as primary user-facing doc (flash → configure)
- `buildroot-external/README.buildroot.md` — stale `pi5start.local` → `cm5.local`; reframed as developer/builder reference
- `README.md` on `master` (commit `a991419`) — legacy notice added pointing to new primary branch

## Next Suggested Release

- Next planned release: **v1.4.0** — next Buildroot release cycle.


## Update (2026-05-09, LTC recovered + .245 provenance snapshot + payload build status)

- User-confirmed state: LTC is working again on `.245`.
- Final `.245` provenance snapshot captured at `2026-05-09T22:50:08Z`:
  - host: `sdl-clock`
  - kernel: `Linux sdl-clock 6.12.41-v8 #3 SMP PREEMPT Wed Jan 14 11:49:24 UTC 2026 aarch64 GNU/Linux`
  - active runtime processes included:
    - `/bin/sh /root/alsa-ltc_pokemon.sh start`
    - `/root/alsa-ltc plughw:2,0 255.255.255.255 1245`
    - `/root/sdl-clock -C /boot/clock.ini`
    - `/root/clock-bridge`
- Snapshot hashes (`SHA256:path:sum`):
  - `/boot/Image`: `3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80`
  - `/boot/alsa-ltc`: `c78c3fc8094dd701a9f63465641525998812db9c56be68f703173178eb830417`
  - `/boot/sdl-clock`: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
  - `/boot/clock-bridge`: `09c4bf22e4956a172153f636a8f311bb73c404f8aa8a67ebe89ec419eb7a75dd`
  - `/boot/alsa-ltc_cmd.sh`: `67ca8da31b6791c8ba0438edae2467322a4a02641ec1aa2e9cc75ff24506fb82`
  - `/root/alsa-ltc`: `c78c3fc8094dd701a9f63465641525998812db9c56be68f703173178eb830417`
  - `/root/sdl-clock`: `3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d`
  - `/root/clock-bridge`: `09c4bf22e4956a172153f636a8f311bb73c404f8aa8a67ebe89ec419eb7a75dd`
  - `/root/alsa-ltc_cmd.sh`: `67ca8da31b6791c8ba0438edae2467322a4a02641ec1aa2e9cc75ff24506fb82`
- LTC command file content at snapshot time remained:
  - `/root/alsa-ltc - 255.255.255.255 1245`
- Current payload run on cm5 (`br-payload-20260509-154818`) completed successfully:
  - exit marker: `BR_BUILD_EXIT:0`
  - log confirms payload injections:
    - `Payload mode: injected prebuilt kernel modules from /srv/clock8002/prebuilt-kernel-bundles/current/modules/lib`
    - `Payload mode: injected prebuilt kernel assets from /srv/clock8002/prebuilt-kernel-bundles/current`
  - same log also contains kernel link markers (`vmlinux`, `.tmp_vmlinux*`) during the run; this indicates the attempted skip path did not fully eliminate kernel build steps in this invocation.
