#!/bin/sh
# Cycle the playback endpoint's sample rate while capture runs at a fixed 44100.
# Each change forces the device through set_interface + set_freq control
# transfers - the same operations that return -110 when the controller wedges.
set -u
DEV=${DEV:-hw:CARD=Device,DEV=0}
RATES=${RATES:-"48000 44100 32000 96000 22050 16000 8000"}
HOLD=${HOLD:-3}
LOG=/var/log/ltc-forensics/ramp-log.txt

n=0
while :; do
    for r in $RATES; do
        n=$((n + 1))
        timeout $((HOLD + 2)) aplay -D "$DEV" -f S16_LE -r "$r" -c 2 -t raw \
            -d "$HOLD" /dev/zero >/dev/null 2>&1
        rc=$?
        [ $rc -ne 0 ] && echo "$(date '+%H:%M:%S') rate=$r aplay_rc=$rc" >> "$LOG"
        if [ $((n % 20)) -eq 0 ]; then
            echo "$(date '+%H:%M:%S') rate-cycle: $n changes" >> "$LOG"
        fi
    done
done
