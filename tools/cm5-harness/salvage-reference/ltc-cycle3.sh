#!/bin/sh
# Multi-boot LTC stability harness — PRODUCTION CONDITIONS ONLY.
# No artificial load: the original failures occurred with no known memory or CPU
# pressure, so nothing is added here that was not present then.
# Each cycle: boot -> wait for LTC stream AND display up -> settle -> measure.
# Config: /etc/default/ltc-cycle

BASE=/opt/clock8002
LOGDIR=$BASE/logs
COUNT=$BASE/ltc-cycle.count
STOP=$BASE/ltc-cycle.stop

MAX_CYCLES=20
MONITOR_S=240
STREAM_TIMEOUT=90
DISPLAY_TIMEOUT=120
LABEL="unlabelled"
RESULTS=$LOGDIR/ltc-cycle-results.csv
[ -f /etc/default/ltc-cycle ] && . /etc/default/ltc-cycle

mkdir -p "$LOGDIR"
[ -f "$RESULTS" ] || echo "cycle,label,timestamp,kernel,pagesize,ltc_sha,boot_to_stream_s,boot_to_display_s,display_ok,monitored_s,outcome,gaps_1s,avail_mb,swap_mb,frames,ltc_decoded,errors" > "$RESULTS"

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
avail_mb()   { awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo; }
swap_mb()    { awk '/SwapTotal/{t=$2} /SwapFree/{f=$2} END{print int((t-f)/1024)}' /proc/meminfo; }

row() {
    echo "$n,\"$LABEL\",$(date '+%Y-%m-%d %H:%M:%S'),$kern,$psize,$sha,$1,$2,$3,$4,$5,$6,$7,$8,$9,${10},${11}" >> "$RESULTS"
    sync
}

finish() {
    echo "cycle $n: $1"
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

# --- settle 1: LTC stream RUNNING ---
w=0
while [ "$w" -lt "$STREAM_TIMEOUT" ]; do
    is_running && break
    sleep 1; w=$((w + 1))
done
if ! is_running; then
    row "" "" no 0 never_started 0 "$(avail_mb)" "$(swap_mb)" "" "" ""
    finish never_started
fi
boot_to_stream=$(up_s)

# --- settle 2: display up. Requires this unit to be Type=simple, otherwise
# --- multi-user.target stays blocked and clock8002 can never start.
w=0
display_ok=no
while [ "$w" -lt "$DISPLAY_TIMEOUT" ]; do
    if [ "$(systemctl is-active clock8002 2>/dev/null)" = "active" ]; then
        display_ok=yes; break
    fi
    sleep 1; w=$((w + 1))
done
boot_to_display=$(up_s)
sleep 5

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
        if [ "$same" -ge 2 ]; then outcome=wedge_hw_frozen; break; fi
    else
        same=0
    fi
    prev=$cur
done

[ "$outcome" != "clean" ] && sleep 10

hb=$(journalctl -u alsa-ltc -b --no-pager 2>/dev/null | grep '\[heartbeat\]' | tail -1)
frames=$(echo "$hb" | sed -n 's/.*\[heartbeat\] \([0-9]*\) frames.*/\1/p')
ltc=$(echo "$hb"    | sed -n 's/.*frames, \([0-9]*\) ltc.*/\1/p')
errs=$(echo "$hb"   | sed -n 's/.*decoded, \([0-9]*\) errors.*/\1/p')

row "$boot_to_stream" "$boot_to_display" "$display_ok" "$elapsed" "$outcome" "$gaps" "$(avail_mb)" "$(swap_mb)" "$frames" "$ltc" "$errs"
finish "$outcome (stream ${boot_to_stream}s, display=${display_ok} @${boot_to_display}s, ran ${elapsed}s, gaps=$gaps)"
