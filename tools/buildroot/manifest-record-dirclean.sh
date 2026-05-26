#!/bin/sh
# manifest-record-dirclean.sh — Append a dirclean event to the output dir manifest.
#
# Call this BEFORE running `make <pkg>-dirclean` so the manifest tracks what
# was selectively cleaned and when.  The preflight script uses DIRCLEAN_HISTORY
# to understand why a package directory may be absent even though the overall
# build status is 'success'.
#
# Usage:
#   manifest-record-dirclean.sh <output_dir> <package_name>
#
# Example:
#   manifest-record-dirclean.sh ~/buildroot/output clock8002

usage() {
    cat <<'EOF'
Usage: manifest-record-dirclean.sh <output_dir> <package_name>

  output_dir     Path to the Buildroot output directory
  package_name   Package name as passed to make (e.g. clock8002, sdl3)
EOF
}

OUTPUT_DIR="${1:-}"
PKG="${2:-}"

if [ -z "$OUTPUT_DIR" ] || [ -z "$PKG" ]; then
    echo "ERROR: output_dir and package_name are required" >&2
    usage
    exit 1
fi

MANIFEST="${OUTPUT_DIR}/.clock8002-build-state"

if [ ! -f "$MANIFEST" ]; then
    echo "[manifest-record-dirclean] WARNING: no manifest at ${MANIFEST} — skipping record" >&2
    exit 0
fi

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ENTRY="${TS}:${PKG}"

# Read existing history (may be empty or have prior entries separated by comma).
PREV_HISTORY=$(grep "^DIRCLEAN_HISTORY=" "$MANIFEST" 2>/dev/null \
               | head -1 | cut -d= -f2- | tr -d '"' || true)

if [ -z "$PREV_HISTORY" ]; then
    NEW_HISTORY="${ENTRY}"
else
    NEW_HISTORY="${PREV_HISTORY},${ENTRY}"
fi

# Update the DIRCLEAN_HISTORY field in-place.
sed -i.bak \
    -e "s|^DIRCLEAN_HISTORY=.*|DIRCLEAN_HISTORY=\"${NEW_HISTORY}\"|" \
    "$MANIFEST"
rm -f "${MANIFEST}.bak"

echo "[manifest-record-dirclean] recorded: ${ENTRY}"
echo "[manifest-record-dirclean] manifest: ${MANIFEST}"
