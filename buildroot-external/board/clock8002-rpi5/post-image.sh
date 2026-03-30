#!/bin/sh
set -eu

BOARD_DIR="$(dirname "$0")"
BOARD_NAME="$(basename "${BOARD_DIR}")"
GENIMAGE_CFG="${BOARD_DIR}/genimage-${BOARD_NAME}.cfg"
GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"

# Generate genimage config from template when a board-specific cfg is absent.
if [ ! -e "${GENIMAGE_CFG}" ]; then
	GENIMAGE_CFG="${BINARIES_DIR}/genimage.cfg"
	rm -f "${GENIMAGE_CFG}"

	# Always regenerate top-level boot files to avoid stale settings between
	# incremental builds. Pi 5 should use ttyAMA10 for early serial console.
	cat > "${BINARIES_DIR}/config.txt" << 'EOF'
[all]
arm_64bit=1
kernel=Image
disable_overscan=1
EOF

	cat > "${BINARIES_DIR}/cmdline.txt" << 'EOF'
root=/dev/mmcblk0p2 rootwait console=tty1 console=ttyAMA10,115200
EOF

	FILES=""

	for i in "${BINARIES_DIR}"/*.dtb; do
		[ -e "${i}" ] || continue
		FILES="${FILES}\t\t\t\"${i#${BINARIES_DIR}/}\",\n"
	done

	if [ -d "${BINARIES_DIR}/rpi-firmware" ]; then
		for i in "${BINARIES_DIR}"/rpi-firmware/*; do
			[ -f "${i}" ] || continue
			case "${i}" in
				*.dtb|*/cmdline.txt)
					continue
					;;
			esac
			FILES="${FILES}\t\t\t\"${i#${BINARIES_DIR}/}\",\n"
		done
	fi

	for i in config.txt cmdline.txt; do
		if [ -f "${BINARIES_DIR}/${i}" ]; then
			FILES="${FILES}\t\t\t\"${i}\",\n"
		fi
	done

	KERNEL=""
	if [ -f "${BINARIES_DIR}/rpi-firmware/config.txt" ]; then
		KERNEL="$(sed -n 's/^kernel=//p' "${BINARIES_DIR}/rpi-firmware/config.txt")"
	elif [ -f "${BINARIES_DIR}/config.txt" ]; then
		KERNEL="$(sed -n 's/^kernel=//p' "${BINARIES_DIR}/config.txt")"
	fi
	if [ -z "${KERNEL}" ] && [ -f "${BINARIES_DIR}/Image" ]; then
		KERNEL="Image"
	fi

	if [ -n "${KERNEL}" ]; then
		FILES="${FILES}\t\t\t\"${KERNEL}\",\n"
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
