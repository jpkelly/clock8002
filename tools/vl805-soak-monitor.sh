#!/bin/bash
# vl805-soak-monitor.sh — continuous soak test monitor for VL805/USB/LTC health
#
# Polls every 30s and logs to /tmp/vl805-monitor.log (and stdout).
# Alerts on: xHCI error increases, alsa-ltc restart increases, LTC gap events.
#
# Usage:
#   scp tools/vl805-soak-monitor.sh pi@<host>:/tmp/
#   ssh pi@<host> 'nohup /tmp/vl805-soak-monitor.sh &'
#
# View log remotely:
#   ssh pi@<host> 'tail -20 /tmp/vl805-monitor.log'
#
# Works on both Trixie (bash) and Buildroot (bash or sh with minor edits).

LOG=/tmp/vl805-monitor.log
INTERVAL=30

echo "=== VL805 soak monitor started $(date) ===" | tee -a "$LOG"

PREV_XHCI=0
PREV_RESTARTS=0
PREV_GAPS=0

while true; do
    NOW=$(date "+%Y-%m-%d %H:%M:%S")
    UP=$(uptime -p)
    XHCI=$(dmesg | grep -c "usb_set_interface")
    RESTARTS=$(systemctl show alsa-ltc --property=NRestarts --value)
    LTC_STATE=$(systemctl is-active alsa-ltc)
    GAPS=$(journalctl -u alsa-ltc --no-pager 2>/dev/null | grep -c "\[gap\]")
    LAST_HB=$(journalctl -u alsa-ltc --no-pager 2>/dev/null | grep heartbeat | tail -1 | sed "s/.*heartbeat] //")
    TEMP=$(vcgencmd measure_temp 2>/dev/null | sed "s/temp=//")

    LINE="$NOW | up: $UP | xhci_err: $XHCI | ltc: $LTC_STATE | restarts: $RESTARTS | gaps: $GAPS | $LAST_HB | $TEMP"
    echo "$LINE" | tee -a "$LOG"

    if [ "$XHCI" -gt "$PREV_XHCI" ]; then
        echo "*** ALERT: xHCI errors increased $PREV_XHCI -> $XHCI ***" | tee -a "$LOG"
        dmesg | grep "usb_set_interface" | tail -5 >> "$LOG"
    fi
    if [ "$RESTARTS" -gt "$PREV_RESTARTS" ]; then
        echo "*** ALERT: alsa-ltc restarts increased $PREV_RESTARTS -> $RESTARTS ***" | tee -a "$LOG"
    fi
    if [ "$GAPS" -gt "$PREV_GAPS" ]; then
        echo "*** ALERT: LTC gap detected ($PREV_GAPS -> $GAPS) ***" | tee -a "$LOG"
        journalctl -u alsa-ltc --no-pager 2>/dev/null | grep "\[gap\]" | tail -3 >> "$LOG"
    fi

    PREV_XHCI=$XHCI
    PREV_RESTARTS=$RESTARTS
    PREV_GAPS=$GAPS
    sleep $INTERVAL
done
