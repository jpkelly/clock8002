#!/bin/sh
# BusyBox/Buildroot wedge watcher. Python-free counterpart of xhci-wedge-watch.py.
# Ring buffer lives in /tmp (RAM) so the SD card only gets written on a wedge.
# Dumps to /boot because / is a RAM rootfs and is lost on reboot.

set -u

INTERVAL=${INTERVAL:-0.1}
KEEP=${KEEP:-300}
STALL_SAMPLES=${STALL_SAMPLES:-3}
USB_PATH=${USB_PATH:-1-1.1}
PCI=${PCI:-0001:01:00.0}
OUTDIR=${OUTDIR:-/boot/wedge-logs}
RING=/tmp/xhci-ring.csv
HEARTBEAT_EVERY=${HEARTBEAT_EVERY:-3000}

mount | grep -q debugfs || mount -t debugfs none /sys/kernel/debug 2>/dev/null

D=/sys/kernel/debug/usb/xhci/$PCI
[ -d "$D" ] || { echo "no debugfs at $D"; exit 1; }

CARD=$(awk '/USB-Audio/{print $1; exit}' /proc/asound/cards)
[ -n "$CARD" ] || { echo "no USB-Audio card"; exit 1; }
STATUS=/proc/asound/card$CARD/pcm0c/sub0/status

