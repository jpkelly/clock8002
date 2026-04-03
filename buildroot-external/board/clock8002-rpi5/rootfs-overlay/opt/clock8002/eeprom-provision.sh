#!/bin/sh
# eeprom-provision.sh — First-boot EEPROM provisioning for clock8002 Buildroot.
#
# Ensures BOOT_ORDER=0xf1 (SD-only) and BOOT_UART=1 are programmed into the
# Pi 5 EEPROM.  Runs once as a systemd oneshot service, disables itself, and
# reboots to let the firmware apply the staged update.
set -eu

DESIRED_BOOT_ORDER="0xf1"
DESIRED_BOOT_UART="1"
SERVICE_NAME="eeprom-provision.service"
FW_DIR="/usr/lib/firmware/raspberrypi/bootloader-2712/default"

# Read current EEPROM config.
current_boot_order=$(rpi-eeprom-config 2>/dev/null | sed -n 's/^BOOT_ORDER=//p')
current_boot_uart=$(rpi-eeprom-config 2>/dev/null | sed -n 's/^BOOT_UART=//p')

# If already correct, disable service and exit quietly.
if [ "$current_boot_order" = "$DESIRED_BOOT_ORDER" ] && \
   [ "$current_boot_uart" = "$DESIRED_BOOT_UART" ]; then
	echo "EEPROM already provisioned (BOOT_ORDER=${current_boot_order}, BOOT_UART=${current_boot_uart}). Disabling service."
	systemctl disable "$SERVICE_NAME"
	exit 0
fi

echo "EEPROM needs provisioning: BOOT_ORDER=${current_boot_order:-unset} -> ${DESIRED_BOOT_ORDER}, BOOT_UART=${current_boot_uart:-unset} -> ${DESIRED_BOOT_UART}"

# Find the latest firmware blob.
EEPROM_BLOB=$(ls "${FW_DIR}"/pieeprom-*.bin 2>/dev/null | sort | tail -1)
if [ -z "$EEPROM_BLOB" ]; then
	echo "ERROR: No pieeprom-*.bin found in ${FW_DIR}" >&2
	exit 1
fi

# Build custom blob with desired config.
TMPBLOB=$(mktemp /tmp/pieeprom-custom.XXXXXX.bin)
TMPCFG=$(mktemp /tmp/eeprom-cfg.XXXXXX)
trap 'rm -f "$TMPBLOB" "$TMPCFG"' EXIT

cat > "$TMPCFG" <<EOF
[all]
BOOT_UART=${DESIRED_BOOT_UART}
BOOT_ORDER=${DESIRED_BOOT_ORDER}
EOF

rpi-eeprom-config --config "$TMPCFG" --out "$TMPBLOB" "$EEPROM_BLOB"
echo "Built custom EEPROM blob from $(basename "$EEPROM_BLOB")"

# Stage the update for next reboot (-d = use blob's config, -f = force).
rpi-eeprom-update -d -f "$TMPBLOB"
echo "EEPROM update staged. Disabling service and rebooting..."

# Disable service so it doesn't run on subsequent boots.
systemctl disable "$SERVICE_NAME"

# Reboot to let firmware apply the staged EEPROM update.
reboot
