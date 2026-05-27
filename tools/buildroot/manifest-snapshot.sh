#!/bin/sh
# manifest-snapshot.sh — Write/update the per-output-dir build state manifest.
#
# Run this from the BUILD HOST (cm5) — it writes directly to the output dir.
#
# Usage:
#   manifest-snapshot.sh --start  <output_dir> [--src <repo>] [--br <buildroot>] [--target <make_target>]
#   manifest-snapshot.sh --finish <output_dir> <exit_code>
#
# The manifest lives at: <output_dir>/.clock8002-build-state
# Format: sh-sourceable KEY="value" lines (no spaces in values except quoted).
#
# Fields written on --start:
#   MANIFEST_VERSION, BUILD_STATUS=in-progress, BUILD_STARTED,
#   SRC_REPO_PATH, SRC_GIT_HEAD, SRC_GIT_BRANCH, SRC_GIT_DIRTY,
#   OVERLAY_DIR, OVERLAY_FINGERPRINT,
#   GOLDEN_CARD_FINGERPRINT (golden-working-card/ tree hash),
#   ALSA_LTC_SRC_HASH (v4/alsa-ltc.c file hash),
#   BR_CONFIG_HASH, BR2_EXTERNAL_VERSION, LAST_MAKE_TARGET,
#   SRC_GIT_LOG (last 20 commits, pipe-delimited one-liners)
#
# Fields added/updated on --finish:
#   BUILD_STATUS=success|failed, BUILD_FINISHED, IMAGE_SHA256

MANIFEST_VERSION="1"

usage() {
    cat <<'EOF'
Usage:
  manifest-snapshot.sh --start  <output_dir> [options]
  manifest-snapshot.sh --finish <output_dir> <exit_code>

Options for --start:
  --src  <path>   clock8002-root-ram repo on this host (default: ~/clock8002-root-ram)
  --br   <path>   Buildroot tree root (default: ~/buildroot)
  --target <str>  Make target description (default: "make")
EOF
}

MODE=""
OUTPUT_DIR=""
SRC_REPO=""
BR_DIR=""
MAKE_TARGET="make"
EXIT_CODE=""

while [ $# -gt 0 ]; do
    case "$1" in
        --start)  MODE="start";  OUTPUT_DIR="${2:-}"; shift 2 ;;
        --finish) MODE="finish"; OUTPUT_DIR="${2:-}"; EXIT_CODE="${3:-}"; shift 3 ;;
        --src)    SRC_REPO="${2:-}"; shift 2 ;;
        --br)     BR_DIR="${2:-}"; shift 2 ;;
        --target) MAKE_TARGET="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage; exit 2 ;;
    esac
done

if [ -z "$MODE" ] || [ -z "$OUTPUT_DIR" ]; then
    echo "ERROR: --start or --finish and an output_dir are required" >&2
    usage
    exit 2
fi

if [ "$MODE" = "finish" ] && [ -z "$EXIT_CODE" ]; then
    echo "ERROR: --finish requires an exit_code argument" >&2
    exit 2
fi

SRC_REPO="${SRC_REPO:-${HOME}/clock8002-root-ram}"
BR_DIR="${BR_DIR:-${HOME}/buildroot}"
MANIFEST="${OUTPUT_DIR}/.clock8002-build-state"
OVERLAY_DIR="${SRC_REPO}/buildroot-external/board/clock8002-rpi5/rootfs-overlay"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Compute a stable fingerprint of all files in a directory tree.
# Uses sorted sha256sum of every regular file relative to the directory root.
overlay_fingerprint() {
    dir="$1"
    if [ ! -d "$dir" ]; then
        echo "missing"
        return
    fi
    # find all regular files, sort by path, hash them, then hash the combined list
    (
        cd "$dir"
        find . -type f -print | LC_ALL=C sort | while IFS= read -r f; do
            sha256sum "$f"
        done
    ) | sha256sum | awk '{print $1}'
}

# sha256 of a single file
file_hash() {
    if [ -f "$1" ]; then
        sha256sum "$1" | awk '{print $1}'
    else
        echo "missing"
    fi
}

# Escape a value for safe embedding inside double quotes in a sh-sourceable file.
escape_dq() {
    printf '%s' "$1" | sed 's/[\\`"$]/\\&/g'
}

# Append a line to the DIRCLEAN_HISTORY field in the manifest.
# Used by manifest-record-dirclean.sh; not called from here.
# (Placed here as a note — see manifest-record-dirclean.sh.)