SLOT=""
for s in "$D"/devices/*; do
    [ -f "$s/name" ] || continue
    if [ "$(cat "$s/name")" = "$USB_PATH" ]; then SLOT=$s; break; fi
done
[ -n "$SLOT" ] || { echo "no slot for $USB_PATH"; exit 1; }

EP=""
for e in "$SLOT"/ep*; do
    [ -d "$e" ] || continue
    if grep -q Isoch "$e/trbs" 2>/dev/null; then EP=$e; break; fi
done
[ -n "$EP" ] || { echo "no isoc endpoint in $SLOT"; exit 1; }

sleep "$INTERVAL" 2>/dev/null || INTERVAL=1

mkdir -p "$OUTDIR"
: > "$RING"

echo "watching card=$CARD slot=$SLOT ep=$(basename "$EP") interval=${INTERVAL}s"
echo "dumps -> $OUTDIR"

pidof_ltc() { ps -eo pid,args 2>/dev/null | grep '[a]lsa-ltc ' | awk '{print $1; exit}'; }

sample() {
    UPT=$(cut -d' ' -f1 /proc/uptime)
    MF=$(awk '/MFINDEX/{print $3}' "$D/reg-runtime" 2>/dev/null)
    STS=$(awk '/USBSTS/{print $3}' "$D/reg-op" 2>/dev/null)
    CMD=$(awk '/USBCMD/{print $3}' "$D/reg-op" 2>/dev/null)
    IMAN=$(awk '/IR0_IMAN/{print $3}' "$D/reg-runtime" 2>/dev/null)
    ERDP=$(awk '/IR0_ERDP_LOW/{print $3}' "$D/reg-runtime" 2>/dev/null)
    HW=$(awk '/hw_ptr/{print $3}' "$STATUS" 2>/dev/null)
    APPL=$(awk '/appl_ptr/{print $3}' "$STATUS" 2>/dev/null)
    URB=$(cat "/sys/bus/usb/devices/$USB_PATH/urbnum" 2>/dev/null)
    IRQ=$(awk -v p="$PCI" '$0 ~ p {s=0; for(i=2;i<=5;i++) s+=$i; print s; exit}' /proc/interrupts)
    ENQ=$(cat "$EP/enqueue" 2>/dev/null)
    DEQ=$(cat "$EP/dequeue" 2>/dev/null)
    echo "$UPT,${MF:-?},${STS:-?},${CMD:-?},${IMAN:-?},${ERDP:-?},${HW:-0},${APPL:-0},${URB:-0},${IRQ:-0},${ENQ:-?},${DEQ:-?}"
}

dump() {
    reason=$1
    ts=$(date +%Y%m%d-%H%M%S 2>/dev/null || cut -d. -f1 /proc/uptime)
    f="$OUTDIR/wedge-$ts.log"
    {
        echo "=== xhci wedge capture (BusyBox watcher) ==="
        echo "reason      : $reason"
        echo "wallclock   : $(date 2>/dev/null)"
        echo "uptime      : $(cut -d' ' -f1 /proc/uptime)s"
        echo "kernel      : $(uname -r)"
        echo "chassis     : $(tr -d '\000' < /proc/device-tree/serial-number 2>/dev/null)"
        echo "alsa-ltc    : $(/root/alsa-ltc --version 2>&1 | head -1)"
        echo "running     : $(ps -eo args 2>/dev/null | grep '[a]lsa-ltc ' | head -1)"
        echo "card=$CARD slot=$SLOT ep=$(basename "$EP") interval=${INTERVAL}s"
        echo
        echo "--- ring: uptime,MFINDEX,USBSTS,USBCMD,IMAN,ERDP,hw_ptr,appl_ptr,urbnum,irq,enq,deq ---"
        tail -n "$KEEP" "$RING"
        echo
        echo "--- MFINDEX advancing after wedge? ---"
        i=0
        while [ $i -lt 6 ]; do
            echo "  $(awk '/MFINDEX/{print $3}' "$D/reg-runtime") USBSTS=$(awk '/USBSTS/{print $3}' "$D/reg-op")"
            i=$((i+1)); sleep "$INTERVAL"
        done
        echo
        echo "--- pcm status ---";      cat "$STATUS" 2>/dev/null
        echo "--- pcm hw_params ---";   cat "/proc/asound/card$CARD/pcm0c/sub0/hw_params" 2>/dev/null
        echo "--- stream0 ---";         cat "/proc/asound/card$CARD/stream0" 2>/dev/null
        echo "--- reg-op ---";          cat "$D/reg-op" 2>/dev/null
        echo "--- reg-runtime ---";     cat "$D/reg-runtime" 2>/dev/null
        echo "--- ports ---"
        for p in "$D"/ports/*; do echo "  $(basename "$p"): $(cat "$p/portsc" 2>/dev/null)"; done
        echo "--- rings ---"
        for r in command-ring event-ring; do
            for k in enqueue dequeue cycle; do
                echo "  $r/$k: $(cat "$D/$r/$k" 2>/dev/null)"
            done
        done
        echo "--- slot-context ---";    cat "$SLOT/slot-context" 2>/dev/null
        echo "--- ep-context (active only) ---"
        grep -v INVALID "$SLOT/ep-context" 2>/dev/null
        echo "--- isoc trbs ---";       head -20 "$EP/trbs" 2>/dev/null
        echo "--- interrupts ---";      grep -E "$PCI|xhci" /proc/interrupts
        echo "--- lsusb ---";           lsusb 2>/dev/null
        echo "--- dmesg tail ---";      dmesg | tail -60
    } > "$f" 2>&1
    sync
    echo "WEDGE ($reason) -> $f"
}

LASTHW=""
DUMPED_HW=""
STILL=0
N=0
LASTPID=$(pidof_ltc)
echo "baseline alsa-ltc pid: ${LASTPID:-none}"

while :; do
    LINE=$(sample)
    echo "$LINE" >> "$RING"
    N=$((N+1))

    if [ $((N % 600)) -eq 0 ]; then
        tail -n "$KEEP" "$RING" > "$RING.tmp" && mv "$RING.tmp" "$RING"
    fi
    if [ $((N % HEARTBEAT_EVERY)) -eq 0 ]; then
        echo "alive uptime=$(cut -d' ' -f1 /proc/uptime)s samples=$N pid=$(pidof_ltc)" >> "$OUTDIR/heartbeat.log"
        sync
    fi

    HW=$(echo "$LINE" | cut -d, -f7)
    if [ "$HW" = "0" ]; then
        # PCM not open yet -- absent hw_ptr must not read as frozen
        STILL=0
        LASTHW=""
    elif [ "$HW" = "$LASTHW" ]; then
        STILL=$((STILL+1))
        if [ "$STILL" -eq "$STALL_SAMPLES" ] && [ "$DUMPED_HW" != "$HW" ]; then
            dump "hw_ptr frozen at $HW"
            DUMPED_HW=$HW
        fi
    else
        STILL=0
        LASTHW=$HW
    fi

    PID=$(pidof_ltc)
    if [ "${PID:-none}" != "${LASTPID:-none}" ]; then
        echo "$(date 2>/dev/null) pid ${LASTPID:-none} -> ${PID:-none} at uptime $(cut -d' ' -f1 /proc/uptime)s" >> "$OUTDIR/respawn.log"
        sync
        dump "alsa-ltc respawn ${LASTPID:-none} -> ${PID:-none}"
        LASTPID=$PID
    fi

    sleep "$INTERVAL"
done
