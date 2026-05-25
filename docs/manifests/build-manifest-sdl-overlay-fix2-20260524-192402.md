# Build Manifest: sdl-overlay-fix2-20260524-192402

## Identity
- Build name: sdl-overlay-fix2-20260524-192402
- Date/time (local): 2026-05-24 19:24:02 PDT
- Build host: pi@cm5.local
- Screen session: br-sdl-overlay-fix2-20260524-192402
- Operator: jp
- Status: in-progress

## Source Inputs
- Repo path: /home/pi/clock8002-root-ram
- Branch: feature/root-ram
- Commit (full SHA): 93d4d13cba8d997b3d56e71ac962f6d18f2c114c
- Commit summary: fix: install sdl-clock to /root/ from package; remove stale overlay binary
- Working tree at build start: clean
- BR2_EXTERNAL path: /home/pi/clock8002-root-ram/buildroot-external
- Buildroot dir: /home/pi/buildroot
- Buildroot version epoch: 1742234000

## Kernel Inputs
- Prebuilt kernel enabled: 1
- Kernel bundle path: /srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234
- Kernel bundle symlink resolved: /srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234
- Bundle Image sha256: 3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80

## Build Command
- Reproduce with: `cd /home/pi/buildroot && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234 make clock8002-dirclean && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234 make`
- Output directory: /home/pi/buildroot/output
- Log file: /tmp/br-sdl-overlay-fix2-20260524-192402.log
- Exit file: /tmp/br-sdl-overlay-fix2-20260524-192402.exit

## Output Artifacts (filled automatically on build completion)
- sdcard.img path: /home/pi/buildroot/output/images/sdcard.img
- sdcard.img sha256: PENDING
- Image sha256 (output): PENDING
- rootfs.cpio sha256: PENDING

## Prebuilt Kernel Verification (filled automatically on build completion)
- verify_image_match: PENDING
- verify_overlays_match: PENDING
- verify_modules_match: PENDING

## Build Timing
- Start: 2026-05-24 19:24:02 PDT
- End: PENDING
- Elapsed: PENDING

## Runtime Binary Hashes (fill after flash and validation)
- sdl-clock sha256: PENDING
- alsa-ltc sha256: PENDING
- config.txt sha256: PENDING
- cmdline.txt sha256: PENDING

## Validation Results
- Boot status: PENDING
- Services (`clock8002`, `alsa-ltc`, `oled-daemon`): PENDING
- LTC status: PENDING
- Tester: PENDING
- Test date/time: PENDING

## Verdict
- Classification: candidate
- Notes: PENDING
