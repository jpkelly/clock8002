#!/bin/sh
# alsa-ltc-monitor.sh — BusyBox-compatible LTC/USB health monitor
#
# Diagnosing alsa-ltc decode failures, USB audio device errors,
# and USB isochronous transfer timeouts on Buildroot (BusyBox) targets.
#
# Tracks per-poll:
#   - alsa-ltc process liveness + PID change restart count
#   - pokemon watchdog liveness
#   - USB audio device presence + authorized state (sysfs)
#   - ALSA card enumeration (USB Audio on /proc/asound/cards)
#   - PCM capture device state (running/prepared/closed)
#   - dmesg USB error count + delta (error -110, timed out, disconnect, etc.)
#   - Recent relevant dmesg lines emitted when new errors appear
#   - alsa-ltc verbose trial run output (stderr capture at startup)
#   - UDP LTC socket presence (port 1245 / 0x04DD)
#   - CPU temperature (millidegrees → °C)
#   - System load average
#   - Available memory
#
# BusyBox compatible: sh, awk, grep, cat, ps, dmesg, sleep
# No bash, no systemctl, no lsusb, no pgrep, no vcgencmd required.
#
# Usage (run on target as root):
#   sh /tmp/alsa-ltc-monitor.sh [interval_seconds]
#
# Default interval: 10 seconds.
# Log: /tmp/alsa-ltc-monitor-YYYYMMDD-HHMMSS.log
#
# View live:
#   tail -f /tmp/alsa-ltc-monitor-*.log
#
# Typically deployed via:
#   scp tools/alsa-ltc-monitor.sh root@<host>:/tmp/
#   ssh root@<host> 'sh /tmp/alsa-ltc-monitor.sh 10 > /tmp/altc-mon.log 2>&1 &'

INTERVAL=${1:-10}
LOG=/tmp/alsa-ltc-monitor-$(date +%Y%m%d-%H%M%S).log

LAST_ALTC_PID=""
RESTART_COUNT=0
LAST_DME_ERR=0

log() { printf '%s\n' "$*" | tee -a "$LOG"; }
ts()  { date '+%Y-%m-%d %H:%M:%S'; }

# ---------------------------------------------------------------------------
# Probe helpers
# ---------------------------------------------------------------------------

get_altc_pid() {
    ps 2>/dev/null | grep "[/]opt/clock8002/alsa-ltc" | head -1 | awk '{print $1}'
}

get_pokemon_pid() {
    ps 2>/dev/null | grep "alsa-ltc_pokemon" | grep -v grep | head -1 | awk '{print $1}'
}

get_usb_audio_card() {
    grep -E "^ *[0-9]" /proc/asound/cards 2>/dev/null \
        | grep -Ei "USB.Audio|USB Audio" \
        | awk '{print $1}' | head -1
}

get_pcm_status() {
    CARD=$1
    [ -z "$CARD" ] && echo "N/A" && return
    STATUS=$(cat /proc/asound/card${CARD}/pcm0c/sub0/status 2>/dev/null | head -1)
    [ -z "$STATUS" ] && echo "no-entry" || echo "$STATUS"
}

get_temp() {
    T=$(cat /sys/class/thermal/thermal_zone0/temp 2>/dev/null)
    [ -z "$T" ] && echo "N/A" && return
    awk -v t="$T" 'BEGIN{printf "%.1fC", t/1000}'
}

get_usb_device_state() {
    # Returns "present/auth=1" or "present/auth=0" or "absent"
    if [ -d /sys/bus/usb/devices/1-1.1 ]; then
        AUTH=$(cat /sys/bus/usb/devices/1-1.1/authorized 2>/dev/null)
        printf "present/auth=%s" "${AUTH:-?}"
    else
        echo "absent"
    fi
}

dmesg_usb_error_count() {
    dmesg 2>/dev/null | grep -cE \
        "error -110|timed out|usb_set_interface failed|cannot set param|Cannot open audio|USB disconnect|usb [0-9]-[0-9].*disconnect"
}

dmesg_recent_usb_lines() {
    # Last 5 relevant dmesg lines
    dmesg 2>/dev/null | grep -E \
        "error -110|timed out|usb_set_interface|disconnect|snd.usb|alsa-ltc|USB Audio|sound" \
        | tail -5
}

check_ltc_udp_socket() {
    # Port 1245 = 0x04DD — check if any process is listening (IPv4 or IPv6)
    IPV4=$(awk 'NR>1 && $2 ~ /:04DD$/ {print "yes"; exit}' /proc/net/udp 2>/dev/null)
    IPV6=$(awk 'NR>1 && $2 ~ /:04DD$/ {print "yes"; exit}' /proc/net/udp6 2>/dev/null)
    if [ "$IPV4" = "yes" ] || [ "$IPV6" = "yes" ]; then
        echo "listening"
    else
        echo "GONE"
    fi
}

# ---------------------------------------------------------------------------
# Startup banner + one-shot verbose probe
# ---------------------------------------------------------------------------

log "==============================================================="
log "  alsa-ltc USB/LTC monitor  started $(ts)"
log "  interval=${INTERVAL}s   log=$LOG"
log "==============================================================="
log ""

log "--- Startup probe ---"
log "ALSA cards:"
cat /proc/asound/cards 2>/dev/null | tee -a "$LOG" || log "(none)"
log ""
log "USB devices (sysfs):"
for d in /sys/bus/usb/devices/[0-9]*; do
    PROD=$(cat "$d/product" 2>/dev/null)
    VID=$(cat "$d/idVendor" 2>/dev/null)
    PID_DEV=$(cat "$d/idProduct" 2>/dev/null)
    AUTH=$(cat "$d/authorized" 2>/dev/null)
    [ -n "$VID" ] && log "  $(basename $d)  ${VID}:${PID_DEV}  auth=${AUTH}  ${PROD}"
