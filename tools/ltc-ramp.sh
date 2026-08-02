#!/bin/sh
# Alternating poll-rate arms, to test whether the recorder's own MMIO/config
# read pressure provokes the VL805 hang. Arms alternate rather than ramping
# monotonically so that session-long drift cannot masquerade as a rate response.
set -u

DEFAULTS=/etc/default/ltc-forensics
OUTDIR=/var/log/ltc-forensics
LOG=$OUTDIR/ramp-log.txt
ARM_S=${ARM_S:-600}
SEQ=${SEQ:-"10 100 10 100 10 100"}

log() { echo "$(date '+%H:%M:%S') $*" >> "$LOG"; }

# Wedge-class events only; app_gap and friends are not the failure under test.
count_wedges_since() {
    since=$1
    n=0
    for d in "$OUTDIR"/event-*; do
        [ -d "$d" ] || continue
        case "$d" in *frozen*|*mmio_all_ones*|*usbsts*|*pcie_*) ;; *) continue ;; esac
        ts=$(basename "$d" | sed 's/^event-[0-9]*-//; s/-.*//')
        [ "$ts" -ge "$since" ] 2>/dev/null && n=$((n + 1))
    done
    echo "$n"
}

log "=== ramp start, arm=${ARM_S}s, seq=[$SEQ] ==="
arm=0
for hz in $SEQ; do
    arm=$((arm + 1))
    sed -i "s/^HZ=.*/HZ=$hz/" "$DEFAULTS"
    systemctl restart ltc-forensics
    start=$(date '+%H%M%S')
    log "arm $arm: hz=$hz START"
    sleep "$ARM_S"
    n=$(count_wedges_since "$start")
    log "arm $arm: hz=$hz END  wedge_events=$n  ltc_restarts=$(systemctl show alsa-ltc -p NRestarts --value)"
done
log "=== ramp complete ==="
