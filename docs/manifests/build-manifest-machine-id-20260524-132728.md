# Build Manifest: machine-id-20260524-132728

## Identity
- Build name: machine-id-20260524-132728
- Date/time (local): 2026-05-24 13:27:28 PDT
- Build host: pi@cm5.local
- Screen session: br-machine-id-20260524-132728
- Operator: jp
- Status: known-good

## Source Inputs
- Repo path: /home/pi/clock8002-root-ram
- Branch: feature/root-ram
- Commit (full SHA): 4e9600209603bff34b4c736864b740c30951ff2a
- Commit summary: feat: persist machine-id across reboots via FAT partition
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
- Log file: /tmp/br-machine-id-20260524-132728.log
- Exit file: /tmp/br-machine-id-20260524-132728.exit

## Output Artifacts (filled automatically on build completion)
- sdcard.img path: /home/pi/buildroot/output/images/sdcard.img
- sdcard.img sha256: ee38995c9b973967ca159ff8fced6e8eb69a60dc1a1531908cb294e3a7c84335
- Image sha256 (output): 3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80
- rootfs.cpio sha256: 7c7f010f999a13fe6f50951625366504ef8e17536c3434818b67f91aded59f04

## Prebuilt Kernel Verification (filled automatically on build completion)
- verify_image_match: PASS
- verify_overlays_match: SKIP (known path mismatch false-negative)
- verify_modules_match: SKIP (known path mismatch false-negative)

## Build Timing
- Start: 2026-05-24 13:27:28 PDT
- End: 2026-05-24 13:29:20 PDT
- Elapsed: 1m52s

## Runtime Binary Hashes (fill after flash and validation)
- sdl-clock sha256: 3e1b789c80eae6c59f249856bb23cb6239681e656be54c1b3afe32182523621d
- alsa-ltc sha256: c78c3fc8094dd701a9f63465641525998812db9c56be68f703173178eb830417
- config.txt sha256: 3e9dd6dcd71a41644855a5f27272610eea604a2b8c2a59d2957c0b35141712bd
- cmdline.txt sha256: a0a5ecc95fb8e1c34cb9a5701fba0317a88fd564c1dee4ec7541ba3eed9e6f18

## Validation Results
- Boot status: OK
- Services (`clock8002`, `alsa-ltc`, `oled-daemon`): OK
- LTC status: OK
- Power button: OK
- machine-id persistence: PASS (stable across reboots)
- Tester: jp
- Test date/time: 2026-05-24 PDT

## Verdict
- Classification: candidate
- Notes: PENDING
