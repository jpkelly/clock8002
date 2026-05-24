#!/usr/bin/env bash
# cm5-build-launch.sh — Launch a Buildroot build on cm5 with mandatory manifest creation.
#
# Writes docs/manifests/build-manifest-<purpose>-<timestamp>.md capturing all
# reproducibility inputs BEFORE the screen session starts.  The manifest is
# your record: given its contents you can always rebuild the same image from
# scratch, even after every output directory has been deleted.
#
# Usage:
#   tools/buildroot/cm5-build-launch.sh --purpose <label> [options]
#
# Example:
#   tools/buildroot/cm5-build-launch.sh --purpose ltc-fix
#   tools/buildroot/cm5-build-launch.sh --purpose release-1.3 \
#       --bundle /srv/clock8002/prebuilt-kernel-bundles/bundle-245-6.12.41-v8-20260509-161234

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
HOST="pi@cm5.local"
BUILDROOT_DIR="/home/pi/buildroot"
SOURCE_DIR="/home/pi/clock8002-root-ram"
BUNDLE_SYMLINK="/srv/clock8002/prebuilt-kernel-bundles/current"
BUNDLE_OVERRIDE=""
PREBUILT_KERNEL=1
KEY_ARG=""
PURPOSE=""
OPERATOR="${USER:-unknown}"

usage() {
    cat <<EOF
Usage: $(basename "$0") --purpose <label> [options]

Required:
  --purpose <label>       Short label for this build, used in session/manifest name.
                          E.g. "powerbtn", "ltc-fix", "release-1.3"

Options:
  --bundle <path>         Explicit kernel bundle path on cm5 (default: current symlink)
  --no-prebuilt-kernel    Set CLOCK8002_PREBUILT_KERNEL=0 (compile kernel from source)
  --key <pubkey>          BR2_PICLOCKKEY value to bake dev SSH key into image
  --host <user@host>      SSH host (default: $HOST)
  --buildroot <path>      Buildroot directory on cm5 (default: $BUILDROOT_DIR)
  --source <path>         clock8002-root-ram path on cm5 (default: $SOURCE_DIR)
  -h, --help              Show help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --purpose)           PURPOSE="${2:-}"; shift 2 ;;
        --bundle)            BUNDLE_OVERRIDE="${2:-}"; shift 2 ;;
        --no-prebuilt-kernel) PREBUILT_KERNEL=0; shift ;;
        --key)               KEY_ARG="${2:-}"; shift 2 ;;
        --host)              HOST="${2:-}"; shift 2 ;;
        --buildroot)         BUILDROOT_DIR="${2:-}"; shift 2 ;;
        --source)            SOURCE_DIR="${2:-}"; shift 2 ;;
        -h|--help)           usage; exit 0 ;;
        *)                   echo "Unknown argument: $1" >&2; usage; exit 2 ;;
    esac
done

if [[ -z "$PURPOSE" ]]; then
    echo "ERROR: --purpose is required" >&2
    echo "" >&2
    usage
    exit 2
fi

# Sanitize purpose for use in filenames and screen session names.
PURPOSE_SAFE="${PURPOSE//[^a-zA-Z0-9_-]/-}"
PURPOSE_SAFE="${PURPOSE_SAFE,,}"
TS="$(date +%Y%m%d-%H%M%S)"
SESSION="br-${PURPOSE_SAFE}-${TS}"
MANIFEST_NAME="build-manifest-${PURPOSE_SAFE}-${TS}.md"
MANIFEST_PATH="${REPO_ROOT}/docs/manifests/${MANIFEST_NAME}"

echo "=== cm5-build-launch ==="
echo "Session:  ${SESSION}"
echo "Manifest: docs/manifests/${MANIFEST_NAME}"
echo "Host:     ${HOST}"
echo ""

