#!/bin/sh
# manifest-preflight.sh — Compare output dir build state to current source/config.
#
# Run this from the BUILD HOST (cm5) BEFORE invoking make.
# Reads <output_dir>/.clock8002-build-state, computes current state, diffs them,
# and prints a recommendation with the exact command to run.
#
# Exit codes:
#   0  — recommendation is INCREMENTAL or UP-TO-DATE (safe to proceed)
#   1  — recommendation is FULL-CLEAN (destructive change detected)
#   2  — recommendation is FULL-CLEAN (manifest missing/corrupted/interrupted)
#   3  — usage error
#
# Usage:
#   manifest-preflight.sh <output_dir> [--src <repo>] [--br <buildroot>]
#
# Examples:
#   manifest-preflight.sh ~/buildroot/output
#   manifest-preflight.sh ~/output-clean-bda4db9-20260525-175132 --src ~/clock8002-root-ram

usage() {
    cat <<'EOF'
Usage: manifest-preflight.sh <output_dir> [options]

Options:
  --src  <path>   clock8002-root-ram repo on this host (default: ~/clock8002-root-ram)
  --br   <path>   Buildroot tree root (default: ~/buildroot)
  -h, --help      Show this help

Exit codes:
  0   Incremental or up-to-date — safe to proceed
  1   Full clean required — destructive change detected
  2   Full clean required — no/bad manifest
  3   Usage error
EOF
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
OUTPUT_DIR=""
SRC_REPO=""
BR_DIR=""

while [ $# -gt 0 ]; do
    case "$1" in
        --src)    SRC_REPO="${2:-}"; shift 2 ;;
        --br)     BR_DIR="${2:-}";  shift 2 ;;
        -h|--help) usage; exit 3 ;;
        -*)       echo "Unknown option: $1" >&2; usage; exit 3 ;;
        *)
            if [ -z "$OUTPUT_DIR" ]; then
                OUTPUT_DIR="$1"; shift
            else
                echo "Unexpected argument: $1" >&2; usage; exit 3
            fi
            ;;
    esac
done

if [ -z "$OUTPUT_DIR" ]; then
    echo "ERROR: output_dir is required" >&2
    usage
    exit 3
fi

SRC_REPO="${SRC_REPO:-${HOME}/clock8002-root-ram}"
BR_DIR="${BR_DIR:-${HOME}/buildroot}"
MANIFEST="${OUTPUT_DIR}/.clock8002-build-state"
OVERLAY_DIR="${SRC_REPO}/buildroot-external/board/clock8002-rpi5/rootfs-overlay"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
hr() { printf '%0.s-' $(seq 1 70); echo; }

overlay_fingerprint() {
    dir="$1"
    if [ ! -d "$dir" ]; then echo "missing"; return; fi
    (
        cd "$dir"
        find . -type f -print | LC_ALL=C sort | while IFS= read -r f; do
            sha256sum "$f"
        done
    ) | sha256sum | awk '{print $1}'
}

file_hash() {
    if [ -f "$1" ]; then sha256sum "$1" | awk '{print $1}'; else echo "missing"; fi
}

ok()   { echo "  [OK]   $*"; }
warn() { echo "  [WARN] $*"; }
diff_() { echo "  [DIFF] $*"; }
fail() { echo "  [FAIL] $*"; }

# ---------------------------------------------------------------------------
# Collect CURRENT state
# ---------------------------------------------------------------------------
echo ""
hr
echo " manifest-preflight: ${OUTPUT_DIR}"
hr

echo ""
echo "Collecting current state..."
CUR_HEAD=$(git -C "$SRC_REPO" rev-parse HEAD 2>/dev/null || echo "unknown")
CUR_SHORT=$(git -C "$SRC_REPO" rev-parse --short HEAD 2>/dev/null || echo "unknown")
CUR_BRANCH=$(git -C "$SRC_REPO" branch --show-current 2>/dev/null || echo "unknown")
CUR_DIRTY=$(git -C "$SRC_REPO" status --short 2>/dev/null | wc -l | tr -d ' ')
[ "$CUR_DIRTY" -gt 0 ] && CUR_DIRTY_FLAG="true" || CUR_DIRTY_FLAG="false"
CUR_OVERLAY_FP=$(overlay_fingerprint "$OVERLAY_DIR")
CUR_BR_CONFIG_HASH=$(file_hash "${BR_DIR}/.config")
CUR_BR2_EXT_VER=$(grep "^BR2_EXTERNAL_CLOCK8002_VERSION=" "${BR_DIR}/.config" 2>/dev/null \
                  | cut -d= -f2- | tr -d '"' || echo "unknown")

