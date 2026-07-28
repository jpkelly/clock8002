#!/bin/sh
# piclock-ltc-soak.sh — Trixie / Raspberry Pi OS soak logger for LTC + 5V rail.
#
# Runs ON the Pi 5 target (Trixie, systemd, vcgencmd available). This is NOT a
# Buildroot/BusyBox script — it deliberately uses systemctl, journalctl and
# vcgencmd, none of which exist on the Buildroot image.
#
# Correlates LTC decoder health against power-rail health so a USB audio
# brownout (CM108 dongle MCU hang -> "usb_set_interface failed (-110)") can be
# tied to a 5V sag rather than guessed at.
#
# Usage (on target):
#   nohup setsid sh /opt/clock8002/piclock-ltc-soak.sh 30 >/dev/null 2>&1 &
#
# Args: $1 = sample interval seconds (default 30)
# Log:  /var/log/piclock-ltc-soak.log   (append; header written on each start)

INTERVAL="${1:-30}"
LOG=/var/log/piclock-ltc-soak.log

ext5v()    { vcgencmd pmic_read_adc EXT5V_V 2>/dev/null | sed 's/.*=\([0-9.]*\)V/\1/'; }
throttled(){ vcgencmd get_throttled 2>/dev/null | cut -d= -f2; }
temp()     { vcgencmd measure_temp 2>/dev/null | sed 's/temp=//;s/'"'"'C//'; }
uvcount()  { dmesg 2>/dev/null | grep -ci 'Undervoltage detected'; }
setifcnt() { dmesg 2>/dev/null | grep -c 'usb_set_interface failed'; }
# The REAL failure is xHCI host-controller death. usb_set_interface is only a
# downstream consequence of it, so counting that alone (as this script used to)
# cannot distinguish a dead controller from a sulking dongle.
hcdeadcnt(){ dmesg 2>/dev/null | grep -cE 'HC died|assume dead|Host halt failed'; }
urbcnt()   { dmesg 2>/dev/null | grep -c 'active urbs on EP'; }
cardgone() { grep -q 'USB-Audio' /proc/asound/cards 2>/dev/null && echo 0 || echo 1; }
restarts() { systemctl show alsa-ltc.service -p NRestarts --value 2>/dev/null; }
active()   { systemctl is-active alsa-ltc.service 2>/dev/null; }

# Latest cumulative "[heartbeat] N frames, M ltc decoded, E errors"
hb_field() {
    journalctl -b -u alsa-ltc.service --no-pager 2>/dev/null \
        | grep '\[heartbeat\]' | tail -1 \
        | sed -n "s/.*\[heartbeat\] \([0-9]*\) frames, \([0-9]*\) ltc decoded, \([0-9]*\) errors.*/\\$1/p"
}

say() { echo "$*" | tee -a "$LOG"; }

say "==================================================================="
say "SOAK START $(date '+%F %T %Z')  interval=${INTERVAL}s  boot=$(uptime -s)"
say "PSU: max_current=$(od -An -tu4 --endian=big /proc/device-tree/chosen/power/max_current 2>/dev/null | tr -d ' ')mA usb_max_current_enable=$(od -An -tu4 --endian=big /proc/device-tree/chosen/power/usb_max_current_enable 2>/dev/null | tr -d ' ')"
say "ts                  up_s  ext5v   thr      temp  ltc      rst decoded  err setif uv"
say "-------------------------------------------------------------------"

BASE_RST=$(restarts); [ -n "$BASE_RST" ] || BASE_RST=0
LAST_DEC=-1
STALL=0

while :; do
    TS=$(date '+%F %T')
    UP=$(cut -d. -f1 /proc/uptime)
    E5=$(ext5v); TH=$(throttled); TC=$(temp)
    AC=$(active); RS=$(restarts)
    DEC=$(hb_field 2)
    ERR=$(hb_field 3)
    SI=$(setifcnt); UV=$(uvcount)
    [ -n "$DEC" ] || DEC=0
    [ -n "$ERR" ] || ERR=0
    [ -n "$RS" ]  || RS=0

    printf '%s %6s %-7s %-8s %-5s %-8s %-3s %-7s %-4s %-5s %s\n' \
        "$TS" "$UP" "$E5" "$TH" "$TC" "$AC" "$RS" "$DEC" "$ERR" "$SI" "$UV" \
        | tee -a "$LOG"

    # ---- regression alerts ----
    [ "$TH" != "0x0" ] && say "  !! ALERT throttle/undervoltage flags set: $TH"
    [ "$UV" -gt 0 ] 2>/dev/null && say "  !! ALERT $UV undervoltage event(s) in dmesg"
    HC=$(hcdeadcnt); UB=$(urbcnt); CG=$(cardgone)
    [ "$HC" -gt 0 ] 2>/dev/null && say "  !! FATAL $HC xHCI HOST CONTROLLER DEATH event(s) — VL805 dead, reboot required"
    [ "$UB" -gt 0 ] 2>/dev/null && say "  !! ALERT $UB stranded-isochronous-URB event(s) — primary fault signature"
    [ "$CG" -eq 1 ] 2>/dev/null && say "  !! FATAL USB audio card absent from /proc/asound/cards"
    [ "$SI" -gt 0 ] 2>/dev/null && say "  !! ALERT $SI usb_set_interface failure(s) — downstream consequence, not the cause"
    [ "$AC" != "active" ] && say "  !! ALERT alsa-ltc not active (state=$AC)"
    [ "$RS" -gt "$BASE_RST" ] 2>/dev/null && say "  !! ALERT alsa-ltc restarted ($BASE_RST -> $RS)"

    # LTC decode progress stall detection (heartbeats are ~30s apart)
    if [ "$DEC" -eq "$LAST_DEC" ] 2>/dev/null; then
        STALL=$((STALL + 1))
        [ "$STALL" -ge 3 ] && say "  !! ALERT LTC decode count stalled at $DEC for $STALL samples"
    else
        STALL=0
    fi
    LAST_DEC=$DEC

    sleep "$INTERVAL"
done
