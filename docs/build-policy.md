# Build Policy

## Purpose
This policy prevents mixed-source and ambiguous builds. A build is valid only if
all preflight checks pass and a manifest is produced.

## Hard Rules
1. Use one source tree per build. Do not mix files from multiple repos in one build.
2. Use one branch/commit set per build and record exact SHAs in a manifest.
3. Use a fresh output directory for every build.
4. Build in a screen session on cm5.
5. Pin the exact prebuilt kernel bundle path. Do not rely on an unverified symlink.
6. Do not treat image filenames as provenance. Trust only manifest values and hashes.
7. Do not promote a build without hardware validation notes.

## Preflight Checklist (must pass)
1. `git -C <source-repo> status --short` is empty.
2. `git -C <source-repo> rev-parse --short HEAD` matches intended SHA.
3. `git -C <external-repo> status --short` is empty.
4. `git -C <external-repo> rev-parse --short HEAD` matches intended SHA.
5. Kernel bundle path exists and is readable.
6. Output directory does not already exist.

## Standard Build Shape
1. Run `clock8002_rpi5_defconfig` in a new `O=` output path.
2. Run `clock8002-dirclean` before the main `make`.
3. Build with `CLOCK8002_PREBUILT_KERNEL=1` unless explicitly overridden.
4. On success, compute hashes and write manifest.

## Promotion Rules
1. Candidate build: image produced + manifest complete.
2. Known-good build: candidate plus hardware checks pass (boot, services, LTC).
3. Reproducible build: known-good behavior reproduced from a fresh output dir using the same manifest inputs.

## Minimum Hardware Validation
1. Boot succeeds.
2. `clock8002`, `alsa-ltc`, and `oled-daemon` are running.
3. LTC decode verified.
4. Required serial/network behavior verified for the target scenario.