# --- Step 1: collect all inputs from cm5 in one SSH call ---
echo "Collecting inputs from ${HOST}..."
INPUTS=$(ssh "$HOST" sh -s -- "$SOURCE_DIR" "$BUILDROOT_DIR" "$BUNDLE_SYMLINK" <<'REMOTE'
set -u
SOURCE_DIR="$1"
BUILDROOT_DIR="$2"
BUNDLE_SYMLINK="$3"

BRANCH=$(git -C "$SOURCE_DIR" branch --show-current 2>/dev/null || echo "unknown")
COMMIT=$(git -C "$SOURCE_DIR" rev-parse HEAD 2>/dev/null || echo "unknown")
SHORT=$(git -C "$SOURCE_DIR" rev-parse --short HEAD 2>/dev/null || echo "unknown")
SUBJECT=$(git -C "$SOURCE_DIR" log --format=%s -1 2>/dev/null || echo "unknown")
DIRTY=$(git -C "$SOURCE_DIR" status --short 2>/dev/null || true)
BR2_EPOCH=$(grep "^BR2_VERSION_EPOCH " "$BUILDROOT_DIR/Makefile" 2>/dev/null | awk '{print $NF}' || echo "unknown")
BUNDLE_REAL=$(readlink -f "$BUNDLE_SYMLINK" 2>/dev/null || echo "not-found")
if [ -f "${BUNDLE_REAL}/Image" ]; then
    BUNDLE_IMG_SHA=$(sha256sum "${BUNDLE_REAL}/Image" | awk '{print $1}')
else
    BUNDLE_IMG_SHA="missing-or-not-found"
fi
ACTIVE_SCREENS=$(screen -ls 2>/dev/null | grep -E "^\s+[0-9]+" || true)

# Output as key=value, one per line.  Values with spaces are tricky so we
# base64-encode the ones that might contain them.
echo "branch=${BRANCH}"
echo "commit=${COMMIT}"
echo "short=${SHORT}"
echo "subject_b64=$(printf '%s' "$SUBJECT" | base64 -w0)"
echo "dirty_b64=$(printf '%s' "$DIRTY" | base64 -w0)"
echo "br2_epoch=${BR2_EPOCH}"
echo "bundle_real=${BUNDLE_REAL}"
echo "bundle_img_sha=${BUNDLE_IMG_SHA}"
echo "screens_b64=$(printf '%s' "$ACTIVE_SCREENS" | base64 -w0)"
REMOTE
)

# Parse outputs.
_get() { echo "$INPUTS" | grep "^${1}=" | head -1 | cut -d= -f2-; }
_b64() { _get "$1" | base64 -d 2>/dev/null || true; }

BRANCH=$(_get branch)
COMMIT=$(_get commit)
SHORT=$(_get short)
SUBJECT=$(_b64 subject_b64)
DIRTY=$(_b64 dirty_b64)
BR2_EPOCH=$(_get br2_epoch)
BUNDLE_REAL=$(_get bundle_real)
BUNDLE_IMG_SHA=$(_get bundle_img_sha)
ACTIVE_SCREENS=$(_b64 screens_b64)
BUNDLE_PATH="${BUNDLE_OVERRIDE:-${BUNDLE_REAL}}"

echo "Branch:    ${BRANCH}"
echo "Commit:    ${SHORT}  ${SUBJECT}"
echo "Bundle:    ${BUNDLE_REAL}"
echo "BR2 epoch: ${BR2_EPOCH}"
echo ""

# --- Step 2: preflight checks ---
PREFLIGHT_OK=1

if [[ -n "$DIRTY" ]]; then
    echo "PREFLIGHT FAIL: working tree is dirty:" >&2
    echo "$DIRTY" >&2
    PREFLIGHT_OK=0
fi

if [[ "$BUNDLE_IMG_SHA" == "missing-or-not-found" ]]; then
    echo "PREFLIGHT FAIL: kernel bundle Image not found at: ${BUNDLE_PATH}" >&2
    PREFLIGHT_OK=0
fi

if [[ "$PREFLIGHT_OK" == "0" ]]; then
    exit 1
fi

if [[ -n "$ACTIVE_SCREENS" ]]; then
    echo "WARNING: active screen sessions already running on cm5:"
    echo "$ACTIVE_SCREENS"
    echo ""
    read -r -p "Continue anyway? [y/N] " CONFIRM
    if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
        echo "Aborted." >&2
        exit 1
    fi
    echo ""
fi

