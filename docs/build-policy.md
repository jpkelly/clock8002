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

---

## Output Directory State Tracking

Each Buildroot output directory on cm5 carries a machine-readable manifest at
`<output_dir>/.clock8002-build-state`. This file is the single source of truth
for the build state of that directory. It is written by `manifest-snapshot.sh`
and read by `manifest-preflight.sh`.

### Workflow

**Before every build:**

```sh
tools/buildroot/manifest-preflight.sh <output_dir>
```

The preflight script compares current source/overlay/config state against the
manifest and prints one of four recommendations:

| Recommendation | Trigger condition |
|---|---|
| **UP TO DATE** — no build needed | All inputs match |
| **INCREMENTAL — clock8002-dirclean + make** | Only Go source changed |
| **INCREMENTAL — forced rootfs rebuild** | Only overlay files changed |
| **INCREMENTAL — dirclean + rootfs rebuild** | Both Go source and overlay changed |
| **FULL CLEAN REBUILD** | .config changed, branch changed, or bad/missing manifest |

The preflight is **advisory** — you can override. Exit code 0 = incremental safe,
1 = full clean recommended, 2 = full clean required (interrupted/failed/missing),
3 = usage error.

**Starting a build:**

```sh
ssh pi@cm5.local 'sh' <<'REMOTE'
tools/buildroot/manifest-snapshot.sh --start <output_dir> \
    --src ~/clock8002-root-ram --br ~/buildroot \
    --target "make clock8002-dirclean && make"
REMOTE
```

**After the build completes:**

```sh
# On cm5 (substitute actual exit code):
tools/buildroot/manifest-snapshot.sh --finish <output_dir> <exit_code>
```

**Before a dirclean step:**

```sh
tools/buildroot/manifest-record-dirclean.sh <output_dir> clock8002
make O=<output_dir> clock8002-dirclean
```

### Decision Rules

| Manifest state | Action |
|---|---|
| Missing | Full clean rebuild |
| `BUILD_STATUS=in-progress` | Full clean rebuild (interrupted) |
| `BUILD_STATUS=failed` | Full clean rebuild |
| Branch mismatch | Full clean rebuild |
| `.config` hash mismatch | Full clean rebuild |
| Overlay fingerprint mismatch | Forced rootfs rebuild (delete fakeroot stamps) |
| `SRC_GIT_HEAD` mismatch only | `clock8002-dirclean && make` |
| All fields match | No-op |

### Manifest Fields

| Field | Description |
|---|---|
| `MANIFEST_VERSION` | Schema version (currently 1) |
| `BUILD_STATUS` | `in-progress` / `success` / `failed` |
| `BUILD_STARTED` / `BUILD_FINISHED` | ISO 8601 UTC timestamps |
| `SRC_REPO_PATH` | Absolute path to clock8002-root-ram on the build host |
| `SRC_GIT_HEAD` | Full SHA of the source commit |
| `SRC_GIT_BRANCH` | Branch name |
| `SRC_GIT_DIRTY` | `true` if uncommitted changes were present at build start |
| `OVERLAY_DIR` | Path to the rootfs overlay |
| `OVERLAY_FINGERPRINT` | SHA256 of sorted hashes of all overlay files |
| `BR_CONFIG_HASH` | SHA256 of `~/buildroot/.config` |
| `BR2_EXTERNAL_VERSION` | `BR2_EXTERNAL_CLOCK8002_VERSION` from `.config` |
| `LAST_MAKE_TARGET` | Human-readable description of the make invocation |
| `IMAGE_SHA256` | SHA256 of `images/sdcard.img` (set on success) |
| `DIRCLEAN_HISTORY` | Comma-separated `timestamp:package` dirclean events |

### Output Directory Naming

When a clean build succeeds, optionally rename `~/buildroot/output` to a
named archive directory:

```sh
mv ~/buildroot/output ~/output-<7-char-commit>-<timestamp>
```

Only rename if the manifest shows `BUILD_STATUS=success`. Drop directories with
`in-progress` or `failed` status — they are not safe to use as incremental bases.
