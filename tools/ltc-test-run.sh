#!/bin/sh
# ltc-test-run.sh — launch alsa-ltc for a controlled soak test, early in boot.
#
# TRIXIE TARGET ONLY (systemd). Not for the Buildroot image.
#
# WHY THIS EXISTS
# The CM108 USB audio dongle becomes unresponsive to SET_INTERFACE ~20s after
# enumeration if no USB traffic has started (see the comment in v4/alsa-ltc.c).
# Launching by hand over SSH cannot reliably hit that window — observed launches
# landed at 19.6s (worked) and 26.8s / 60s+ (failed with usb_set_interface -110).
# Those failures are a KNOWN startup-window issue, NOT the silent-stall fault
# under investigation, and they were polluting every manual test.
#
# This script is started by ltc-test.service a few seconds into boot so that
# every run begins at the same time-since-enumeration, inside the window.
#
# It also keeps logs OFF /tmp: /tmp is tmpfs on these units (494 MB of 986 MB
# total RAM on the 1 GB chassis), and a runaway diagnostic writer previously
# exhausted RAM and took the whole system down, losing networking entirely.
#
# Config (optional): /etc/default/ltc-test
#   BIN=/opt/clock8002/alsa-ltc-usb-parity8
#   READ_FRAMES=1024
#   CHANNELS=1
#   DEVICE=plughw:CARD=Device,DEV=0
#   OSC_IP=255.255.255.255
#   OSC_PORT=1245
#   EXTRA=-d
#   DELAY_S=0        # seconds to wait before starting capture
#   RAW_ARGS=0       # 1 = pass ONLY <device> <ip> <port>, no flags
#
# RAW_ARGS exists so this harness can also launch the UPSTREAM clock-8001
# alsa-ltc binary, whose interface is exactly:
#     alsa-ltc <alsa-device> <OSC destination ip> <OSC port>
# It accepts none of our flags (-d, --read-frames, --channels, --buffer-frames)
# and would abort on them. Using the same harness for both keeps launch timing,
# logging and metadata identical across the comparison.
#
# DELAY_S exists to test an observed pattern: two cold boots launched at ~4.7s
# uptime both stalled after 100352 and 101376 frames (~2.3s of audio) -- within
# 1% of each other -- while launches at ~19.6s survived ~14s and >28s. Either
# the dongle needs settling time after enumeration, or launching at 4.7s
# collides with the rest of boot (multi-user.target completes ~11.5s).
# Keep DELAY_S small enough that capture still starts INSIDE the ~20s window in
# which the dongle answers SET_INTERFACE; a longer delay just reproduces the
# known startup-window failure instead of testing anything.

CONF=/etc/default/ltc-test
[ -f "$CONF" ] && . "$CONF"

# Defaults applied AFTER sourcing, via ${VAR:-default}, so both the config file
# and an inherited environment variable can override them. Do NOT assign
# defaults before sourcing — that silently clobbers env overrides.
BIN=${BIN:-/opt/clock8002/alsa-ltc-usb-parity8}
READ_FRAMES=${READ_FRAMES:-1024}
CHANNELS=${CHANNELS:-1}
DEVICE=${DEVICE:-plughw:CARD=Device,DEV=0}
OSC_IP=${OSC_IP:-255.255.255.255}
OSC_PORT=${OSC_PORT:-1245}
# NOTE on EXTRA: use ${EXTRA-default}, NOT ${EXTRA:-default}. The colon form
# treats an EMPTY value as "unset" and substitutes the default, which made it
# impossible to run without -d: setting `EXTRA=` in the config silently still
# passed -d. The no-colon form only substitutes when EXTRA is genuinely unset,
# so `EXTRA=` now correctly means "no extra flags" (production mode).
EXTRA=${EXTRA--d}
DELAY_S=${DELAY_S:-0}
RAW_ARGS=${RAW_ARGS:-0}

LOGDIR=${LOGDIR:-/opt/clock8002/logs}
mkdir -p "$LOGDIR" 2>/dev/null
TS=$(date +%Y%m%d-%H%M%S)
LOG="$LOGDIR/ltc-$TS.log"

# Sleep BEFORE sampling uptime so uptime_at_launch reflects the real moment
# capture begins, not when systemd started this script.
if [ "$DELAY_S" != "0" ]; then
    sleep "$DELAY_S"
fi

UPTIME_AT_LAUNCH=$(cut -d' ' -f1 /proc/uptime)
# Best-effort enumeration timestamp. `dmesg` is usually restricted to root
# (kernel.dmesg_restrict=1) and this runs as pi, so fall back to journalctl's
# monotonic kernel log before giving up.
ENUM_AT=$(dmesg 2>/dev/null | awk '/1-1\.1: New USB device found/{gsub(/[][]/,"",$1); print $1; exit}')
if [ -z "$ENUM_AT" ]; then
    ENUM_AT=$(journalctl -k -o short-monotonic --no-pager 2>/dev/null \
              | awk '/1-1\.1: New USB device found/{gsub(/[][]/,"",$1); print $1; exit}')
fi

{
    echo "=== ltc-test harness run $TS ==="
    echo "binary          : $BIN"
    echo "sha256          : $(sha256sum "$BIN" 2>/dev/null | cut -d' ' -f1)"
    if [ "$RAW_ARGS" = "1" ]; then
        echo "args            : $DEVICE $OSC_IP $OSC_PORT   (RAW_ARGS: upstream interface, no flags)"
    else
        echo "args            : $EXTRA --read-frames $READ_FRAMES --channels $CHANNELS $DEVICE $OSC_IP $OSC_PORT"
    fi
    echo "delay_s         : ${DELAY_S}s (configured startup delay)"
    echo "uptime_at_launch: ${UPTIME_AT_LAUNCH}s"
    if [ -n "$ENUM_AT" ]; then
        echo "enumerated_at   : ${ENUM_AT}s"
    else
        echo "enumerated_at   : unknown (dmesg restricted, journalctl unavailable)"
    fi
    echo "window_margin   : uptime_at_launch minus enumerated_at must be well under ~20s"
    echo "kernel          : $(uname -r)"
    echo "chassis_serial  : $(awk '/Serial/{print $3}' /proc/cpuinfo)"
    echo "=========================================="
} >> "$LOG" 2>&1

# exec so systemd tracks the real process, and so signals reach it directly.
if [ "$RAW_ARGS" = "1" ]; then
    exec "$BIN" "$DEVICE" "$OSC_IP" "$OSC_PORT" >> "$LOG" 2>&1
fi
exec "$BIN" $EXTRA --read-frames "$READ_FRAMES" --channels "$CHANNELS" \
     "$DEVICE" "$OSC_IP" "$OSC_PORT" >> "$LOG" 2>&1
