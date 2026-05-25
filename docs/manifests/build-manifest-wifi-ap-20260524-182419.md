# Build Manifest: wifi-ap-20260524-182419

## Identity
- Build name: wifi-ap-20260524-182419
- Date/time (local): 2026-05-24 18:24:19 PDT
- Build host: pi@cm5.local
- Screen session: br-wifi-ap-20260524-182419
- Operator: jp
- Status: candidate

## Source Inputs
- Repo path: /home/pi/clock8002-root-ram
- Branch: feature/root-ram
- Commit (full SHA): d4db3be57ef1d0425d57fabe3adf6d6b7ff8412b
- Commit summary: feat: Wi-Fi AP mode via hostapd boot injection
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
- Log file: /tmp/br-wifi-ap-20260524-182419.log
- Exit file: /tmp/br-wifi-ap-20260524-182419.exit

## Output Artifacts (filled automatically on build completion)
- sdcard.img path: /home/pi/buildroot/output/images/sdcard.img
- sdcard.img sha256: 531185770cf01bc3bbbb6fe7bb94878ad2f813733da607ecdec6ffe5a97497b3
- Image sha256 (output): 3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80
- rootfs.cpio sha256 (gz): f7af48e8d12df5b6d25d1e6b57f708088b8250edebbc773d58cf436164b61d2e

## Prebuilt Kernel Verification (filled automatically on build completion)
- verify_image_match: PASS (output Image sha256 matches bundle Image sha256: 3062308d...)
- verify_overlays_match: PENDING
- verify_modules_match: PENDING

## Build Timing
- Start: 2026-05-24 18:24:19 PDT
- End: 2026-05-24 18:26:30 PDT
- Elapsed: ~2 min

## Runtime Binary Hashes (fill after flash and validation)
- sdl-clock sha256: PENDING
- alsa-ltc sha256: PENDING
- config.txt sha256: PENDING
- cmdline.txt sha256: PENDING

## Validation Results
- Boot status: PASS
- Wi-Fi AP (piClock-ap): PASS — SSID visible, AP mode confirmed
- Services (`clock8002`, `alsa-ltc`, `oled-daemon`): PENDING
- LTC status: PENDING
- Tester: jp
- Test date/time: 2026-05-24

## Verdict
- Classification: known-good (partial — AP confirmed; full LTC/services validation pending)
- Notes: Wi-Fi AP via hostapd boot injection working on first boot from this image.
