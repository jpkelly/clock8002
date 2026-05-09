#!/bin/sh
# Payload-only build wrapper: always inject kernel + modules from prebuilt bundle.
#
# Usage:
#   build-with-kernel-fallback.sh [BUILDROOT_DIR] [PREBUILT_KERNEL_BUNDLE]
#
# Examples:
#   ./build-with-kernel-fallback.sh ~/buildroot
#   ./build-with-kernel-fallback.sh ~/buildroot /srv/clock8002/prebuilt-kernel-bundles/bundle-<id>

set +e

BUILDROOT_DIR="${1:-${BUILDROOT_DIR:-$HOME/buildroot}}"
PREBUILT_BUNDLE="${2:-${PREBUILT_KERNEL_BUNDLE:-${CLOCK8002_PREBUILT_KERNEL_BUNDLE:-}}}"
PREBUILT_STORE="${CLOCK8002_PREBUILT_KERNEL_STORE:-/srv/clock8002/prebuilt-kernel-bundles}"
PREBUILT_CURRENT="${PREBUILT_STORE}/current"
OUTPUT_DIR="${BUILDROOT_OUTPUT_DIR:-${BUILDROOT_DIR}/output}"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
VERIFY_SCRIPT="${SCRIPT_DIR}/verify-prebuilt-kernel-bundle.sh"

STAMP="$(date +%Y%m%d-%H%M%S)"
BUILD_LOG="/tmp/br-payload-${STAMP}.log"
VERIFY_LOG="/tmp/br-payload-${STAMP}.verify.log"
HASH_LOG="/tmp/br-payload-${STAMP}.sha256.log"
KERNEL_SRC_LOG="/tmp/br-payload-${STAMP}.linux-source.log"

if [ ! -f "${BUILDROOT_DIR}/Makefile" ]; then
        echo "ERROR: Buildroot not found at ${BUILDROOT_DIR}" >&2
        exit 2
fi

if [ -z "${PREBUILT_BUNDLE}" ] && [ -d "${PREBUILT_CURRENT}" ]; then
        PREBUILT_BUNDLE="${PREBUILT_CURRENT}"
fi

if [ -z "${PREBUILT_BUNDLE}" ] || [ ! -d "${PREBUILT_BUNDLE}" ]; then
        echo "PAYLOAD_STATUS:missing_bundle"
        echo "PAYLOAD_BUNDLE:${PREBUILT_BUNDLE}"
        echo "PAYLOAD_BUNDLE_DEFAULT:${PREBUILT_CURRENT}"
        exit 3
fi

bundle_looks_valid() {
        BUNDLE="$1"

        [ -d "${BUNDLE}" ] || return 1
        [ -f "${BUNDLE}/Image" ] || return 1

        if [ ! -d "${BUNDLE}/dtbs" ] && [ ! -d "${BUNDLE}/dtb" ]; then
                return 1
        fi

        if [ ! -d "${BUNDLE}/overlays" ]; then
                return 1
        fi

        if [ ! -d "${BUNDLE}/modules" ] && [ ! -d "${BUNDLE}/modules/lib/modules" ]; then
                return 1
        fi

        return 0
}

if [ -x "${VERIFY_SCRIPT}" ]; then
        "${VERIFY_SCRIPT}" "${PREBUILT_BUNDLE}" > "${VERIFY_LOG}" 2>&1
        RC_VERIFY=$?
        if [ "${RC_VERIFY}" -ne 0 ]; then
                echo "PAYLOAD_STATUS:invalid_bundle"
                echo "PAYLOAD_BUNDLE:${PREBUILT_BUNDLE}"
                echo "PAYLOAD_VERIFY_LOG:${VERIFY_LOG}"
                exit 4
        fi
else
        if ! bundle_looks_valid "${PREBUILT_BUNDLE}"; then
                echo "PAYLOAD_STATUS:invalid_bundle"
                echo "PAYLOAD_BUNDLE:${PREBUILT_BUNDLE}"
                exit 4
        fi

        if [ -f "${PREBUILT_BUNDLE}/SHA256SUMS" ]; then
                (
                        cd "${PREBUILT_BUNDLE}" || exit 98
                        sha256sum -c SHA256SUMS > "${HASH_LOG}" 2>&1
                )
                RC_HASH=$?
                if [ "${RC_HASH}" -ne 0 ]; then
                        echo "PAYLOAD_STATUS:bundle_hash_failed"
                        echo "PAYLOAD_HASH_LOG:${HASH_LOG}"
                        exit 4
                fi
        fi
fi

echo "BUILD_MODE:PAYLOAD_ONLY"
echo "PAYLOAD_BUNDLE:${PREBUILT_BUNDLE}"
if [ "${PREBUILT_BUNDLE}" = "${PREBUILT_CURRENT}" ]; then
        echo "PAYLOAD_BUNDLE_SOURCE:default_current"
fi
echo "PAYLOAD_LOG:${BUILD_LOG}"

# Ensure kernel source dir exists, then pre-stamp linux package as completed so
# payload builds never compile kernel/modules from source.
(
        cd "${BUILDROOT_DIR}" || exit 97
        CLOCK8002_PREBUILT_KERNEL=1 \
        CLOCK8002_PREBUILT_KERNEL_BUNDLE="${PREBUILT_BUNDLE}" \
        make linux-source > "${KERNEL_SRC_LOG}" 2>&1
)
RC_SRC=$?
if [ "${RC_SRC}" -ne 0 ]; then
        echo "PAYLOAD_STATUS:linux_source_failed"
        echo "PAYLOAD_KERNEL_SOURCE_LOG:${KERNEL_SRC_LOG}"
        exit "${RC_SRC}"
fi

LINUX_DIR="$(find "${OUTPUT_DIR}/build" -maxdepth 1 -type d -name 'linux*' 2>/dev/null | head -n 1)"
if [ -z "${LINUX_DIR}" ]; then
        echo "PAYLOAD_STATUS:no_linux_build_dir"
        echo "PAYLOAD_OUTPUT_DIR:${OUTPUT_DIR}"
        exit 5
fi

for stamp in \
        .stamp_downloaded \
        .stamp_extracted \
        .stamp_patched \
        .stamp_configured \
        .stamp_built \
        .stamp_staging_installed \
        .stamp_target_installed \
        .stamp_images_installed; do
        touch "${LINUX_DIR}/${stamp}"
done

echo "PAYLOAD_KERNEL_SOURCE_LOG:${KERNEL_SRC_LOG}"
echo "PAYLOAD_KERNEL_STAMP_DIR:${LINUX_DIR}"

(
        cd "${BUILDROOT_DIR}" || exit 97
        CLOCK8002_PREBUILT_KERNEL=1 \
        CLOCK8002_PREBUILT_KERNEL_BUNDLE="${PREBUILT_BUNDLE}" \
        make > "${BUILD_LOG}" 2>&1
)
RC=$?

echo "BUILD_EXIT:${RC}"
if [ "${RC}" -eq 0 ]; then
        echo "PAYLOAD_RESULT:success"
else
        echo "PAYLOAD_RESULT:failed"
fi

exit "${RC}"