echo "  src:     ${SRC_REPO}"
echo "  branch:  ${CUR_BRANCH}"
echo "  head:    ${CUR_SHORT} (${CUR_HEAD})"
echo "  dirty:   ${CUR_DIRTY_FLAG}"
echo "  overlay: ${CUR_OVERLAY_FP}"
echo "  br_cfg:  ${CUR_BR_CONFIG_HASH}"
echo "  ext_ver: ${CUR_BR2_EXT_VER}"
echo ""

# ---------------------------------------------------------------------------
# Case: no manifest at all
# ---------------------------------------------------------------------------
if [ ! -f "$MANIFEST" ]; then
    fail "No manifest found at: ${MANIFEST}"
    echo ""
    echo "RECOMMENDATION: FULL CLEAN REBUILD"
    echo "  This output directory has no recorded build state."
    echo "  Run:"
    echo "    cd ~/buildroot && make clean && make"
    echo ""
    exit 2
fi

# ---------------------------------------------------------------------------
# Load manifest
# ---------------------------------------------------------------------------
# shellcheck disable=SC1090
. "$MANIFEST"
# After sourcing: MANIFEST_VERSION, BUILD_STATUS, BUILD_STARTED, BUILD_FINISHED,
# SRC_REPO_PATH, SRC_GIT_HEAD, SRC_GIT_BRANCH, SRC_GIT_DIRTY,
# OVERLAY_FINGERPRINT, BR_CONFIG_HASH, BR2_EXTERNAL_VERSION,
# LAST_MAKE_TARGET, IMAGE_SHA256, DIRCLEAN_HISTORY

echo "Loaded manifest: ${MANIFEST}"
echo "  built:   ${BUILD_STARTED:-unknown}"
echo "  status:  ${BUILD_STATUS:-unknown}"
echo "  head:    ${SRC_GIT_HEAD:-unknown}"
echo "  branch:  ${SRC_GIT_BRANCH:-unknown}"
echo ""

# ---------------------------------------------------------------------------
# Case: interrupted or failed build
# ---------------------------------------------------------------------------
if [ "${BUILD_STATUS:-}" = "in-progress" ]; then
    fail "Previous build status is 'in-progress' — build was interrupted."
    echo ""
    echo "RECOMMENDATION: FULL CLEAN REBUILD"
    echo "  The output directory may be in a partially-built state."
    echo "  Run:"
    echo "    cd ~/buildroot && make clean && make"
    echo ""
    exit 2
fi

if [ "${BUILD_STATUS:-}" = "failed" ]; then
    fail "Previous build status is 'failed'."
    echo ""
    echo "RECOMMENDATION: FULL CLEAN REBUILD"
    echo "  The output directory is from a failed build."
    echo "  Run:"
    echo "    cd ~/buildroot && make clean && make"
    echo ""
    exit 2
fi

# ---------------------------------------------------------------------------
# Compare fields
# ---------------------------------------------------------------------------
NEEDS_FULL_CLEAN=0
NEEDS_ROOTFS_REBUILD=0
NEEDS_DIRCLEAN=0

echo "Comparing state..."
echo ""

# Branch changed → always full clean
if [ "${SRC_GIT_BRANCH:-}" != "$CUR_BRANCH" ]; then
    diff_ "Branch changed: '${SRC_GIT_BRANCH:-unknown}' → '${CUR_BRANCH}'"
    NEEDS_FULL_CLEAN=1
else
    ok "Branch matches: ${CUR_BRANCH}"
fi

# .config changed → full clean
if [ "${BR_CONFIG_HASH:-}" != "$CUR_BR_CONFIG_HASH" ]; then
    diff_ "Buildroot .config changed"
    diff_ "  was: ${BR_CONFIG_HASH:-unknown}"
    diff_ "  now: ${CUR_BR_CONFIG_HASH}"
    NEEDS_FULL_CLEAN=1
else
    ok ".config hash matches"
fi

