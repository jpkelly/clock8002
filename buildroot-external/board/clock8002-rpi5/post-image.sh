#!/bin/sh
set -eu

BOARD_DIR="$(dirname "$0")"
GOLDEN_DIR="${BOARD_DIR}/golden-working-card"
BOOT_SOURCE_DIR="${GOLDEN_DIR}/boot"
GOLDEN_ROOT_DIR="${GOLDEN_DIR}/root"
BOARD_NAME="$(basename "${BOARD_DIR}")"
GENIMAGE_CFG="${BOARD_DIR}/genimage-${BOARD_NAME}.cfg"
GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"
BOOT_RUNTIME_FILES="alsa-ltc alsa-ltc_cmd.sh alsa-ltc_pokemon.sh sdl-clock clock_cmd.sh clock_pokemon.sh clock-bridge clock_bridge_cmd.sh clock_bridge_pokemon.sh DejaVuSans.ttf"
VOICE_SRC="${BUILD_DIR}/clock8002-prototype/voices"

RPI_FW_BOOT_DIR=""
for candidate in "${BUILD_DIR}"/rpi-firmware-*/boot; do
	if [ -d "${candidate}" ]; then
		RPI_FW_BOOT_DIR="${candidate}"
		break
	fi
done

if [ ! -d "${BOOT_SOURCE_DIR}" ]; then
	echo "Missing golden working-card boot payload: ${BOOT_SOURCE_DIR}" >&2
	exit 1
fi

# Always sync the copied working-card boot payload into BINARIES_DIR so
# incremental image rebuilds do not reuse stale staged files.
for f in config.txt cmdline.txt; do
	if [ -f "${BOOT_SOURCE_DIR}/${f}" ]; then
		cp -f "${BOOT_SOURCE_DIR}/${f}" "${BINARIES_DIR}/rpi-firmware/${f}"
	elif [ -f "${BOARD_DIR}/${f}" ]; then
		cp -f "${BOARD_DIR}/${f}" "${BINARIES_DIR}/rpi-firmware/${f}"
	fi
done
if [ -f "${BINARIES_DIR}/rpi-firmware/config.txt" ]; then
	cp -f "${BINARIES_DIR}/rpi-firmware/config.txt" "${BINARIES_DIR}/config.txt"
fi
if [ -f "${BINARIES_DIR}/rpi-firmware/cmdline.txt" ]; then
	cp -f "${BINARIES_DIR}/rpi-firmware/cmdline.txt" "${BINARIES_DIR}/cmdline.txt"
