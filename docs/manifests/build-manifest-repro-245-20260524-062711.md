# Build Manifest: repro-245-20260524-062711

## Identity
- Build name: repro-245-20260524-062711
- Date/time (local): 2026-05-24 06:27:11
- Build host: pi@cm5.local
- Operator: GitHub Copilot (assisted)
- Status: known-good

## Source Inputs
- Source repo path: /home/pi/clock8002-root-ram/v4
- Source repo commit (full SHA): 99fe9908bb71a5ad2c51b25439fda35788ae3b36
- External repo path: /home/pi/clock8002-root-ram
- External repo commit (full SHA): 99fe9908bb71a5ad2c51b25439fda35788ae3b36
- Note: source path was corrected from /home/pi/clock8002/v4 (squashfs-era) via sed before incremental rebuild; both source and external are the same repo at 99fe990

## Kernel Inputs
- Prebuilt kernel enabled: 1
- Kernel bundle absolute path: /srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234

## Build Command Shape
- Defconfig: `CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-repro-245-20260524-062711 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot clock8002_rpi5_defconfig`
- Dirclean: `CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-repro-245-20260524-062711 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot clock8002-dirclean`
- Main make: `CLOCK8002_PREBUILT_KERNEL=1 BR2_PICLOCKKEY=<cm5 ~/.ssh/id_rsa.pub> make O=/home/pi/output-repro-245-20260524-062711 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot`
- Output directory: /home/pi/output-repro-245-20260524-062711
- Screen session: br-build-repro-245
- Log file: /tmp/br-build-repro-245.log
- Exit file: /tmp/br-build-repro-245.exit

## Output Artifacts
- sdcard.img path: /home/pi/output-repro-245-20260524-062711/images/sdcard.img
- sdcard.img sha256: 7e785383aff30a9e1f32bcdefee41e245f56c7630de47ca484c2a7396ded0e24
- Image sha256: 7e785383aff30a9e1f32bcdefee41e245f56c7630de47ca484c2a7396ded0e24
- rootfs.cpio.gz sha256: 43799deb632372895db61629eaf38c4439283c9c92b4dbefddf25c7b7e7c21e7

## Runtime Binary Hashes
- sdl-clock sha256: a9a236ab81d1d5a1ced7d497ed5af2e3e65b2bef64362b8b13d0d7f23c82da45  (from golden-working-card; compiled at cf73f8f)
- alsa-ltc sha256: 24e945ab24a4a2d5f6921d0bc8490dbbe3b21a9fbc769c435c88033e101469fd  (path: /boot/alsa-ltc)
- setup.sh sha256: 14ea7bed4d6f71896c4df1c17bdeb6d5cbb7e21e1013b9415f66856f410ca5a4
- config.txt sha256: 3e9dd6dcd71a41644855a5f27272610eea604a2b8c2a59d2957c0b35141712bd
- cmdline.txt sha256: a0a5ecc95fb8e1c34cb9a5701fba0317a88fd564c1dee4ec7541ba3eed9e6f18

## Validation Results
- Boot status: OK
- sdl-clock: running, keyboard I/Q confirmed (I=toggle overlay, Q=quit)
- alsa-ltc: running
- oled-daemon: running
- LTC status: decode verified
- Web UI: OK — reports version ram-root 99fe990 BR
- Validated: 2026-05-24
- Tester: jp

## Notes
- Build started as repro-245 with wrong source path; root cause identified mid-session.
- CLOCK8002_SOURCE_DIR corrected from /home/pi/clock8002/v4 to /home/pi/clock8002-root-ram/v4 via sed on cm5 .config before incremental rebuild.
- sdl-clock binary in image comes from golden-working-card (CLOCK8002_PREBUILT_KERNEL=1 path), not compiled during this build. Binary was updated in commit 99fe990 and was independently verified on device before the rebuild.
- SDL3 keyboard input uses SDL_EVDEV_DEVICES workaround in clock_cmd.sh (SDL3 built without libudev; full libudev rebuild pending a future clean build).
- Screen session for incremental rebuild: br-incr-iq-20260524.