# Build Manifest: repro-245-20260524-062711

## Identity
- Build name: repro-245-20260524-062711
- Date/time (local): 2026-05-24 06:27:11
- Build host: pi@cm5.local
- Operator: GitHub Copilot (assisted)
- Status: in-progress

## Source Inputs
- Source repo path: /home/pi/clock8002
- Source repo commit (full SHA): 3d835b27fc5e4ca5de689a1d79ee4df4b2d89c9b
- External repo path: /home/pi/clock8002-root-ram
- External repo commit (full SHA): 51703f6198f2d872daa4f654f5ca34c31c5dd535

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

## Output Artifacts (pending until build completion)
- sdcard.img path: /home/pi/output-repro-245-20260524-062711/images/sdcard.img
- sdcard.img sha256: PENDING
- Image sha256: PENDING
- rootfs.cpio.gz sha256: PENDING

## Runtime Binary Hashes (pending)
- sdl-clock sha256: PENDING
- alsa-ltc sha256: PENDING
- setup.sh sha256: PENDING
- config.txt sha256: PENDING
- cmdline.txt sha256: PENDING

## Validation Results (pending)
- Boot status: PENDING
- Services (`clock8002`, `alsa-ltc`, `oled-daemon`): PENDING
- LTC status: PENDING
- Tester: PENDING

## Notes
- This is an intentional mixed-input repro build to match observed .245 behavior.
- Provenance is taken from verified cm5 state at build start.