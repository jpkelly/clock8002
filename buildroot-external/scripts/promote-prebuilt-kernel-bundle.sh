#!/bin/sh
# Promote a verified prebuilt kernel bundle into the canonical store.
#
# Usage:
#   promote-prebuilt-kernel-bundle.sh <source-bundle-dir> [bundle-id] [store-root]
#
# Defaults:
#   store-root: /srv/clock8002/prebuilt-kernel-bundles
#
# Behavior:
#   - verifies source bundle
#   - copies into store-root/<bundle-id>
#   - writes BUNDLE.PROVENANCE
#   - makes bundle read-only
#   - repoints store-root/current -> <bundle-id>

set +e

SOURCE_BUNDLE="${1:-}"
BUNDLE_ID="${2:-}"
STORE_ROOT="${3:-${PREBUILT_KERNEL_STORE:-/srv/clock8002/prebuilt-kernel-bundles}}"
STAMP="$(date +%Y%m%d-%H%M%S)"

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
VERIFY_SCRIPT="${SCRIPT_DIR}/verify-prebuilt-kernel-bundle.sh"

if [ -z "${SOURCE_BUNDLE}" ]; then
        echo "PROMOTE_STATUS:missing_source_bundle" >&2
        exit 2
fi

if [ ! -x "${VERIFY_SCRIPT}" ]; then
        echo "PROMOTE_STATUS:missing_verify_script" >&2
        echo "PROMOTE_VERIFY_SCRIPT:${VERIFY_SCRIPT}" >&2
        exit 2
fi

"${VERIFY_SCRIPT}" "${SOURCE_BUNDLE}"
RC_VERIFY_SRC=$?
if [ "${RC_VERIFY_SRC}" -ne 0 ]; then
        echo "PROMOTE_STATUS:source_verify_failed" >&2
        exit 1
fi

if [ -z "${BUNDLE_ID}" ]; then
        IMAGE_HASH="$(sha256sum "${SOURCE_BUNDLE}/Image" | awk '{print substr($1,1,12)}')"
        BUNDLE_ID="bundle-${STAMP}-${IMAGE_HASH}"
fi

DEST_BUNDLE="${STORE_ROOT}/${BUNDLE_ID}"

mkdir -p "${STORE_ROOT}"
RC_MKDIR=$?
if [ "${RC_MKDIR}" -ne 0 ]; then
        echo "PROMOTE_STATUS:store_root_create_failed" >&2
        echo "PROMOTE_STORE_ROOT:${STORE_ROOT}" >&2
        exit 1
fi

if [ -e "${DEST_BUNDLE}" ]; then
        echo "PROMOTE_STATUS:bundle_id_exists" >&2
        echo "PROMOTE_DEST:${DEST_BUNDLE}" >&2
        exit 1
fi

if command -v rsync >/dev/null 2>&1; then
        rsync -a "${SOURCE_BUNDLE}/" "${DEST_BUNDLE}/"
        RC_COPY=$?
else
        cp -a "${SOURCE_BUNDLE}" "${DEST_BUNDLE}"
        RC_COPY=$?
fi

if [ "${RC_COPY}" -ne 0 ]; then
        echo "PROMOTE_STATUS:copy_failed" >&2
        echo "PROMOTE_SOURCE:${SOURCE_BUNDLE}" >&2
        echo "PROMOTE_DEST:${DEST_BUNDLE}" >&2
        exit 1
fi

"${VERIFY_SCRIPT}" "${DEST_BUNDLE}"
RC_VERIFY_DEST=$?
if [ "${RC_VERIFY_DEST}" -ne 0 ]; then
        rm -rf "${DEST_BUNDLE}"
        echo "PROMOTE_STATUS:dest_verify_failed" >&2
        exit 1
fi

REAL_SOURCE="$(CDPATH= cd -- "${SOURCE_BUNDLE}" && pwd)"
IMAGE_SHA256="$(sha256sum "${DEST_BUNDLE}/Image" | awk '{print $1}')"

cat > "${DEST_BUNDLE}/BUNDLE.PROVENANCE" <<EOF
bundle_id=${BUNDLE_ID}
promoted_at_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)
promoted_from=${REAL_SOURCE}
image_sha256=${IMAGE_SHA256}
EOF

# Make promoted bundles immutable by convention (read-only).
chmod -R a-w "${DEST_BUNDLE}"

(
        cd "${STORE_ROOT}" || exit 97
        ln -sfn "${BUNDLE_ID}" current
)
RC_LINK=$?
if [ "${RC_LINK}" -ne 0 ]; then
        echo "PROMOTE_STATUS:current_link_failed" >&2
        echo "PROMOTE_CURRENT:${STORE_ROOT}/current" >&2
        exit 1
fi

echo "PROMOTE_STATUS:ok"
echo "PROMOTE_SOURCE:${SOURCE_BUNDLE}"
echo "PROMOTE_STORE_ROOT:${STORE_ROOT}"
echo "PROMOTE_BUNDLE_ID:${BUNDLE_ID}"
echo "PROMOTE_DEST:${DEST_BUNDLE}"
echo "PROMOTE_CURRENT:${STORE_ROOT}/current"

exit 0
