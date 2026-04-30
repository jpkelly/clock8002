#!/bin/bash
# Soak test monitor for piclockBR — logs metrics every hour to soak-<date>.log
# Usage: ./tools/soak-monitor.sh [host] [interval_seconds]

HOST="${1:-192.168.8.245}"
INTERVAL="${2:-3600}"
LOGFILE="/tmp/soak-$(date +%Y%m%d-%H%M%S).log"
KEY="$HOME/.ssh/id_rsa"

log() {
    echo "$*" | tee -a "$LOGFILE"
}

check() {
    ssh -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
        -i "$KEY" root@"$HOST" '
UPTIME=$(uptime)
PID=$(ps aux | grep "sdl3-clock" | grep -v grep | awk "{print \$1}" | head -1)
if [ -n "$PID" ]; then
    VMRSS=$(grep VmRSS /proc/$PID/status | awk "{print \$2}")
    VMSWAP=$(grep VmSwap /proc/$PID/status | awk "{print \$2}")
    PROC_OK="YES (PID '$PID')"
else
    VMRSS="N/A"
    VMSWAP="N/A"
    PROC_OK="NOT RUNNING"
fi
FREE=$(free | awk "/Mem:/ {print \$4}")
SWAP_USED=$(free | awk "/Swap:/ {print \$3}")
echo "uptime:      $UPTIME"
echo "sdl3-clock:  $PROC_OK"
echo "VmRSS:       ${VMRSS} kB"
echo "VmSwap:      ${VMSWAP} kB"
echo "RAM free:    ${FREE} kB"
echo "swap used:   ${SWAP_USED} kB"
'
}

log "=== Soak monitor started: commit af37c54 ==="
log "Host: $HOST | Interval: ${INTERVAL}s | Log: $LOGFILE"
log ""

SAMPLE=0
START=$(date +%s)

while true; do
    NOW=$(date +%s)
    ELAPSED=$(( (NOW - START) / 3600 ))
    SAMPLE=$((SAMPLE + 1))
    log "--- Sample $SAMPLE | ~${ELAPSED}h elapsed | $(date -u '+%Y-%m-%d %H:%M:%S UTC') ---"
    RESULT=$(check 2>&1)
    if [ $? -ne 0 ]; then
        log "ERROR: SSH failed — device unreachable or crashed"
        log "$RESULT"
    else
        log "$RESULT"
    fi
    log ""
    sleep "$INTERVAL"
done
