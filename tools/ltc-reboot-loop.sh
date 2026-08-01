#!/bin/sh
# Reboot repeatedly until the LTC capture wedges, to build sample size fast.
# Stops and preserves state on the first wedge so forensics stay intact.
#
# NOTE: these are WARM reboots. Memory records that back-to-back warm reboots
# cluster fast failures, so absolute times here are NOT comparable to manual
# cold-boot runs -- only A-vs-B under this same protocol is meaningful.

set -u

CFG=/etc/default/ltc-reboot-loop
STATE=/opt/clock8002/reboot-loop.count
STOP=/opt/clock8002/reboot-loop.stop
RESULTS=/opt/clock8002/logs/reboot-loop-results.csv

MAX_CYCLES=20
WEDGE_TIMEOUT=180
START_TIMEOUT=60
ARM_LABEL="unlabelled"

# Kernel-alternation test (page-size stability comparison). Disabled by
# default so existing single-kernel arms are unaffected.
ALTERNATE_KERNEL=0
CONFIG_TXT=/boot/firmware/config.txt
KERNEL_A=kernel_2712.img   # 16K pages, rpi-2712 flavour
KERNEL_B=kernel8.img       # 4K pages, rpi-v8 flavour
[ -f "$CFG" ] && . "$CFG"

mkdir -p /opt/clock8002/logs
if [ "$ALTERNATE_KERNEL" = "1" ]; then
    [ -f "$RESULTS" ] || echo "cycle,arm,kernel,pagesize,wallclock,launch_uptime,end_uptime,audio_s,outcome" > "$RESULTS"
else
    [ -f "$RESULTS" ] || echo "cycle,arm,wallclock,launch_uptime,end_uptime,audio_s,outcome" > "$RESULTS"
fi

log() { echo "$(date '+%H:%M:%S') $*"; }

# Alternates the config.txt kernel= line for the NEXT boot. Only active when
# ALTERNATE_KERNEL=1. auto_initramfs=1 picks the matching initramfs by name.
flip_kernel() {
    [ "$ALTERNATE_KERNEL" = "1" ] || return 0
    CUR_CFG=$(awk -F= '/^[[:space:]]*kernel=/{print $2; exit}' "$CONFIG_TXT")
    if [ "$CUR_CFG" = "$KERNEL_B" ]; then
        NEXT=$KERNEL_A
    else
        NEXT=$KERNEL_B
    fi
    if grep -q '^kernel=' "$CONFIG_TXT"; then
        sed -i "s/^kernel=.*/kernel=$NEXT/" "$CONFIG_TXT"
    else
        sed -i "/^\[pi5\]$/a kernel=$NEXT" "$CONFIG_TXT"
    fi
    sync
    log "flipped kernel -> $NEXT for next boot (was '$CUR_CFG')"
}

# A wedged controller can stall systemd shutdown (usb_set_interface burns a 5s
# timeout per interface), so force the reboot if the clean path does not land.
do_reboot() {
    flip_kernel
    sync
    sleep 3
    reboot
    sleep 90
    log "clean reboot did not complete in 90s -- forcing"
    sync
    reboot -f
}

if [ -f "$STOP" ]; then
    log "stop flag present ($STOP) -- loop disabled, doing nothing"
    exit 0
fi

CYCLE=$(cat "$STATE" 2>/dev/null || echo 0)
CYCLE=$((CYCLE + 1))
echo "$CYCLE" > "$STATE"
KERNEL_NOW=$(uname -r)
PAGESIZE_NOW=$(getconf PAGESIZE)
log "cycle $CYCLE / $MAX_CYCLES  arm='$ARM_LABEL'  kernel=$KERNEL_NOW  pagesize=$PAGESIZE_NOW"

if [ "$CYCLE" -gt "$MAX_CYCLES" ]; then
    log "reached MAX_CYCLES -- stopping without reboot"
    touch "$STOP"
    exit 0
fi

# Row-prefix helper so every exit path stays in sync with the CSV schema.
row_prefix() {
    if [ "$ALTERNATE_KERNEL" = "1" ]; then
        printf '%s,"%s",%s,%s,' "$CYCLE" "$ARM_LABEL" "$KERNEL_NOW" "$PAGESIZE_NOW"
    else
        printf '%s,"%s",' "$CYCLE" "$ARM_LABEL"
    fi
}

CARD=$(awk '/USB-Audio/{print $1; exit}' /proc/asound/cards 2>/dev/null)
if [ -z "$CARD" ]; then
    log "no USB-Audio card at all -- recording enumeration failure"
    echo "$(row_prefix)$(date '+%F %T'),,$(cut -d' ' -f1 /proc/uptime),0,no_card" >> "$RESULTS"
    do_reboot; exit 0
fi
S=/proc/asound/card$CARD/pcm0c/sub0/status

read_hw() { awk '/hw_ptr/{print $3}' "$S" 2>/dev/null; }

# Wait for the stream to actually start before judging anything.
LAUNCH=""
i=0
while [ $i -lt "$START_TIMEOUT" ]; do
    H=$(read_hw)
    if [ -n "$H" ] && [ "$H" -gt 0 ] 2>/dev/null; then
        LAUNCH=$(cut -d' ' -f1 /proc/uptime)
        log "stream up at uptime ${LAUNCH}s"
        break
    fi
    i=$((i + 1)); sleep 1
done

if [ -z "$LAUNCH" ]; then
    log "stream never started within ${START_TIMEOUT}s -- Fault 1 (open/SET_INTERFACE)"
    echo "$(row_prefix)$(date '+%F %T'),,$(cut -d' ' -f1 /proc/uptime),0,never_started" >> "$RESULTS"
    do_reboot; exit 0
fi

LAST=$(read_hw)
STILL=0
i=0
while [ $i -lt "$WEDGE_TIMEOUT" ]; do
    i=$((i + 1))
    sleep 1
    H=$(read_hw)
    if [ -z "$H" ]; then
        # PCM closed: the app gave up and exited. Counts as a wedge.
        UP=$(cut -d' ' -f1 /proc/uptime)
        A=$(awk -v h="${LAST:-0}" 'BEGIN{printf "%.2f", h/44100}')
        log "PCM closed (app exited) at ${UP}s after ${A}s audio -- WEDGE"
        echo "$(row_prefix)$(date '+%F %T'),$LAUNCH,$UP,$A,wedge_app_exit" >> "$RESULTS"
        sleep 10   # let xhci-wedge-watch finish writing its dump
        do_reboot
        exit 0
    fi
    if [ "$H" = "$LAST" ]; then
        STILL=$((STILL + 1))
        if [ "$STILL" -ge 3 ]; then
            UP=$(cut -d' ' -f1 /proc/uptime)
            A=$(awk -v h="$H" 'BEGIN{printf "%.2f", h/44100}')
            log "hw_ptr frozen at $H (${A}s audio) at uptime ${UP}s -- WEDGE"
            echo "$(row_prefix)$(date '+%F %T'),$LAUNCH,$UP,$A,wedge_hw_frozen" >> "$RESULTS"
            sleep 10   # let xhci-wedge-watch finish writing its dump
            do_reboot
            exit 0
        fi
    else
        STILL=0
        LAST=$H
    fi
done

UP=$(cut -d' ' -f1 /proc/uptime)
A=$(awk -v h="${LAST:-0}" 'BEGIN{printf "%.2f", h/44100}')
log "no wedge in ${WEDGE_TIMEOUT}s (${A}s audio) -- rebooting for next cycle"
echo "$(row_prefix)$(date '+%F %T'),$LAUNCH,$UP,$A,clean_timeout" >> "$RESULTS"
do_reboot
