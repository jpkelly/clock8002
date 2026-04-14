#!/bin/bash
# vl805-usb-health-monitor.sh — per-minute VL805/USB/LTC health monitor
#
# Logs every 60 seconds to ~/monitor.log.
# Fields: timestamp, temp, load, alsa-ltc restarts, USB -110 error count,
#         VIA hub presence, CM108 audio dongle presence, sdl-clock RSS,
#         available memory, throttle state.
#
# Designed for diagnosing VL805 isochronous USB failures (usb_set_interface -110).
# usb-hub and usb-audio drop to 0 if devices fall off the bus entirely.
# throttle captures under-voltage and ARM throttle events (sticky since boot).
#
# Usage (Trixie, run as pi):
#   scp tools/vl805-usb-health-monitor.sh pi@<host>:~/monitor.sh
#   ssh pi@<host> 'nohup bash ~/monitor.sh > /dev/null 2>&1 &'
#
# View log:
#   ssh pi@<host> 'tail -20 ~/monitor.log'
#
# Note: does NOT survive reboot — restart manually after power cycle.

LOG=~/monitor.log

echo "=== vl805-usb-health-monitor started $(date) ===" >> "$LOG"

while true; do
  TS=$(date '+%Y-%m-%d %H:%M:%S')
  TEMP=$(vcgencmd measure_temp | sed 's/temp=//')
  LOAD=$(cut -d' ' -f1-3 /proc/loadavg)
  RESTARTS=$(systemctl show alsa-ltc -p NRestarts --value 2>/dev/null || echo "?")
  USB_ERRORS=$(dmesg | grep -E "error -110|usb_set_interface failed" | wc -l)
  USB_HUB=$(lsusb | grep -c "2109:3431")
  USB_AUDIO=$(lsusb | grep -c "0d8c:0014")
  RSS=$(ps -p $(pgrep -f sdl-clock | head -1) -o rss= 2>/dev/null | tr -d ' ')
  MEM_AVAIL=$(awk '/MemAvailable/ {printf "%.0f", $2/1024}' /proc/meminfo)
  THROTTLE=$(vcgencmd get_throttled | sed 's/throttled=//')
  echo "$TS temp=$TEMP load=$LOAD alsa-restarts=$RESTARTS usb-errors=$USB_ERRORS usb-hub=$USB_HUB usb-audio=$USB_AUDIO rss=${RSS}kB mem-avail=${MEM_AVAIL}MB throttle=$THROTTLE" >> "$LOG"
  sleep 60
done
