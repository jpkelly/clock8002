# Build Manifest: powerbtn-20260524-100420

## Identity
- Build name: powerbtn-20260524-100420
- Date/time (local): 2026-05-24 10:04:20 PDT
- Build host: pi@cm5.local
- Screen session: br-powerbtn-20260524-2
- Operator: jp
- Status: known-good

## Source Inputs
- Repo path: /home/pi/clock8002-root-ram
- Branch: feature/root-ram
- Commit (full SHA): 522361eeff5ed2ee1acb4b7ae3b401e3015fdc51
- Commit summary: feat: add robust power-button handler; docs: remove stale wrapper script refs
- Working tree at build start: clean
- BR2_EXTERNAL path: /home/pi/clock8002-root-ram/buildroot-external
- BR2_EXTERNAL version string: working-2026-05-23-powerbutton-ltc-11-g522361e

## Kernel Inputs
- Prebuilt kernel enabled: 1
- Kernel bundle absolute path: /srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234
- Kernel bundle symlink used: /srv/clock8002/prebuilt-kernel-bundles/current
- Bundle Image sha256: 3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80
- Bundle BUNDLE.PROVENANCE: not present

## Build Command
- Defconfig: (used pre-existing buildroot config — no defconfig step in this launch)
- Dirclean: `cd /home/pi/buildroot && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make clock8002-dirclean`
- Main make: `cd /home/pi/buildroot && CLOCK8002_PREBUILT_KERNEL=1 CLOCK8002_PREBUILT_KERNEL_BUNDLE=/srv/clock8002/prebuilt-kernel-bundles/current make`
- Output directory: /home/pi/buildroot/output
- Log file: /tmp/br-powerbtn-20260524-2.log
- Exit file: /tmp/br-powerbtn-20260524-2.exit
- Post-build report: /home/pi/kernel-dev-snapshots/report-br-powerbtn-20260524-2-<timestamp>.txt

## Notes
- Output directory is the default /home/pi/buildroot/output rather than a per-build O= path (policy deviation; noted for reproducibility).
- To reproduce: use fresh `O=` path, same commit, same bundle, same BR2_EXTERNAL.
- Kernel was compiled from source during this build (BR2_LINUX_KERNEL=y in defconfig) but prebuilt payload is injected over it in post-build/post-image hooks. Final image kernel comes from the bundle.
- Compiled kernel tree preserved at: /home/pi/kernel-dev-snapshots/linux-custom-br-powerbtn-20260524-2-<timestamp>

## Output Artifacts
- sdcard.img path: /home/pi/buildroot/output/images/sdcard.img
- sdcard.img sha256: a7e25ad78f27df7a1d69890574876c889e23e811e77e501125af8a7c5d961c9b
- Image sha256 (output): 3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80
- Image sha256 (bundle — already known): 3062308d7916349dc5737c9c02dbbffac6f23a9344fb14b6d3d442bc09c83f80
- rootfs.cpio sha256: 18508cbb3ce638770a4abc68c82a8d58f4a39154ff512d59757dcae022599d53

## Prebuilt Kernel Verification (from post-build report)
- verify_image_match: PASS
- verify_overlays_match: FAIL (watcher path mismatch — bundle has 13 overlays, output/images/rpi-firmware/overlays has 361; post-image.sh injection verified correct by hardware validation)
- verify_modules_match: FAIL (watcher path mismatch — bundle modules/lib structure differs from comparison path; post-build.sh injection verified correct by hardware validation)

## Build Timing
- Start: 2026-05-24 10:04:20 PDT
- End: 2026-05-24 12:35:27 PDT
- Elapsed: 2h 31m 7s

## Runtime Binary Hashes (fill after flash/validation)
- sdl-clock sha256: PENDING
- alsa-ltc sha256: PENDING
- config.txt sha256: PENDING
- cmdline.txt sha256: PENDING

## Validation Results
- Boot status: PASS
- Services (`clock8002`, `alsa-ltc`, `oled-daemon`): PASS
- LTC status: PASS — LTC decode confirmed working
- Tester: jp
- Test date/time: 2026-05-24 PDT

## Verdict
- Classification: known-good
- Notes: Boot, services, and LTC all confirmed working on piClock.local. First known-good image from the feature/root-ram branch with power-button handler.