# --- Step 3: write manifest locally ---
echo "Writing manifest..."
mkdir -p "$(dirname "$MANIFEST_PATH")"
DIRTY_STATUS="$([ -z "$DIRTY" ] && echo "clean" || echo "DIRTY — see notes")"
DIRTY_NOTES="$([ -z "$DIRTY" ] && echo "" || printf "\n- Working tree dirty at launch:\n\`\`\`\n%s\n\`\`\`" "$DIRTY")"

cat > "$MANIFEST_PATH" <<EOF
# Build Manifest: ${PURPOSE_SAFE}-${TS}

## Identity
- Build name: ${PURPOSE_SAFE}-${TS}
- Date/time (local): $(date '+%Y-%m-%d %H:%M:%S %Z')
- Build host: ${HOST}
- Screen session: ${SESSION}
- Operator: ${OPERATOR}
- Status: in-progress

## Source Inputs
- Repo path: ${SOURCE_DIR}
- Branch: ${BRANCH}
- Commit (full SHA): ${COMMIT}
- Commit summary: ${SUBJECT}
- Working tree at build start: ${DIRTY_STATUS}
- BR2_EXTERNAL path: ${SOURCE_DIR}/buildroot-external
- Buildroot dir: ${BUILDROOT_DIR}
- Buildroot version epoch: ${BR2_EPOCH}

## Kernel Inputs
- Prebuilt kernel enabled: ${PREBUILT_KERNEL}
- Kernel bundle path: ${BUNDLE_PATH}
- Kernel bundle symlink resolved: ${BUNDLE_REAL}
- Bundle Image sha256: ${BUNDLE_IMG_SHA}

## Build Command
- Reproduce with: \`cd ${BUILDROOT_DIR} && CLOCK8002_PREBUILT_KERNEL=${PREBUILT_KERNEL} CLOCK8002_PREBUILT_KERNEL_BUNDLE=${BUNDLE_PATH} make clock8002-dirclean && CLOCK8002_PREBUILT_KERNEL=${PREBUILT_KERNEL} CLOCK8002_PREBUILT_KERNEL_BUNDLE=${BUNDLE_PATH} make\`
- Output directory: ${BUILDROOT_DIR}/output
- Log file: /tmp/${SESSION}.log
- Exit file: /tmp/${SESSION}.exit

## Output Artifacts (filled automatically on build completion)
- sdcard.img path: ${BUILDROOT_DIR}/output/images/sdcard.img
- sdcard.img sha256: PENDING
- Image sha256 (output): PENDING
- rootfs.cpio sha256: PENDING

## Prebuilt Kernel Verification (filled automatically on build completion)
- verify_image_match: PENDING
- verify_overlays_match: PENDING
- verify_modules_match: PENDING

## Build Timing
- Start: $(date '+%Y-%m-%d %H:%M:%S %Z')
- End: PENDING
- Elapsed: PENDING

## Runtime Binary Hashes (fill after flash and validation)
- sdl-clock sha256: PENDING
- alsa-ltc sha256: PENDING
- config.txt sha256: PENDING
- cmdline.txt sha256: PENDING

## Validation Results
- Boot status: PENDING
- Services (\`clock8002\`, \`alsa-ltc\`, \`oled-daemon\`): PENDING
- LTC status: PENDING
- Tester: PENDING
- Test date/time: PENDING

## Verdict
- Classification: candidate
- Notes: PENDING${DIRTY_NOTES}
EOF
echo "Written: docs/manifests/${MANIFEST_NAME}"
echo ""

# --- Step 4: arm post-build watcher on cm5 ---
# Pipe the watcher script content to cm5; no nested heredoc needed.
START_EPOCH="$(date +%s)"
MANIFEST_REMOTE="${SOURCE_DIR}/docs/manifests/${MANIFEST_NAME}"
OUTPUT_DIR="${BUILDROOT_DIR}/output"

WATCHER_SCRIPT="/home/pi/kernel-dev-snapshots/postbuild-${SESSION}.sh"
WATCHER_LOG="/home/pi/kernel-dev-snapshots/postbuild-${SESSION}.run.log"

python3 - "$SESSION" "$OUTPUT_DIR" "$BUNDLE_REAL" "$START_EPOCH" "$MANIFEST_REMOTE" <<'PYEOF'
import sys, base64
session, output_dir, bundle, start_epoch, manifest = sys.argv[1:]
script = """\
#!/bin/sh
set -u
EXIT_FILE="/tmp/{session}.exit"
OUTPUT_DIR="{output_dir}"
BUNDLE="{bundle}"
BUILD_START_EPOCH={start_epoch}
MANIFEST="{manifest}"

mkdir -p "$(dirname "$MANIFEST")"
while [ ! -f "$EXIT_FILE" ]; do sleep 20; done

ts=$(date +%Y%m%d-%H%M%S)
END_EPOCH=$(date +%s)
END_HUMAN=$(date '+%Y-%m-%d %H:%M:%S %Z')
ELAPSED=$((END_EPOCH - BUILD_START_EPOCH))
EH=$((ELAPSED/3600)); EM=$(((ELAPSED%3600)/60)); ES=$((ELAPSED%60))
RC=$(cat "$EXIT_FILE" 2>/dev/null || echo unknown)

SDCARD_SHA=MISSING; IMAGE_SHA=MISSING; ROOTFS_SHA=MISSING
IMG_MATCH=SKIP; OV_MATCH=SKIP; MOD_MATCH=SKIP

[ -f "$OUTPUT_DIR/images/sdcard.img" ] && SDCARD_SHA=$(sha256sum "$OUTPUT_DIR/images/sdcard.img" | awk '{{print $1}}')
[ -f "$OUTPUT_DIR/images/Image" ] && IMAGE_SHA=$(sha256sum "$OUTPUT_DIR/images/Image" | awk '{{print $1}}')
[ -f "$OUTPUT_DIR/images/rootfs.cpio" ] && ROOTFS_SHA=$(sha256sum "$OUTPUT_DIR/images/rootfs.cpio" | awk '{{print $1}}')

if [ -f "$BUNDLE/Image" ] && [ -f "$OUTPUT_DIR/images/Image" ]; then
  B=$(sha256sum "$BUNDLE/Image" | awk '{{print $1}}')
  O=$(sha256sum "$OUTPUT_DIR/images/Image" | awk '{{print $1}}')
  [ "$B" = "$O" ] && IMG_MATCH=PASS || IMG_MATCH=FAIL
fi

if [ -d "$BUNDLE/overlays" ] && [ -d "$OUTPUT_DIR/images/rpi-firmware/overlays" ]; then
  B=$(find "$BUNDLE/overlays" -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{{print $1}}')
  O=$(find "$OUTPUT_DIR/images/rpi-firmware/overlays" -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{{print $1}}')
  [ "$B" = "$O" ] && OV_MATCH=PASS || OV_MATCH=FAIL
fi

MODSRC=""
[ -d "$BUNDLE/modules/lib/modules" ] && MODSRC="$BUNDLE/modules/lib/modules" || {{ [ -d "$BUNDLE/modules" ] && MODSRC="$BUNDLE/modules"; }}
if [ -n "$MODSRC" ] && [ -d "$OUTPUT_DIR/target/lib/modules" ]; then
  B=$(find "$MODSRC" -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{{print $1}}')
  O=$(find "$OUTPUT_DIR/target/lib/modules" -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{{print $1}}')
  [ "$B" = "$O" ] && MOD_MATCH=PASS || MOD_MATCH=FAIL
fi

if [ -f "$MANIFEST" ]; then
  sed -i "s|^- sdcard.img sha256: PENDING|- sdcard.img sha256: $SDCARD_SHA|" "$MANIFEST"
  sed -i "s|^- Image sha256 (output): PENDING|- Image sha256 (output): $IMAGE_SHA|" "$MANIFEST"
  sed -i "s|^- rootfs.cpio sha256: PENDING|- rootfs.cpio sha256: $ROOTFS_SHA|" "$MANIFEST"
  sed -i "s|^- verify_image_match: PENDING|- verify_image_match: $IMG_MATCH|" "$MANIFEST"
  sed -i "s|^- verify_overlays_match: PENDING|- verify_overlays_match: $OV_MATCH|" "$MANIFEST"
  sed -i "s|^- verify_modules_match: PENDING|- verify_modules_match: $MOD_MATCH|" "$MANIFEST"
  sed -i "s|^- End: PENDING|- End: $END_HUMAN|" "$MANIFEST"
  sed -i "s|^- Elapsed: PENDING|- Elapsed: ${{EH}}h ${{EM}}m ${{ES}}s|" "$MANIFEST"
  sed -i "s|^- Status: in-progress|- Status: build-complete exit=$RC|" "$MANIFEST"
fi

echo "watcher-complete session={session} rc=$RC elapsed=${{EH}}h${{EM}}m${{ES}}s"
""".format(
    session=session,
    output_dir=output_dir,
    bundle=bundle,
    start_epoch=start_epoch,
    manifest=manifest,
)
print(script)
PYEOF | ssh "$HOST" "mkdir -p /home/pi/kernel-dev-snapshots && cat > '${WATCHER_SCRIPT}' && chmod +x '${WATCHER_SCRIPT}'"

ssh "$HOST" "nohup '${WATCHER_SCRIPT}' > '${WATCHER_LOG}' 2>&1 & echo watcher_pid=\$!"
echo ""

# --- Step 5: launch screen session ---
echo "Launching screen session: ${SESSION}"

MAKE_CMD="cd ${BUILDROOT_DIR} || exit 2"
MAKE_CMD="${MAKE_CMD}; export CLOCK8002_PREBUILT_KERNEL=${PREBUILT_KERNEL}"
MAKE_CMD="${MAKE_CMD}; export CLOCK8002_PREBUILT_KERNEL_BUNDLE=${BUNDLE_PATH}"
MAKE_CMD="${MAKE_CMD}; export CLOCK8002_BUILD_SESSION=${SESSION}"
if [[ -n "$KEY_ARG" ]]; then
    MAKE_CMD="${MAKE_CMD}; export BR2_PICLOCKKEY='${KEY_ARG}'"
fi
MAKE_CMD="${MAKE_CMD}; make clock8002-dirclean && make"
MAKE_CMD="${MAKE_CMD}; rc=\$?; printf '%s\n' \"\$rc\" > /tmp/${SESSION}.exit; exit \"\$rc\""

ssh "$HOST" "screen -dmS '${SESSION}' sh -lc '${MAKE_CMD}'"
echo ""

# --- Step 6: print monitoring commands ---
cat <<EOF
=== Build launched ===

Monitor:
  tools/buildroot/cm5-build-status.sh --session ${SESSION}

Check completion:
  ssh ${HOST} 'cat /tmp/${SESSION}.exit'

Manifest (auto-updated on completion):
  docs/manifests/${MANIFEST_NAME}

Changes are local only — not deployed yet. Push when ready.
EOF