done
log ""
log "Initial dmesg USB/audio lines (last 10):"
dmesg 2>/dev/null | grep -E "usb|snd|sound|alsa|audio" | tail -10 | while read LINE; do
    log "  $LINE"
done
log ""
log "Process state at startup:"
ps 2>/dev/null | grep -E "alsa-ltc|sdl3-clock|pokemon" | grep -v grep | while read L; do
    log "  $L"
done
log ""

# Capture a 3-second verbose trial run of alsa-ltc (without killing the watchdog)
# Only if no instance is currently running
TRIAL_PID=$(get_altc_pid)
if [ -z "$TRIAL_PID" ]; then
    log "--- Verbose trial run (3s, no running instance found) ---"
    CARD_NUM=$(get_usb_audio_card)
    if [ -n "$CARD_NUM" ]; then
        TRIAL_OUT=$( /opt/clock8002/alsa-ltc -v plughw:${CARD_NUM},0 255.255.255.255 1245 2>&1 & \
            TPID=$!; sleep 3; kill $TPID 2>/dev/null; wait $TPID 2>/dev/null )
        log "$TRIAL_OUT"
    else
        log "  USB audio card not enumerated — skipping trial run"
    fi
    log ""
fi

log "--- Polling every ${INTERVAL}s ---"
log "$(printf '%-20s %-22s %-8s %-8s %-15s %-14s %-18s %-12s %-10s %-7s %-8s %s' \
    TIMESTAMP ALTC_PID/STATUS RESTART POKEMON USB_DEVICE ALSA_CARD PCM_STATUS UDP_1245 DME_ERR+D TEMP LOAD MEM_MB)"
log "$(printf '%0.s-' $(seq 1 160))"

# ---------------------------------------------------------------------------
# Main poll loop
# ---------------------------------------------------------------------------

while true; do

    TS=$(ts)

    # alsa-ltc PID + restart detection
    ALTC_PID=$(get_altc_pid)
    if [ -z "$ALTC_PID" ]; then
        ALTC_DISP="DEAD"
        if [ -n "$LAST_ALTC_PID" ]; then
            RESTART_COUNT=$((RESTART_COUNT + 1))
        fi
        LAST_ALTC_PID=""
    else
        if [ -n "$LAST_ALTC_PID" ] && [ "$ALTC_PID" != "$LAST_ALTC_PID" ]; then
            RESTART_COUNT=$((RESTART_COUNT + 1))
        fi
        LAST_ALTC_PID="$ALTC_PID"
        ALTC_DISP="pid=$ALTC_PID"
    fi

    # Pokemon watchdog
    POKEMON_PID=$(get_pokemon_pid)
    [ -z "$POKEMON_PID" ] && POKEMON_DISP="DEAD" || POKEMON_DISP="pid=$POKEMON_PID"

    # USB device
    USB_STATE=$(get_usb_device_state)

    # ALSA card
    CARD_NUM=$(get_usb_audio_card)
    if [ -z "$CARD_NUM" ]; then
        ALSA_DISP="ABSENT"
        PCM_DISP="N/A"
    else
        ALSA_DISP="card${CARD_NUM}(USB)"
        PCM_DISP=$(get_pcm_status "$CARD_NUM")
    fi

    # UDP LTC socket
    UDP_DISP=$(check_ltc_udp_socket)

    # dmesg error count + delta
    DME_ERR=$(dmesg_usb_error_count)
    DME_DELTA=$((DME_ERR - LAST_DME_ERR))
    LAST_DME_ERR=$DME_ERR
    DME_DISP="${DME_ERR}(+${DME_DELTA})"

    # Temperature
    TEMP=$(get_temp)

    # Load
    LOAD=$(cut -d' ' -f1-3 /proc/loadavg 2>/dev/null)

    # Memory
    MEM_MB=$(awk '/MemAvailable/{printf "%.0f", $2/1024}' /proc/meminfo 2>/dev/null)

    # Log one-liner
    log "$(printf '%-20s %-22s %-8s %-8s %-15s %-14s %-18s %-12s %-10s %-7s %-8s %s' \
        "$TS" "$ALTC_DISP" "$RESTART_COUNT" "$POKEMON_DISP" \
        "$USB_STATE" "$ALSA_DISP" "$PCM_DISP" "$UDP_DISP" \
        "$DME_DISP" "$TEMP" "$LOAD" "${MEM_MB}MB")"

    # On new dmesg errors, dump the relevant lines
    if [ "$DME_DELTA" -gt 0 ]; then
        log "  !! NEW dmesg errors (delta=$DME_DELTA):"
        dmesg_recent_usb_lines | while read LINE; do
            log "    $LINE"
        done
    fi

    # On PCM closed/dead: dump PCM hw_params for diagnostics
    if [ -n "$CARD_NUM" ] && [ "$PCM_DISP" = "closed" ]; then
        HW=$(cat /proc/asound/card${CARD_NUM}/pcm0c/sub0/hw_params 2>/dev/null | head -1)
        [ -n "$HW" ] && log "  hw_params: $HW"
    fi

    # On alsa-ltc dead: show cmdline of pokemon to confirm it is still looping
    if [ "$ALTC_DISP" = "DEAD" ] && [ "$POKEMON_DISP" != "DEAD" ]; then
        log "  (watchdog alive, alsa-ltc between restarts)"
    fi

    sleep "$INTERVAL"

done