# ---------------------------------------------------------------------------
# --start: write full manifest
# ---------------------------------------------------------------------------
if [ "$MODE" = "start" ]; then
    mkdir -p "$OUTPUT_DIR"

    GIT_HEAD=$(git -C "$SRC_REPO" rev-parse HEAD 2>/dev/null || echo "unknown")
    GIT_BRANCH=$(git -C "$SRC_REPO" branch --show-current 2>/dev/null || echo "unknown")
    GIT_DIRTY=$(git -C "$SRC_REPO" status --short 2>/dev/null | wc -l | tr -d ' ')
    [ "$GIT_DIRTY" -gt 0 ] && GIT_DIRTY_FLAG="true" || GIT_DIRTY_FLAG="false"
    GIT_LOG=$(git -C "$SRC_REPO" log --oneline -20 2>/dev/null | tr '\n' '|' | sed 's/|$//')

    OVERLAY_FP=$(overlay_fingerprint "$OVERLAY_DIR")
    GOLDEN_CARD_DIR="${SRC_REPO}/buildroot-external/board/clock8002-rpi5/golden-working-card"
    GOLDEN_CARD_FP=$(overlay_fingerprint "$GOLDEN_CARD_DIR")
    ALSA_LTC_HASH=$(file_hash "${SRC_REPO}/v4/alsa-ltc.c")
    BR_CONFIG_HASH=$(file_hash "${BR_DIR}/.config")
    BR2_EXT_VER=$(grep "^BR2_EXTERNAL_CLOCK8002_VERSION=" "${BR_DIR}/.config" 2>/dev/null \
                  | cut -d= -f2- | tr -d '"' || echo "unknown")

    SRC_REPO_ESC=$(escape_dq "$SRC_REPO")
    GIT_HEAD_ESC=$(escape_dq "$GIT_HEAD")
    GIT_BRANCH_ESC=$(escape_dq "$GIT_BRANCH")
    OVERLAY_DIR_ESC=$(escape_dq "$OVERLAY_DIR")
    OVERLAY_FP_ESC=$(escape_dq "$OVERLAY_FP")
    GOLDEN_CARD_FP_ESC=$(escape_dq "$GOLDEN_CARD_FP")
    ALSA_LTC_HASH_ESC=$(escape_dq "$ALSA_LTC_HASH")
    BR_CONFIG_HASH_ESC=$(escape_dq "$BR_CONFIG_HASH")
    BR2_EXT_VER_ESC=$(escape_dq "$BR2_EXT_VER")
    MAKE_TARGET_ESC=$(escape_dq "$MAKE_TARGET")
    GIT_LOG_ESC=$(escape_dq "$GIT_LOG")

    # Preserve DIRCLEAN_HISTORY if an existing manifest is present (e.g. re-launch
    # after an interrupted build on the same output dir).
    PREV_HISTORY=""
    if [ -f "$MANIFEST" ]; then
        # shellcheck disable=SC1090
        PREV_HISTORY=$(grep "^DIRCLEAN_HISTORY=" "$MANIFEST" 2>/dev/null \
                       | head -1 | cut -d= -f2- | tr -d '"' || true)
    fi
    PREV_HISTORY_ESC=$(escape_dq "$PREV_HISTORY")

    cat > "$MANIFEST" <<EOF
# clock8002 build state — do not hand-edit
# Generated by manifest-snapshot.sh --start
# Source: ${SRC_REPO_ESC}

MANIFEST_VERSION="${MANIFEST_VERSION}"
BUILD_STATUS="in-progress"
BUILD_STARTED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_FINISHED=""
SRC_REPO_PATH="${SRC_REPO_ESC}"
SRC_GIT_HEAD="${GIT_HEAD_ESC}"
SRC_GIT_BRANCH="${GIT_BRANCH_ESC}"
SRC_GIT_DIRTY="${GIT_DIRTY_FLAG}"
OVERLAY_DIR="${OVERLAY_DIR_ESC}"
OVERLAY_FINGERPRINT="${OVERLAY_FP_ESC}"
GOLDEN_CARD_FINGERPRINT="${GOLDEN_CARD_FP_ESC}"
ALSA_LTC_SRC_HASH="${ALSA_LTC_HASH_ESC}"
BR_CONFIG_HASH="${BR_CONFIG_HASH_ESC}"
BR2_EXTERNAL_VERSION="${BR2_EXT_VER_ESC}"
LAST_MAKE_TARGET="${MAKE_TARGET_ESC}"
SRC_GIT_LOG="${GIT_LOG_ESC}"
IMAGE_SHA256=""
DIRCLEAN_HISTORY="${PREV_HISTORY_ESC}"
EOF

    echo "[manifest-snapshot] start: wrote ${MANIFEST}"
    echo "[manifest-snapshot]   branch=${GIT_BRANCH}  head=${GIT_HEAD}"
    echo "[manifest-snapshot]   overlay_fp=${OVERLAY_FP}"
    echo "[manifest-snapshot]   br_config=${BR_CONFIG_HASH}"
fi

# ---------------------------------------------------------------------------
# --finish: update status, timestamp, image hash
# ---------------------------------------------------------------------------
if [ "$MODE" = "finish" ]; then
    if [ ! -f "$MANIFEST" ]; then
        echo "[manifest-snapshot] WARNING: no manifest found at ${MANIFEST}, cannot update" >&2
        exit 1
    fi

    if [ "$EXIT_CODE" = "0" ]; then
        NEW_STATUS="success"
        IMAGE_PATH="${OUTPUT_DIR}/images/sdcard.img"
        IMAGE_HASH=$(file_hash "$IMAGE_PATH")
    else
        NEW_STATUS="failed"
        IMAGE_HASH="n/a"
    fi

    FINISHED="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

    # Use sed to update the three fields in-place.
    # Portable sed -i with backup (works on both GNU and BSD).
    sed -i.bak \
        -e "s|^BUILD_STATUS=.*|BUILD_STATUS=\"${NEW_STATUS}\"|" \
        -e "s|^BUILD_FINISHED=.*|BUILD_FINISHED=\"${FINISHED}\"|" \
        -e "s|^IMAGE_SHA256=.*|IMAGE_SHA256=\"${IMAGE_HASH}\"|" \
        "$MANIFEST"
    rm -f "${MANIFEST}.bak"

    echo "[manifest-snapshot] finish: status=${NEW_STATUS}  image_sha=${IMAGE_HASH}"
    echo "[manifest-snapshot]   manifest=${MANIFEST}"
fi
