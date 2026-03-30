#!/bin/sh
set -eu

BOARD_DIR="$(dirname "$0")"
BOARD_NAME="$(basename "${BOARD_DIR}")"
GENIMAGE_CFG="${BOARD_DIR}/genimage-${BOARD_NAME}.cfg"
GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"

# Generate genimage config from template when a board-specific cfg is absent.
if [ ! -e "${GENIMAGE_CFG}" ]; then
	GENIMAGE_CFG="${BINARIES_DIR}/genimage.cfg"
	FILES=""

	for i in "${BINARIES_DIR}"/*.dtb; do
		FILES="${FILES}\t\t\t\"${i#${BINARIES_DIR}/}\",\n"
	done

	# Buildroot's Pi5 firmware payload does not always provide a top-level
	# config.txt; create one so the SD image has deterministic boot settings.
	if [ ! -f "${BINARIES_DIR}/config.txt" ]; then
		cat > "${BINARIES_DIR}/config.txt" << 'EOF'
[all]
arm_64bit=1
kernel=Image
EOF
	fi
	FILES="${FILES}\t\t\t\"config.txt\",\n"

	# Ensure cmdline.txt is present at boot partition root.
	if [ ! -f "${BINARIES_DIR}/cmdline.txt" ] && [ -f "${BINARIES_DIR}/rpi-firmware/cmdline.txt" ]; then
		cp -f "${BINARIES_DIR}/rpi-firmware/cmdline.txt" "${BINARIES_DIR}/cmdline.txt"
	fi
	if [ -f "${BINARIES_DIR}/cmdline.txt" ]; then
		FILES="${FILES}\t\t\t\"cmdline.txt\",\n"
	fi

	if [ -f "${BINARIES_DIR}/Image" ]; then
		FILES="${FILES}\t\t\t\"Image\",\n"
	fi

	sed "s|#BOOT_FILES#|${FILES}|" "${BOARD_DIR}/genimage.cfg.in" > "${GENIMAGE_CFG}"
fi

# genimage copies rootpath to tmppath/root; pass an empty rootpath to avoid
# redundant copy because rootfs image is already built as rootfs.ext4.
trap 'rm -rf "${ROOTPATH_TMP}"' EXIT
ROOTPATH_TMP="$(mktemp -d)"

rm -rf "${GENIMAGE_TMP}"

genimage \
	--rootpath "${ROOTPATH_TMP}" \
	--tmppath "${GENIMAGE_TMP}" \
	--inputpath "${BINARIES_DIR}" \
	--outputpath "${BINARIES_DIR}" \
	--config "${GENIMAGE_CFG}"
