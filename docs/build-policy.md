# Build Policy

## Purpose
This policy prevents mixed-source and ambiguous builds. A build is valid only if
all preflight checks pass and a manifest is produced.

## Hard Rules
1. Use one source tree per build. Do not mix files from multiple repos in one build.
2. Use one branch/commit set per build and record exact SHAs in a manifest.
3. **Releases** require a fresh output directory. **Release candidates** may use a fresh or reused directory at operator's discretion. **Dev builds** reuse the existing output directory.
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
6. For releases: output directory must not already exist. For release candidates: operator chooses fresh or reuse. For dev builds: output directory must be from the same branch.

## Build Shapes

### Incremental dev build (day-to-day iteration)
- Reuse the existing output directory from the last build on the same branch.
- Run `make O=<outdir> clock8002-dirclean && make O=<outdir> CLOCK8002_PREBUILT_KERNEL=1 ...` in a screen session.
- No manifest required. Note the commit SHA in the screen session name or log.

### Release candidate
- May reuse an existing output directory (for speed) or start from a clean directory (for confidence) — operator's choice at build time.
- Before launching, check cm5 for existing output dirs from the same branch/commit and confirm the choice (reuse or fresh) before proceeding.
- Manifest required. On success, confirm manifest PENDING fields are filled (hashes, elapsed time, exit code).
- Hardware validation required before promoting to release.

### Release (final)
- **Must** start from a fresh output directory. No exceptions.
- Launch with `tools/buildroot/cm5-build-launch.sh --purpose <label>`.
- Manifest required and must be complete before the image is distributed.
- Reproducibility build (independent clean rebuild from the same manifest inputs) recommended before tagging.

## Promotion Rules
1. Candidate build: image produced + manifest complete.
2. Known-good build: candidate plus hardware checks pass (boot, services, LTC).
3. Reproducible build: known-good behavior reproduced from a fresh output dir using the same manifest inputs.

## Minimum Hardware Validation
1. Boot succeeds.
2. `clock8002`, `alsa-ltc`, and `oled-daemon` are running.
3. LTC decode verified.
4. Required serial/network behavior verified for the target scenario.
