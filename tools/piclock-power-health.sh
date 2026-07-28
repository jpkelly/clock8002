#!/bin/sh
# piclock-power-health.sh — Raspberry Pi 5 power supply / USB power diagnostics.
#
# Purpose: determine whether the attached PSU is adequate, and whether 5V rail
# sag is causing USB peripheral brownouts (e.g. the CM108 LTC audio dongle
# hanging with "usb_set_interface failed (-110)").
#
# Usage:  sh piclock-power-health.sh
# Safe to run at any time; read-only apart from printing.

PWR=/proc/device-tree/chosen/power

dt_u32() {
    # Read a big-endian u32 from a device-tree property file.
    [ -f "$1" ] || { echo "n/a"; return; }
    od -An -tu4 --endian=big "$1" 2>/dev/null | tr -d ' \n'
}

ext5v() {
    vcgencmd pmic_read_adc EXT5V_V 2>/dev/null | sed 's/.*=\([0-9.]*\)V/\1/'
}

echo "======================================================================"
echo " piClock power health   $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo "======================================================================"
echo "model     : $(tr -d '\0' < /proc/device-tree/model 2>/dev/null)"
echo "kernel    : $(uname -r)"
echo "booted    : $(uptime -s)   (up $(cut -d. -f1 /proc/uptime)s)"
echo

echo "--- PSU negotiation (set once at boot by firmware) -------------------"
MAXC=$(dt_u32 "$PWR/max_current")
USBMAX=$(dt_u32 "$PWR/usb_max_current_enable")
echo "max_current               = ${MAXC} mA"
echo "usb_max_current_enable    = ${USBMAX}"
echo "usb_over_current_detected = $(dt_u32 "$PWR/usb_over_current_detected")"
echo "sd_overcurrent            = $(dt_u32 "$PWR/sd_overcurrent")"

PDO_NONZERO=$(od -An -tx4 --endian=big "$PWR/usbpd_power_data_objects" 2>/dev/null \
              | tr -d ' \n' | tr -s '0' '0' | sed 's/0*//')
if [ -z "$PDO_NONZERO" ]; then
    echo "usbpd_power_data_objects  = all zeros  -> NOT a USB-PD supply"
else
    echo "usbpd_power_data_objects  = present   -> USB-PD supply detected"
fi

echo
case "$MAXC" in
    5000) echo "VERDICT: OK    5A/27W-class PD supply detected." ;;
    3000) echo "VERDICT: MARGINAL  3A supply; USB budget reduced." ;;
    900)  echo "VERDICT: BAD   900mA fallback — no PD negotiation." ;;
    *)    echo "VERDICT: UNKNOWN max_current=${MAXC}" ;;
esac
if [ "$USBMAX" = "0" ]; then
    echo "         USB current restriction is ACTIVE (600mA total, all ports)."
    echo "         USB peripherals may brown out under load."
else
    echo "         USB ports have full current budget."
fi
echo

echo "--- Throttle / under-voltage flags -----------------------------------"
TH=$(vcgencmd get_throttled 2>/dev/null | cut -d= -f2)
echo "get_throttled = $TH"
# Arithmetic expansion understands the 0x prefix in POSIX shells; printf '%d'
# does not do so portably (dash rejects it).
THN=$(( ${TH:-0} ))
chk() { [ $(( THN & $1 )) -ne 0 ] && echo "    * $2"; }
if [ "$THN" -eq 0 ]; then
    echo "    clean — no under-voltage or throttling since boot"
else
    chk 1      "NOW: under-voltage detected"
    chk 2      "NOW: ARM frequency capped"
    chk 4      "NOW: currently throttled"
    chk 8      "NOW: soft temperature limit active"
    chk 65536  "PAST: under-voltage HAS occurred"
    chk 131072 "PAST: ARM frequency capping HAS occurred"
    chk 262144 "PAST: throttling HAS occurred"
    chk 524288 "PAST: soft temperature limit HAS occurred"
fi
echo
echo "under-voltage events in dmesg: $(dmesg | grep -ci 'Undervoltage detected')"
dmesg -T 2>/dev/null | grep -iE "undervoltage|voltage normalis" | tail -8
echo

echo "--- Rails ------------------------------------------------------------"
echo "EXT5V   = $(ext5v) V   (nominal 5.1V; <4.9V under load is a red flag)"
echo "temp    = $(vcgencmd measure_temp 2>/dev/null | cut -d= -f2)"
echo

echo "--- USB topology / declared draw -------------------------------------"
for d in /sys/bus/usb/devices/*/; do
    [ -f "$d/bMaxPower" ] || continue
    case "$(basename "$d")" in usb*) continue ;; esac
    printf "  %-10s %-8s %s:%s  %s\n" \
        "$(basename "$d")" "$(cat "$d/bMaxPower" 2>/dev/null)" \
        "$(cat "$d/idVendor" 2>/dev/null)" "$(cat "$d/idProduct" 2>/dev/null)" \
        "$(cat "$d/product" 2>/dev/null)"
done
echo

echo "--- LTC decoder health -----------------------------------------------"
echo "alsa-ltc  : $(systemctl is-active alsa-ltc.service 2>/dev/null) / restarts=$(systemctl show alsa-ltc.service -p NRestarts --value 2>/dev/null)"
echo "usb_set_interface failures in dmesg: $(dmesg | grep -c 'usb_set_interface failed')"
echo "last LTC heartbeat:"
journalctl -b -u alsa-ltc.service --no-pager 2>/dev/null | grep '\[heartbeat\]' | tail -3
echo "======================================================================"
