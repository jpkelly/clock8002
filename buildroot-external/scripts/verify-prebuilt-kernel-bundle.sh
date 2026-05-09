#!/bin/sh
# Verify prebuilt kernel bundle layout and integrity.
#
# Usage:
#   verify-prebuilt-kernel-bundle.sh <bundle-dir>
#
# Optional env:
#   STRICT_SHA256=1   # fail if SHA256SUMS is missing

set +e

BUNDLE_DIR="${1:-${PREBUILT_KERNEL_BUNDLE:-}}"
STRICT_SHA256="${STRICT_SHA256:-0}"
STAMP="$(date +%Y%m%d-%H%M%S)"
HASH_LOG="/tmp/prebuilt-bundle-verify-${STAMP}.sha256.log"

if [ -z "${BUNDLE_DIR}" ]; then
        echo "BUNDLE_STATUS:missing_path" >&2
        exit 2
fi

if [ ! -d "${BUNDLE_DIR}" ]; then
        echo "BUNDLE_STATUS:not_a_directory" >&2
        echo "BUNDLE_PATH:${BUNDLE_DIR}" >&2
        exit 2
fi

if [ ! -f "${BUNDLE_DIR}/Image" ]; then
        echo "BUNDLE_STATUS:missing_image" >&2
        echo "BUNDLE_PATH:${BUNDLE_DIR}" >&2
        exit 1
fi

DTB_DIR=""
if [ -d "${BUNDLE_DIR}/dtbs" ]; then
        DTB_DIR="${BUNDLE_DIR}/dtbs"
elif [ -d "${BUNDLE_DIR}/dtb" ]; then
        DTB_DIR="${BUNDLE_DIR}/dtb"
fi

if [ -z "${DTB_DIR}" ]; then
        echo "BUNDLE_STATUS:missing_dtbs" >&2
        echo "BUNDLE_PATH:${BUNDLE_DIR}" >&2
        exit 1
fi

OVERLAY_DIR="${BUNDLE_DIR}/overlays"
if [ ! -d "${OVERLAY_DIR}" ]; then
        echo "BUNDLE_STATUS:missing_overlays" >&2
        echo "BUNDLE_PATH:${BUNDLE_DIR}" >&2
        exit 1
fi

MODULES_DIR=""
if [ -d "${BUNDLE_DIR}/modules/lib/modules" ]; then
        MODULES_DIR="${BUNDLE_DIR}/modules/lib/modules"
elif [ -d "${BUNDLE_DIR}/modules" ]; then
        MODULES_DIR="${BUNDLE_DIR}/modules"
fi

if [ -z "${MODULES_DIR}" ]; then
        echo "BUNDLE_STATUS:missing_modules" >&2
        echo "BUNDLE_PATH:${BUNDLE_DIR}" >&2
        exit 1
fi

DTB_COUNT="$(find "${DTB_DIR}" -type f -name '*.dtb' | wc -l | tr -d ' ')"
if [ "${DTB_COUNT}" -eq 0 ]; then
        echo "BUNDLE_STATUS:empty_dtbs" >&2
        echo "BUNDLE_DTB_DIR:${DTB_DIR}" >&2
        exit 1
fi

OVERLAY_COUNT="$(find "${OVERLAY_DIR}" -mindepth 1 -maxdepth 1 -type f | wc -l | tr -d ' ')"
if [ "${OVERLAY_COUNT}" -eq 0 ]; then
        echo "BUNDLE_STATUS:empty_overlays" >&2
        echo "BUNDLE_OVERLAY_DIR:${OVERLAY_DIR}" >&2
        exit 1
fi

if [ ! -f "${OVERLAY_DIR}/overlay_map.dtb" ]; then
        echo "BUNDLE_STATUS:missing_overlay_map" >&2
        echo "BUNDLE_OVERLAY_DIR:${OVERLAY_DIR}" >&2
        exit 1
fi

if [ ! -f "${OVERLAY_DIR}/bcm2712d0.dtbo" ]; then
        echo "BUNDLE_STATUS:missing_bcm2712d0" >&2
        echo "BUNDLE_OVERLAY_DIR:${OVERLAY_DIR}" >&2
        exit 1
fi

MODULE_RELEASE_COUNT="$(find "${MODULES_DIR}" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
if [ "${MODULE_RELEASE_COUNT}" -eq 0 ]; then
        echo "BUNDLE_STATUS:empty_module_releases" >&2
        echo "BUNDLE_MODULES_DIR:${MODULES_DIR}" >&2
        exit 1
fi

MODULE_FILE_COUNT="$(find "${MODULES_DIR}" -type f -name '*.ko*' | wc -l | tr -d ' ')"
if [ "${MODULE_FILE_COUNT}" -eq 0 ]; then
        echo "BUNDLE_STATUS:no_module_files" >&2
        echo "BUNDLE_MODULES_DIR:${MODULES_DIR}" >&2
        exit 1
fi

SHA_STATUS="missing"
if [ -f "${BUNDLE_DIR}/SHA256SUMS" ]; then
        (
                cd "${BUNDLE_DIR}" || exit 98
                sha256sum -c SHA256SUMS > "${HASH_LOG}" 2>&1
        )
        RC_HASH=$?
        if [ "${RC_HASH}" -ne 0 ]; then
                echo "BUNDLE_STATUS:checksum_failed" >&2
                echo "BUNDLE_HASH_LOG:${HASH_LOG}" >&2
                exit 1
        fi
        SHA_STATUS="verified"
elif [ "${STRICT_SHA256}" = "1" ]; then
        echo "BUNDLE_STATUS:missing_sha256sums" >&2
        echo "BUNDLE_PATH:${BUNDLE_DIR}" >&2
        exit 1
fi

echo "BUNDLE_STATUS:ok"
echo "BUNDLE_PATH:${BUNDLE_DIR}"
echo "BUNDLE_DTB_DIR:${DTB_DIR}"
echo "BUNDLE_DTB_COUNT:${DTB_COUNT}"
echo "BUNDLE_OVERLAY_DIR:${OVERLAY_DIR}"
echo "BUNDLE_OVERLAY_COUNT:${OVERLAY_COUNT}"
echo "BUNDLE_MODULES_DIR:${MODULES_DIR}"
echo "BUNDLE_MODULE_RELEASES:${MODULE_RELEASE_COUNT}"
echo "BUNDLE_MODULE_FILES:${MODULE_FILE_COUNT}"
echo "BUNDLE_SHA256:${SHA_STATUS}"

exit 0