# BR2_EXTERNAL version changed → full clean
if [ "${BR2_EXTERNAL_VERSION:-}" != "$CUR_BR2_EXT_VER" ]; then
    diff_ "BR2_EXTERNAL version changed: '${BR2_EXTERNAL_VERSION:-unknown}' → '${CUR_BR2_EXT_VER}'"
    NEEDS_FULL_CLEAN=1
else
    ok "BR2_EXTERNAL version matches: ${CUR_BR2_EXT_VER}"
fi

# Overlay fingerprint changed → rootfs rebuild required
if [ "${OVERLAY_FINGERPRINT:-}" != "$CUR_OVERLAY_FP" ]; then
    diff_ "rootfs-overlay fingerprint changed"
    diff_ "  was: ${OVERLAY_FINGERPRINT:-unknown}"
    diff_ "  now: ${CUR_OVERLAY_FP}"
    NEEDS_ROOTFS_REBUILD=1
else
    ok "Overlay fingerprint matches"
fi

# Source HEAD changed → at minimum clock8002-dirclean
if [ "${SRC_GIT_HEAD:-}" != "$CUR_HEAD" ]; then
    diff_ "Source HEAD changed: ${SRC_GIT_HEAD:-unknown} → ${CUR_HEAD}"
    NEEDS_DIRCLEAN=1
else
    ok "Source HEAD matches: ${CUR_SHORT}"
fi

# Dirty tree warning
if [ "$CUR_DIRTY_FLAG" = "true" ]; then
    warn "Working tree has ${CUR_DIRTY} uncommitted change(s) — provenance will be imprecise"
fi

# ---------------------------------------------------------------------------
# Emit recommendation
# ---------------------------------------------------------------------------
echo ""
hr
echo " RECOMMENDATION"
hr
echo ""

OUTDIR_VAR="${OUTPUT_DIR}"

if [ "$NEEDS_FULL_CLEAN" -eq 1 ]; then
    echo "  FULL CLEAN REBUILD required (config/branch/ext mismatch)"
    echo ""
    echo "  Run:"
    echo "    cd ~/buildroot && make O=${OUTDIR_VAR} clean && make O=${OUTDIR_VAR}"
    echo ""
    exit 1
fi

if [ "$NEEDS_ROOTFS_REBUILD" -eq 1 ] && [ "$NEEDS_DIRCLEAN" -eq 1 ]; then
    echo "  INCREMENTAL — clock8002 rebuild + forced rootfs rebuild"
    echo "  (Both Go source and overlay changed)"
    echo ""
    echo "  Run:"
    echo "    cd ~/buildroot \\"
    echo "      && make O=${OUTDIR_VAR} clock8002-dirclean \\"
    echo "      && rm -f ${OUTDIR_VAR}/build/buildroot-fs/*/fakeroot.stamp \\"
    echo "      && rm -f ${OUTDIR_VAR}/target/.stamp_target_install_source_check \\"
    echo "      && make O=${OUTDIR_VAR}"
    echo ""
    exit 0
fi

if [ "$NEEDS_ROOTFS_REBUILD" -eq 1 ]; then
    echo "  INCREMENTAL — forced rootfs rebuild only"
    echo "  (Overlay changed; Go source unchanged)"
    echo ""
    echo "  Run:"
    echo "    cd ~/buildroot \\"
    echo "      && rm -f ${OUTDIR_VAR}/build/buildroot-fs/*/fakeroot.stamp \\"
    echo "      && rm -f ${OUTDIR_VAR}/target/.stamp_target_install_source_check \\"
    echo "      && make O=${OUTDIR_VAR}"
    echo ""
    exit 0
fi

if [ "$NEEDS_DIRCLEAN" -eq 1 ]; then
    echo "  INCREMENTAL — clock8002-dirclean + make"
    echo "  (Go source changed; overlay and config unchanged)"
    echo ""
    echo "  Run:"
    echo "    cd ~/buildroot \\"
    echo "      && make O=${OUTDIR_VAR} clock8002-dirclean \\"
    echo "      && make O=${OUTDIR_VAR}"
    echo ""
    exit 0
fi

# Nothing changed
echo "  UP TO DATE — no changes detected"
echo "  The output directory matches current source, overlay, and config."
if [ -n "${IMAGE_SHA256:-}" ] && [ "$IMAGE_SHA256" != "" ]; then
    echo "  Image SHA256: ${IMAGE_SHA256}"
fi
echo ""
exit 0
