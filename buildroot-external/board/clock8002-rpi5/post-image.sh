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

	# Always sync board config/cmdline over the rpi-firmware cached copies
	# so image rebuilds pick up changes without rpi-firmware-dirclean.
	for f in config.txt cmdline.txt; do
		if [ -f "${BOARD_DIR}/${f}" ]; then
			cp -f "${BOARD_DIR}/${f}" "${BINARIES_DIR}/rpi-firmware/${f}"
		fi
	done

	# Promote rpi-firmware config/cmdline to the top-level binaries dir so
	# genimage places them at the root of the FAT partition (/boot/firmware/).
	if [ -f "${BINARIES_DIR}/rpi-firmware/config.txt" ]; then
		cp -f "${BINARIES_DIR}/rpi-firmware/config.txt" "${BINARIES_DIR}/config.txt"
	fi
	if [ -f "${BINARIES_DIR}/rpi-firmware/cmdline.txt" ]; then
		cp -f "${BINARIES_DIR}/rpi-firmware/cmdline.txt" "${BINARIES_DIR}/cmdline.txt"
	fi

	FILES=""

	for i in "${BINARIES_DIR}"/*.dtb; do
		[ -e "${i}" ] || continue
		FILES="${FILES}\t\t\t\"${i#${BINARIES_DIR}/}\",\n"
	done

	if [ -d "${BINARIES_DIR}/rpi-firmware" ]; then
		for i in "${BINARIES_DIR}"/rpi-firmware/*; do
			case "${i}" in
				*.dtb|*/cmdline.txt|*/config.txt)
					continue
					;;
			esac
			if [ -f "${i}" ]; then
				FILES="${FILES}\t\t\t\"${i#${BINARIES_DIR}/}\",\n"
			fi
		done
		# Overlays are injected via mtools after genimage (see below)
		# so the firmware finds them at /overlays/ on the FAT root.
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

	# Prepare piclock config files (injected via mtools after genimage).
	mkdir -p "${BINARIES_DIR}/piclock"
	CLOCK8002_SRC="${BUILD_DIR}/clock8002-prototype"
	for pair in \
		"${BOARD_DIR}/network.ini:network.ini" \
		"${CLOCK8002_SRC}/clock.ini.default:clock.ini" \
		"${CLOCK8002_SRC}/oled/oled.ini:oled.ini"; do
		SRC="${pair%%:*}"
		DST="${pair##*:}"
		[ -f "${SRC}" ] && cp -f "${SRC}" "${BINARIES_DIR}/piclock/${DST}"
	done

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

# Inject piclock/ config files into the FAT boot partition via mtools.
# genimage's mcopy doesn't auto-create subdirectories, so we do it manually.
# We modify boot.vfat in-place, then dd it back into sdcard.img at the boot
# partition offset to avoid a second genimage run which would recreate
# boot.vfat from scratch.  Partition starts at LBA 1 (byte 512).
BOOT_IMG="${BINARIES_DIR}/boot.vfat"
if [ -d "${BINARIES_DIR}/piclock" ] && [ -f "${BOOT_IMG}" ]; then
	MTOOLS_SKIP_CHECK=1 mmd -i "${BOOT_IMG}" ::piclock 2>/dev/null || true
	for ini in "${BINARIES_DIR}"/piclock/*.ini; do
		[ -f "${ini}" ] || continue
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" "${ini}" ::piclock/
	done

	# Compile custom gpu-enable overlay (D0-stepping GPU compatible fix).
	GPU_OVERLAY_SRC="${BOARD_DIR}/gpu-enable.dts"
	OVERLAY_SRC="${BINARIES_DIR}/rpi-firmware/overlays"
	if [ -f "${GPU_OVERLAY_SRC}" ] && [ -d "${OVERLAY_SRC}" ]; then
		"${HOST_DIR}/bin/dtc" -@ -I dts -O dtb \
			-o "${OVERLAY_SRC}/gpu-enable.dtbo" \
			"${GPU_OVERLAY_SRC}"
	fi

	# Inject DT overlays into /overlays/ on the boot FAT partition.
	# Pi 5 firmware expects overlays at the FAT root, not in rpi-firmware/.
	# Only copy the overlays we actually reference in config.txt, plus
	# overlay_map.dtb (firmware overlay name resolution) and bcm2712d0.dtbo
	# (D0-stepping fixups applied by firmware before Linux boots).
	if [ -d "${OVERLAY_SRC}" ]; then
		MTOOLS_SKIP_CHECK=1 mmd -i "${BOOT_IMG}" ::overlays 2>/dev/null || true
		for dtbo in \
			overlay_map.dtb \
			bcm2712d0.dtbo \
			vc4-kms-v3d-pi5.dtbo \
			gpu-enable.dtbo \
			dwc2.dtbo \
			uart1.dtbo \
			uart2.dtbo \
			uart3.dtbo; do
			if [ -f "${OVERLAY_SRC}/${dtbo}" ]; then
				MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" "${OVERLAY_SRC}/${dtbo}" ::overlays/
			fi
		done
	fi

	# Patch sdcard.img with the updated boot.vfat (boot partition at LBA 1 = byte 512).
	dd if="${BOOT_IMG}" of="${BINARIES_DIR}/sdcard.img" bs=512 seek=1 conv=notrunc
fi
