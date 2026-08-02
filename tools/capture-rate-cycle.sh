#!/bin/sh
# Cycle the CAPTURE endpoint's sample rate. Capture is endpoint 0x82 - the one
# that reports "cannot set freq 44100 to ep 0x82" when the controller wedges -
# so each change exercises the exact control path seen failing in the wedge logs.
# alsa-ltc must be stopped first; it holds the capture device exclusively.
set -u
DEV=${DEV:-hw:CARD=Device,DEV=0}
RATES=${RATES:-"48000 44100"}
HOLD=${HOLD:-2}
LOG=/var/log/ltc-forensics/ramp-log.txt

n=0
while :; do
    for r in $RATES; do
        n=$((n + 1))
        timeout $((HOLD + 3)) arecord -D "$DEV" -f S16_LE -r "$r" -c 1 -t raw \
            -d "$HOLD" /dev/null >/dev/null 2>&1
        rc=$?
        [ $rc -ne 0 ] && echo "$(date '+%H:%M:%S') capture rate=$r arecord_rc=$rc" >> "$LOG"
        if [ $((n % 20)) -eq 0 ]; then
            echo "$(date '+%H:%M:%S') capture-rate-cycle: $n changes" >> "$LOG"
        fi
    done
done
