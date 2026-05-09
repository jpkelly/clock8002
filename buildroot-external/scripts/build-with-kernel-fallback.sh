#!/bin/sh
# Build wrapper: Mode A first, Mode B only on kernel-path failure.
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
LOG_A="/tmp/br-modeA-${STAMP}.log"
LOG_B="/tmp/br-modeB-${STAMP}.log"
HASH_LOG="/tmp/br-modeB-${STAMP}.sha256.log"
VERIFY_LOG="/tmp/br-modeB-${STAMP}.verify.log"

if [ ! -f "${BUILDROOT_DIR}/Makefile" ]; then
        echo "ERROR: Buildroot not found at ${BUILDROOT_DIR}" >&2
        exit 2
fi

if [ -z "${PREBUILT_BUNDLE}" ] && [ -d "${PREBUILT_CURRENT}" ]; then
        PREBUILT_BUNDLE="${PREBUILT_CURRENT}"
fi

kernel_failure_detected() {
        LOG_FILE="$1"
        tail -n 500 "${LOG_FILE}" | grep -qiE \
                '>>>[[:space:]]+linux|linux-custom|vmlinux|arch/arm64|scripts/kconfig|Image|modules_install'
        return $?
}

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

echo "MODE_A_LOG:${LOG_A}"
(
        cd "${BUILDROOT_DIR}" || exit 97
        make > "${LOG_A}" 2>&1
)
RC_A=$?

if [ "${RC_A}" -eq 0 ]; then
        echo "BUILD_MODE:A"
        echo "BUILD_EXIT:0"
        exit 0
fi

echo "BUILD_MODE:A"
echo "BUILD_EXIT:${RC_A}"

if [ -z "${PREBUILT_BUNDLE}" ]; then
        echo "PLANB_STATUS:disabled_no_bundle"
        echo "PLANB_BUNDLE_DEFAULT:${PREBUILT_CURRENT}"
        exit "${RC_A}"
fi

if ! kernel_failure_detected "${LOG_A}"; then
        echo "PLANB_STATUS:skipped_non_kernel_failure"
        exit "${RC_A}"
fi

if [ -x "${VERIFY_SCRIPT}" ]; then
        "${VERIFY_SCRIPT}" "${PREBUILT_BUNDLE}" > "${VERIFY_LOG}" 2>&1
        RC_VERIFY=$?
        if [ "${RC_VERIFY}" -ne 0 ]; then
                echo "PLANB_STATUS:invalid_bundle"
                echo "PLANB_BUNDLE:${PREBUILT_BUNDLE}"
                echo "PLANB_VERIFY_LOG:${VERIFY_LOG}"
                exit "${RC_A}"
        fi
else
        if ! bundle_looks_valid "${PREBUILT_BUNDLE}"; then
                echo "PLANB_STATUS:invalid_bundle"
                echo "PLANB_BUNDLE:${PREBUILT_BUNDLE}"
                exit "${RC_A}"
        fi

        if [ -f "${PREBUILT_BUNDLE}/SHA256SUMS" ]; then
                (
                        cd "${PREBUILT_BUNDLE}" || exit 98
                        sha256sum -c SHA256SUMS > "${HASH_LOG}" 2>&1
                )
                RC_HASH=$?
                if [ "${RC_HASH}" -ne 0 ]; then
                        echo "PLANB_STATUS:bundle_hash_failed"
                        echo "PLANB_HASH_LOG:${HASH_LOG}"
                        exit "${RC_A}"
                fi
        fi
fi

LINUX_DIR="$(find "${OUTPUT_DIR}/build" -maxdepth 1 -type d -name 'linux*' 2>/dev/null | head -n 1)"
if [ -z "${LINUX_DIR}" ]; then
        echo "PLANB_STATUS:no_linux_build_dir"
        echo "PLANB_OUTPUT_DIR:${OUTPUT_DIR}"
        exit "${RC_A}"
fi

# Mark kernel package as completed so retry can proceed to post-build/post-image,
# where prebuilt kernel assets are injected under CLOCK8002_PREBUILT_KERNEL=1.
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

echo "PLANB_STATUS:enabled"
echo "PLANB_BUNDLE:${PREBUILT_BUNDLE}"
if [ "${PREBUILT_BUNDLE}" = "${PREBUILT_CURRENT}" ]; then
        echo "PLANB_BUNDLE_SOURCE:default_current"
fi
echo "PLANB_STAMP_DIR:${LINUX_DIR}"
echo "MODE_B_LOG:${LOG_B}"

(
        cd "${BUILDROOT_DIR}" || exit 97
        CLOCK8002_PREBUILT_KERNEL=1 \
        CLOCK8002_PREBUILT_KERNEL_BUNDLE="${PREBUILT_BUNDLE}" \
        make > "${LOG_B}" 2>&1
)
RC_B=$?

echo "BUILD_MODE:B"
echo "BUILD_EXIT:${RC_B}"

if [ "${RC_B}" -eq 0 ]; then
        echo "PLANB_RESULT:success"
else
        echo "PLANB_RESULT:failed"
fi

exit "${RC_B}"
