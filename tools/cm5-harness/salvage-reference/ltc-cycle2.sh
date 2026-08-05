#!/bin/sh
# Multi-boot LTC stress/stability harness.
# Each cycle: boot -> wait for LTC stream AND display up -> measure for
# MONITOR_S. Load is ALTERNATED per cycle (odd=load, even=no load) so that
# time-based drift cannot masquerade as a load effect.
# Config: /etc/default/ltc-cycle

BASE=/opt/clock8002
LOGDIR=$BASE/logs
COUNT=$BASE/ltc-cycle.count
STOP=$BASE/ltc-cycle.stop

MAX_CYCLES=20
MONITOR_S=180
STREAM_TIMEOUT=90
DISPLAY_TIMEOUT=90
ALTERNATE_LOAD=1
LABEL="unlabelled"
RESULTS=$LOGDIR/ltc-cycle-results.csv
[ -f /etc/default/ltc-cycle ] && . /etc/default/ltc-cycle

mkdir -p "$LOGDIR"
[ -f "$RESULTS" ] || echo "cycle,label,timestamp,kernel,pagesize,ltc_sha,load,boot_to_stream_s,boot_to_display_s,monitored_s,outcome,gaps_1s,min_avail_mb,max_swap_mb,oom_killed,frames,ltc_decoded,errors" > "$RESULTS"

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

load=no
if [ "$ALTERNATE_LOAD" = "1" ] && [ $((n % 2)) -eq 1 ]; then load=yes; fi

card=$(awk '/USB Audio/{print $1; exit}' /proc/asound/cards 2>/dev/null)
[ -z "$card" ] && card=0
STATUS=/proc/asound/card$card/pcm0c/sub0/status

read_hw()    { awk '/hw_ptr/{print $3}' "$STATUS" 2>/dev/null; }
is_running() { grep -q 'state: RUNNING' "$STATUS" 2>/dev/null; }
app_alive()  { ps -eo args | grep -q '[a]lsa-ltc'; }
up_s()       { cut -d. -f1 /proc/uptime; }
avail_mb()   { awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo; }
swap_mb()    { awk '/SwapTotal/{t=$2} /SwapFree/{f=$2} END{print int((t-f)/1024)}' /proc/meminfo; }
oom_hits()   { dmesg 2>/dev/null | grep -c 'Out of memory: Killed'; }

row() {
    echo "$n,\"$LABEL\",$(date '+%Y-%m-%d %H:%M:%S'),$kern,$psize,$sha,$load,$1,$2,$3,$4,$5,$6,$7,$8,$9,${10},${11}" >> "$RESULTS"
    sync
}

finish() {
    /opt/clock8002/ltc-stress.sh stop 2>/dev/null
    echo "cycle $n [load=$load]: $1"
    [ "${1#clean}" = "$1" ] && sleep 10
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
    row "" "" 0 never_started 0 "" "" no "" "" ""
    finish never_started
fi
boot_to_stream=$(up_s)

# --- settle 2: display service up (real production state) ---
w=0
while [ "$w" -lt "$DISPLAY_TIMEOUT" ]; do
    [ "$(systemctl is-active clock8002 2>/dev/null)" = "active" ] && break
    sleep 1; w=$((w + 1))
done
boot_to_display=$(up_s)
sleep 5

oom_before=$(oom_hits)
[ "$load" = "yes" ] && /opt/clock8002/ltc-stress.sh start

outcome=clean
gaps=0
min_avail=$(avail_mb)
max_swap=$(swap_mb)
prev=$(read_hw)
same=0
elapsed=0

while [ "$elapsed" -lt "$MONITOR_S" ]; do
    sleep 1
    elapsed=$((elapsed + 1))

    a=$(avail_mb); [ "$a" -lt "$min_avail" ] && min_avail=$a
    s=$(swap_mb);  [ "$s" -gt "$max_swap" ] && max_swap=$s

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

oom_after=$(oom_hits)
oomk=$((oom_after - oom_before))
# an OOM kill invalidates an app_exit verdict - label it honestly
[ "$oomk" -gt 0 ] && [ "$outcome" = "wedge_app_exit" ] && outcome=INVALID_oom_killed

hb=$(journalctl -u alsa-ltc -b --no-pager 2>/dev/null | grep '\[heartbeat\]' | tail -1)
frames=$(echo "$hb" | sed -n 's/.*\[heartbeat\] \([0-9]*\) frames.*/\1/p')
ltc=$(echo "$hb"    | sed -n 's/.*frames, \([0-9]*\) ltc.*/\1/p')
errs=$(echo "$hb"   | sed -n 's/.*decoded, \([0-9]*\) errors.*/\1/p')

row "$boot_to_stream" "$boot_to_display" "$elapsed" "$outcome" "$gaps" "$min_avail" "$max_swap" "$oomk" "$frames" "$ltc" "$errs"
finish "$outcome (stream ${boot_to_stream}s, display ${boot_to_display}s, ran ${elapsed}s, gaps=$gaps, minAvail=${min_avail}MB, maxSwap=${max_swap}MB, oom=$oomk)"
