#!/bin/sh
# Multi-boot LTC stability harness v2.
# Waits for the system to fully settle (LTC stream RUNNING *and* display up)
# before starting the measurement window, so each cycle exercises the real
# production steady state rather than just the early-boot window.
# Config: /etc/default/ltc-cycle

BASE=/opt/clock8002
LOGDIR=$BASE/logs
COUNT=$BASE/ltc-cycle.count
STOP=$BASE/ltc-cycle.stop

MAX_CYCLES=10
MONITOR_S=300
STREAM_TIMEOUT=90
DISPLAY_TIMEOUT=90
LABEL="unlabelled"
RESULTS=$LOGDIR/ltc-cycle-results.csv
[ -f /etc/default/ltc-cycle ] && . /etc/default/ltc-cycle

mkdir -p "$LOGDIR"
[ -f "$RESULTS" ] || echo "cycle,label,timestamp,kernel,pagesize,ltc_sha,boot_to_stream_s,boot_to_display_s,monitor_start_s,monitored_s,outcome,gaps,frames,ltc_decoded,errors" > "$RESULTS"

n=$(cat "$COUNT" 2>/dev/null || echo 0)

if [ -f "$STOP" ]; then
    echo "stop flag present - halting"
    systemctl disable ltc-cycle.service >/dev/null 2>&1
    exit 0
fi
if [ "$n" -ge "$MAX_CYCLES" ]; then
    echo "all $MAX_CYCLES cycles complete - disabling self"
    systemctl disable ltc-cycle.service >/dev/null 2>&1
    exit 0
fi

n=$((n + 1))
echo "$n" > "$COUNT"
sync

kern=$(uname -r)
psize=$(getconf PAGESIZE)
sha=$(sha256sum /opt/clock8002/alsa-ltc 2>/dev/null | cut -c1-8)

card=$(awk '/USB Audio/{print $1; exit}' /proc/asound/cards 2>/dev/null)
[ -z "$card" ] && card=0
STATUS=/proc/asound/card$card/pcm0c/sub0/status

read_hw()    { awk '/hw_ptr/{print $3}' "$STATUS" 2>/dev/null; }
is_running() { grep -q 'state: RUNNING' "$STATUS" 2>/dev/null; }
app_alive()  { ps -eo args | grep -q '[a]lsa-ltc'; }
up_s()       { cut -d. -f1 /proc/uptime; }

row() {
    echo "$n,\"$LABEL\",$(date '+%Y-%m-%d %H:%M:%S'),$kern,$psize,$sha,$1,$2,$3,$4,$5,$6,$7,$8,$9" >> "$RESULTS"
    sync
}

finish() {
    echo "cycle $n: $1"
    [ "$1" = "clean" ] || sleep 10
    if [ "$n" -ge "$MAX_CYCLES" ]; then
        echo "final cycle done - disabling self"
        systemctl disable ltc-cycle.service >/dev/null 2>&1
        sync
        exit 0
    fi
    sync
    reboot
    exit 0
}

# --- settle phase 1: LTC stream must be RUNNING ---
w=0
while [ "$w" -lt "$STREAM_TIMEOUT" ]; do
    is_running && break
    sleep 1; w=$((w + 1))
done
if ! is_running; then
    row "" "" "" 0 never_started 0 "" "" ""
    finish never_started
fi
boot_to_stream=$(up_s)

# --- settle phase 2: display service must be up (real production state) ---
w=0
while [ "$w" -lt "$DISPLAY_TIMEOUT" ]; do
    [ "$(systemctl is-active clock8002 2>/dev/null)" = "active" ] && break
    sleep 1; w=$((w + 1))
done
boot_to_display=$(up_s)
sleep 5   # let the display finish initialising before measuring

monitor_start=$(up_s)
outcome=clean
gaps=0
prev=$(read_hw)
same=0
elapsed=0

while [ "$elapsed" -lt "$MONITOR_S" ]; do
    sleep 1
    elapsed=$((elapsed + 1))

    if ! app_alive; then outcome=wedge_app_exit; break; fi

    cur=$(read_hw)
    if [ -z "$cur" ]; then outcome=wedge_pcm_gone; break; fi

    if [ "$cur" = "$prev" ]; then
        same=$((same + 1))
        [ "$same" -eq 1 ] && gaps=$((gaps + 1))
        # 2 consecutive identical 1s samples = >=2s freeze at 44.1kHz
        if [ "$same" -ge 2 ]; then outcome=wedge_hw_frozen; break; fi
    else
        same=0
    fi
    prev=$cur
done

hb=$(journalctl -u alsa-ltc -b --no-pager 2>/dev/null | grep '\[heartbeat\]' | tail -1)
frames=$(echo "$hb" | sed -n 's/.*\[heartbeat\] \([0-9]*\) frames.*/\1/p')
ltc=$(echo "$hb"    | sed -n 's/.*frames, \([0-9]*\) ltc.*/\1/p')
errs=$(echo "$hb"   | sed -n 's/.*decoded, \([0-9]*\) errors.*/\1/p')

row "$boot_to_stream" "$boot_to_display" "$monitor_start" "$elapsed" "$outcome" "$gaps" "$frames" "$ltc" "$errs"
finish "$outcome (stream ${boot_to_stream}s, display ${boot_to_display}s, monitored ${elapsed}s, 1s-gaps=$gaps)"
