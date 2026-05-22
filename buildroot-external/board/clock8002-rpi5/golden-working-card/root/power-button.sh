#!/bin/sh
# Power button handler for BusyBox init (Pi 5).
# Detects KEY_POWER events and triggers clean shutdown by default.

set -u

LOG_TAG="power-button"
DEV="/dev/input/by-path/platform-pwr_button-event"
ACTION="${POWER_BUTTON_ACTION:-poweroff}"

resolve_fallback_device() {
	awk '
		/^N: Name="pwr_button"/ { hit=1 }
		hit && /^H: Handlers=/ {
			for (i = 1; i <= NF; i++) {
				if ($i ~ /^event[0-9]+$/) {
					sub(/^event/, "", $i)
					print "/dev/input/event" $i
					exit
				}
			}
		}
	' /proc/bus/input/devices
}

if [ ! -e "$DEV" ]; then
	FALLBACK_DEV="$(resolve_fallback_device)"
	if [ -n "${FALLBACK_DEV:-}" ]; then
		DEV="$FALLBACK_DEV"
	fi
fi

if [ ! -e "$DEV" ]; then
	echo "$LOG_TAG: no power button input device found" >&2
	exit 1
fi

echo "$LOG_TAG: watching $DEV for KEY_POWER"

while true; do
	raw="$(dd if="$DEV" bs=24 count=1 2>/dev/null | hexdump -C | head -2)"
	if echo "$raw" | grep -q '01 00 74 00 01 00 00 00'; then
		echo "$LOG_TAG: KEY_POWER pressed"
		if [ "$ACTION" = "poweroff" ]; then
			echo "$LOG_TAG: shutting down"
			/sbin/poweroff
			exit 0
		fi
		echo "$LOG_TAG: ACTION=$ACTION, not powering off"
	fi
done
