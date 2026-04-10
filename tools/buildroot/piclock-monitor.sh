#!/bin/sh
# piclock-monitor.sh — periodic health check for clock8002 Buildroot image
# Logs to journal via logger -t piclock-monitor; query with:
#   journalctl -t piclock-monitor --no-pager

TIMESTAMP=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

# Service states and restart counts
ALSA_STATE=$(systemctl is-active alsa-ltc 2>/dev/null)
ALSA_RESTARTS=$(systemctl show alsa-ltc --property=NRestarts --value 2>/dev/null)
CLOCK_STATE=$(systemctl is-active clock8002 2>/dev/null)
CLOCK_RESTARTS=$(systemctl show clock8002 --property=NRestarts --value 2>/dev/null)
OLED_STATE=$(systemctl is-active oled_daemon 2>/dev/null)

# USB error count accumulated in dmesg since boot
USB_ERRORS=$(dmesg | grep -c "usb_set_interface failed\|cannot set freq" 2>/dev/null || echo 0)

# CM108 power/control — should always be "on" if udev rule is active
CM108_CONTROL="not-found"
for d in /sys/bus/usb/devices/*/idVendor; do
    v=$(cat "$d" 2>/dev/null)
    if [ "$v" = "0d8c" ]; then
        base="${d%idVendor}"
        CM108_CONTROL=$(cat "${base}power/control" 2>/dev/null || echo "unknown")
    fi
done

# sdl-clock process memory
SDL_PID=$(systemctl show clock8002 --property=MainPID --value 2>/dev/null)
SDL_VMRSS="N/A"
SDL_VMSWAP="N/A"
if [ -n "$SDL_PID" ] && [ "$SDL_PID" != "0" ]; then
    SDL_VMRSS=$(grep VmRSS /proc/"$SDL_PID"/status 2>/dev/null | awk '{print $2$3}' || echo "N/A")
    SDL_VMSWAP=$(grep VmSwap /proc/"$SDL_PID"/status 2>/dev/null | awk '{print $2$3}' || echo "N/A")
fi

# System swap
SYS_SWAP_USED=$(free | awk '/Swap:/{print $3}')
SYS_SWAP_TOTAL=$(free | awk '/Swap:/{print $2}')

# Throttle state (over-temp / under-voltage)
THROTTLE=$(vcgencmd get_throttled 2>/dev/null || echo "unavailable")

logger -t piclock-monitor \
    "[$TIMESTAMP] alsa-ltc=$ALSA_STATE(restarts=$ALSA_RESTARTS) clock8002=$CLOCK_STATE(restarts=$CLOCK_RESTARTS) oled=$OLED_STATE | USB_errors=$USB_ERRORS CM108=$CM108_CONTROL | sdl-clock VmRSS=$SDL_VMRSS VmSwap=$SDL_VMSWAP | swap_used=${SYS_SWAP_USED}kB/${SYS_SWAP_TOTAL}kB | $THROTTLE"