fi
for staged in "${BOOT_SOURCE_DIR}"/*; do
	[ -f "${staged}" ] || continue
	case "$(basename "${staged}")" in
		config.txt|cmdline.txt|clock.ini)
			continue
			;;
	esac
	cp -f "${staged}" "${BINARIES_DIR}/$(basename "${staged}")"
done

# Runtime files: always use freshly built binaries from the build dir when
# available.  CLOCK8002_PREBUILT_KERNEL only controls the kernel Image/dtbs;
# it must not gate the application binaries.  Fall back to the golden
# working-card copies only if the build dir does not have the file.
CLOCK8002_BUILD_DIR="${BUILD_DIR}/clock8002-prototype"
for staged in ${BOOT_RUNTIME_FILES}; do
	if [ -f "${CLOCK8002_BUILD_DIR}/${staged}" ]; then
		cp -f "${CLOCK8002_BUILD_DIR}/${staged}" "${BINARIES_DIR}/${staged}"
	elif [ -d "${GOLDEN_ROOT_DIR}" ] && [ -f "${GOLDEN_ROOT_DIR}/${staged}" ]; then
		cp -f "${GOLDEN_ROOT_DIR}/${staged}" "${BINARIES_DIR}/${staged}"
	fi
done

# Default: use prebuilt kernel payload. Custom kernel compile is
# opt-out via CLOCK8002_PREBUILT_KERNEL=0.
if [ "${CLOCK8002_PREBUILT_KERNEL:-1}" != "0" ]; then
	PREBUILT_STORE="${CLOCK8002_PREBUILT_KERNEL_STORE:-/srv/clock8002/prebuilt-kernel-bundles}"
	DEFAULT_BUNDLE="${PREBUILT_STORE}/current"
	BUNDLE_DIR="${CLOCK8002_PREBUILT_KERNEL_BUNDLE:-${DEFAULT_BUNDLE}}"
	if [ -z "${BUNDLE_DIR}" ] || [ ! -d "${BUNDLE_DIR}" ]; then
		echo "Payload mode requires prebuilt kernel bundle: ${BUNDLE_DIR}" >&2
		exit 1
	fi

	if [ -f "${BUNDLE_DIR}/SHA256SUMS" ]; then
		(cd "${BUNDLE_DIR}" && sha256sum -c SHA256SUMS >/dev/null)
	fi

	if [ ! -f "${BUNDLE_DIR}/Image" ]; then
		echo "Plan B bundle missing kernel Image: ${BUNDLE_DIR}/Image" >&2
		exit 1
	fi
	cp -f "${BUNDLE_DIR}/Image" "${BINARIES_DIR}/Image"

	DTB_SRC=""
	if [ -d "${BUNDLE_DIR}/dtbs" ]; then
		DTB_SRC="${BUNDLE_DIR}/dtbs"
	elif [ -d "${BUNDLE_DIR}/dtb" ]; then
		DTB_SRC="${BUNDLE_DIR}/dtb"
	fi

	if [ -z "${DTB_SRC}" ]; then
		echo "Plan B bundle missing dtbs directory (expected dtbs/ or dtb/): ${BUNDLE_DIR}" >&2
		exit 1
	fi

	rm -f "${BINARIES_DIR}"/*.dtb
	DTB_COUNT=0
	for dtb in $(find "${DTB_SRC}" -type f -name '*.dtb'); do
		cp -f "${dtb}" "${BINARIES_DIR}/$(basename "${dtb}")"
		DTB_COUNT=$((DTB_COUNT + 1))
	done
	if [ "${DTB_COUNT}" -eq 0 ]; then
		echo "Plan B bundle dtbs directory contains no .dtb files: ${DTB_SRC}" >&2
		exit 1
	fi

	OVERLAY_SRC="${BUNDLE_DIR}/overlays"
	if [ ! -d "${OVERLAY_SRC}" ]; then
		echo "Plan B bundle missing overlays directory: ${OVERLAY_SRC}" >&2
		exit 1
	fi
	mkdir -p "${BINARIES_DIR}/rpi-firmware/overlays"
	OVERLAY_COUNT=0
	for overlay in "${OVERLAY_SRC}"/*; do
		[ -f "${overlay}" ] || continue
		cp -f "${overlay}" "${BINARIES_DIR}/rpi-firmware/overlays/"
		OVERLAY_COUNT=$((OVERLAY_COUNT + 1))
	done
	if [ "${OVERLAY_COUNT}" -eq 0 ]; then
		echo "Plan B bundle overlays directory contains no files: ${OVERLAY_SRC}" >&2
		exit 1
	fi

	echo "Payload mode: injected prebuilt kernel assets from ${BUNDLE_DIR}" >&2
fi

# Generate genimage config from template when a board-specific cfg is absent.
if [ ! -e "${GENIMAGE_CFG}" ]; then
	GENIMAGE_CFG="${BINARIES_DIR}/genimage.cfg"
	rm -f "${GENIMAGE_CFG}"

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
	KERNEL="$(printf '%s' "${KERNEL}" | tr -d '\r')"
	if [ -z "${KERNEL}" ] && [ -f "${BINARIES_DIR}/Image" ]; then
		KERNEL="Image"
	fi

	if [ -n "${KERNEL}" ]; then
		FILES="${FILES}\t\t\t\"${KERNEL}\",\n"
	fi

	# Stage compressed initramfs for external loading by the Pi firmware.
	# The prebuilt kernel Image may contain an embedded initramfs; the external
	# one takes precedence and ensures TARGET_DIR changes (e.g. init scripts)
	# are actually used at runtime.
	if [ -f "${BINARIES_DIR}/rootfs.cpio" ]; then
		gzip -c "${BINARIES_DIR}/rootfs.cpio" > "${BINARIES_DIR}/rootfs.cpio.gz"
		CPIO_GZ_SIZE=$(stat -c %s "${BINARIES_DIR}/rootfs.cpio.gz" 2>/dev/null || stat -f %z "${BINARIES_DIR}/rootfs.cpio.gz")
		echo "Staged rootfs.cpio.gz (${CPIO_GZ_SIZE} bytes)" >&2
		FILES="${FILES}\t\t\t\"rootfs.cpio.gz\",\n"
	fi

	# Prepare piclock config files (injected via mtools after genimage).
	mkdir -p "${BINARIES_DIR}/piclock"
	CLOCK8002_SRC="${BUILD_DIR}/clock8002-prototype"
	for pair in \
		"${CLOCK8002_SRC}/network.ini.default:network.ini" \
		"${CLOCK8002_SRC}/clock.ini.default:clock.ini" \
		"${CLOCK8002_SRC}/oled/oled.ini:oled.ini"; do
		SRC="${pair%%:*}"
		DST="${pair##*:}"
		[ -f "${SRC}" ] && cp -f "${SRC}" "${BINARIES_DIR}/piclock/${DST}"
	done
	# Stage board-provided piclock files (setup.sh, authorized_keys, etc.).
	if [ -d "${GOLDEN_DIR}/piclock" ]; then
		for f in "${GOLDEN_DIR}/piclock"/*; do
			[ -f "${f}" ] || continue
			cp -f "${f}" "${BINARIES_DIR}/piclock/$(basename "${f}")"
		done
	fi

	# Append extra dev SSH key into piclock/ if BR2_PICLOCKKEY is set at build time.
	# Leave unset for production/release builds. Appends so golden authorized_keys
	# (containing jp@Sapporo.local) is preserved.
	if [ -n "${BR2_PICLOCKKEY:-}" ]; then
		echo "${BR2_PICLOCKKEY}" >> "${BINARIES_DIR}/piclock/authorized_keys"
		chmod 600 "${BINARIES_DIR}/piclock/authorized_keys"
	fi

	# Write build-info.txt into the piclock directory so every image carries its
	# own provenance.  Lands at /boot/piclock/build-info.txt on the device.
	# Use this to identify any SD card and find its matching manifest in git.
	_SOURCE_REPO="$(dirname "${BR2_EXTERNAL_CLOCK8002_PATH}")"
	_COMMIT=$(git -C "${_SOURCE_REPO}" rev-parse HEAD 2>/dev/null || echo "unknown")
	_BRANCH=$(git -C "${_SOURCE_REPO}" branch --show-current 2>/dev/null || echo "unknown")
	_BUNDLE="${CLOCK8002_PREBUILT_KERNEL_BUNDLE:-unknown}"
	_SESSION="${CLOCK8002_BUILD_SESSION:-unknown}"
	_MANIFEST="unknown"
	if [ "${_SESSION}" != "unknown" ]; then
		# Derive manifest name from session: br-<label>-<ts> -> build-manifest-<label>-<ts>.md
		_MANIFEST="build-manifest-${_SESSION#br-}.md"
	fi
	cat > "${BINARIES_DIR}/piclock/build-info.txt" <<BUILDINFO
manifest=${_MANIFEST}
session=${_SESSION}
commit=${_COMMIT}
branch=${_BRANCH}
bundle=${_BUNDLE}
prebuilt_kernel=${CLOCK8002_PREBUILT_KERNEL:-1}
br2_external_version=${BR2_EXTERNAL_CLOCK8002_VERSION:-unknown}
build_timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILDINFO
	unset _SOURCE_REPO _COMMIT _BRANCH _BUNDLE _SESSION _MANIFEST

	# Stage OLED assets to FAT root (logo + daemon binary).
	OLED_SRC="${BUILD_DIR}/clock8002-prototype/oled"
	if [ -f "${OLED_SRC}/piclockLogo.bin" ]; then
		cp -f "${OLED_SRC}/piclockLogo.bin" "${BINARIES_DIR}/piclockLogo.bin"
	fi
	if [ -f "${OLED_SRC}/oled-daemon" ]; then
		cp -f "${OLED_SRC}/oled-daemon" "${BINARIES_DIR}/oled-daemon"
	fi

	# Convert bootsplash PNG to raw RGB565 for direct dd to /dev/fb0 at boot.
	# The embedded initramfs has no fbv/fbi, so we write raw pixels from setup.sh.
	SPLASH_PNG="${BUILD_DIR}/clock8002-prototype/splash/bootsplash.png"
	if [ -f "${SPLASH_PNG}" ] && command -v ffmpeg >/dev/null 2>&1; then
		ffmpeg -y -i "${SPLASH_PNG}" \
			-f rawvideo -pix_fmt rgb565le \
			"${BINARIES_DIR}/bootsplash.raw" \
			>/dev/null 2>&1 && \
			echo "Staged bootsplash.raw ($(stat -c %s "${BINARIES_DIR}/bootsplash.raw") bytes)" >&2 || \
			echo "WARNING: bootsplash.raw conversion failed" >&2
	fi
	# Fix path comment in network.ini (v4 source references Trixie's
	# /boot/firmware/piclock/; on Buildroot the FAT partition mounts at /boot).
	if [ -f "${BINARIES_DIR}/piclock/network.ini" ]; then
		sed -i 's|/boot/firmware/piclock/|/boot/piclock/|g' \
			"${BINARIES_DIR}/piclock/network.ini"
	fi

	sed "s|#BOOT_FILES#|${FILES}|" "${BOARD_DIR}/genimage.cfg.in" > "${GENIMAGE_CFG}"
fi

# genimage copies rootpath to tmppath/root; pass an empty rootpath because
# all boot payloads are already prepared in BINARIES_DIR.
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
	for f in "${BINARIES_DIR}"/piclock/*; do
		[ -f "${f}" ] || continue
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" "${f}" ::piclock/
	done
	for staged in "${BOOT_SOURCE_DIR}"/*; do
		[ -f "${staged}" ] || continue
		case "$(basename "${staged}")" in
			config.txt|cmdline.txt|clock.ini)
				continue
				;;
		esac
		[ -f "${BINARIES_DIR}/$(basename "${staged}")" ] || continue
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
			"${BINARIES_DIR}/$(basename "${staged}")" ::
	done
	for staged in ${BOOT_RUNTIME_FILES}; do
		[ -f "${BINARIES_DIR}/${staged}" ] || continue
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
			"${BINARIES_DIR}/${staged}" ::
	done
	# Inject OLED assets (daemon binary and logo) to FAT root.
	for oled_asset in oled-daemon piclockLogo.bin; do
		[ -f "${BINARIES_DIR}/${oled_asset}" ] || continue
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
			"${BINARIES_DIR}/${oled_asset}" ::
	done

	# Inject bootsplash to FAT root (build artifact, not user config).
	if [ -f "${BINARIES_DIR}/bootsplash.raw" ]; then
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
			"${BINARIES_DIR}/bootsplash.raw" ::
	fi

	# Inject power-button handler to FAT root so setup.sh can copy and start it.
	_PWRBTN_SRC="${BOARD_DIR}/rootfs-overlay/opt/clock8002/power-button.sh"
	if [ -f "${_PWRBTN_SRC}" ]; then
		cp -f "${_PWRBTN_SRC}" "${BINARIES_DIR}/power-button.sh"
		MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
			"${BINARIES_DIR}/power-button.sh" ::
	fi
	unset _PWRBTN_SRC
	if [ -n "${RPI_FW_BOOT_DIR}" ]; then
		for staged in "${RPI_FW_BOOT_DIR}"/*; do
			[ -f "${staged}" ] || continue
			case "$(basename "${staged}")" in
				config.txt|cmdline.txt)
					continue
					;;
			esac
			MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" "${staged}" ::
		done
	fi
	if [ -d "${VOICE_SRC}" ]; then
		MTOOLS_SKIP_CHECK=1 mmd -i "${BOOT_IMG}" ::voices 2>/dev/null || true
		for voice_dir in "${VOICE_SRC}"/*; do
			[ -d "${voice_dir}" ] || continue
			voice_name="$(basename "${voice_dir}")"
			MTOOLS_SKIP_CHECK=1 mmd -i "${BOOT_IMG}" "::voices/${voice_name}" 2>/dev/null || true
			for voice_file in "${voice_dir}"/*; do
				[ -f "${voice_file}" ] || continue
				MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" \
					"${voice_file}" "::voices/${voice_name}/"
			done
		done
	fi

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
			uart0.dtbo \
			uart0-pi5.dtbo \
			uart1.dtbo \
			uart1-pi5.dtbo \
			uart2.dtbo \
			uart2-pi5.dtbo \
			uart3.dtbo \
			uart3-pi5.dtbo; do
			if [ -f "${OVERLAY_SRC}/${dtbo}" ]; then
				MTOOLS_SKIP_CHECK=1 mcopy -o -i "${BOOT_IMG}" "${OVERLAY_SRC}/${dtbo}" ::overlays/
			fi
		done
	fi

	# Patch sdcard.img with the updated boot.vfat (boot partition at LBA 1 = byte 512).
	dd if="${BOOT_IMG}" of="${BINARIES_DIR}/sdcard.img" bs=512 seek=1 conv=notrunc
fi
