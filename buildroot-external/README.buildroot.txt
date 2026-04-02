clock8002 Buildroot Prototype Architecture

Scope
- This tree is an initial br2-external scaffold to support a reproducible
  appliance image for Raspberry Pi 5.
- It intentionally starts as a prototype: package + hooks + sample defconfig.

Goals
1) Keep master stability-testing workflow untouched.
2) Isolate Buildroot work in a dedicated branch.
3) Produce minimal, deterministic images for clock8002 deployments.

Proposed Buildroot layout
- buildroot-external/
  - external.desc / external.mk / Config.in
  - package/clock8002/
    - Config.in
    - clock8002.mk
  - board/clock8002-rpi5/
    - post-build.sh
    - post-image.sh
  - configs/
    - clock8002_rpi5_defconfig.sample

Integration model (phased)
Phase 1
- Install-only package (already scaffolded): copies prebuilt sdl-clock,
  alsa-ltc, and service files if they exist in source dir.

Phase 2
- Build sdl-clock and alsa-ltc in Buildroot package step.
- Add explicit dependencies for SDL2, ALSA LTC, Python/OLED runtime.
- Install clock.ini/network.ini defaults and systemd units deterministically.

Phase 3
- Produce deployment image artifacts with boot + rootfs policies matching
  current install.sh behavior.
- Add CI artifact builds and long-run stability gates (12h / 24h) on test unit.

Notes
- post-build.sh masks ModemManager in target image to keep service noise low.
- defconfig is a sample because package names/options vary by Buildroot release.
  Final defconfig should be generated against a pinned Buildroot version.
