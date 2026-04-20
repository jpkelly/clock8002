#!/bin/sh
# Power button handler for BusyBox init (Pi 5).
# Reads input events from /dev/input/event0 (pwr_button gpio-keys).
# On KEY_POWER press, initiates clean shutdown via /sbin/poweroff.
#
# input_event on aarch64: 24 bytes
#   bytes 0-15:  timestamp (sec + usec)
#   bytes 16-17: type  (EV_KEY = 0x0001)
#   bytes 18-19: code  (KEY_POWER = 0x0074 = 116)
#   bytes 20-23: value (1 = press, 0 = release)

DEV="/dev/input/event0"

if [ ! -c "$DEV" ]; then
    echo "power-button: $DEV not found" >&2
    exit 1
fi

echo "power-button: watching $DEV for KEY_POWER"

while true; do
    # Read one 24-byte input event
    raw=$(dd if="$DEV" bs=24 count=1 2>/dev/null | hexdump -C | head -2)

    # Check for EV_KEY (01 00) + KEY_POWER (74 00) + press (01 00 00 00)
    # At offset 0x10: 01 00 74 00 01 00 00 00
    if echo "$raw" | grep -q '01 00 74 00 01 00 00 00'; then
        echo "power-button: KEY_POWER pressed — shutting down"
        /sbin/poweroff
        exit 0
    fi
done
