#!/bin/sh
# alsa-ltc-usb-wedge-watchdog.sh — detect, log and (optionally) recover from the
# USB audio wedge that makes alsa-ltc crash-loop forever on the Trixie /
# Raspberry Pi OS Pi 5 target (piclock.local).
#
# THIS IS THE TRIXIE TARGET ONLY. It uses systemctl / journalctl / vcgencmd and
# must NOT be assumed to work on the Buildroot image (BusyBox, no systemd).
#
# ---------------------------------------------------------------------------
# CONFIRMED failure chain (Pi 5 Model B Rev 1.1, kernel 6.18.34+rpt-rpi-2712,
# C-Media 0d8c:0014 CM108-class dongle at 1-1.1, behind VIA 2109:3431 hub, on a
# VIA VL805/806 xHCI [1106:3483] PCIe card at 0001:01:00.0).
# Captured end-to-end 2026-07-28:
#
#   1. alsa-ltc streams cleanly for 110 s to 2725 s (0 decode errors).
#   2. PRIMARY FAULT: the VL805 silently stops completing split-isochronous IN
#      URBs for the full-speed dongle. 12 URBs (MAX_URBS) strand in flight. No
#      xHCI event, no error interrupt — ALSA still reports pcm_state=RUNNING.
#      First visible symptom is a decode gap, ~5 s BEFORE the first -EIO:
#        [gap] no LTC decoded for 2464ms, peak_during_gap=32767
#   3. snd_pcm_readi then returns -EIO every ~3.33 s, pcm_state still RUNNING.
#   4. alsa-ltc hits its 10-error limit and tears the stream down.
#   5. SECONDARY CATASTROPHE — this is what makes it permanent. The teardown
#      must cancel the 12 stranded URBs:
#        usb 1-1.1: timeout: still 12 active urbs on EP #82
#        xhci_hcd: xHCI host not responding to stop endpoint command
#        xhci_hcd: Host halt failed, -110
#        xhci_hcd: xHCI host controller not responding, assume dead
#        xhci_hcd: HC died; cleaning up
#      The whole host controller dies, taking the hub and dongle with it.
#      /proc/asound/cards loses the USB card entirely.
#   6. NO SOFTWARE RECOVERY EXISTS. Verified 2026-07-28:
#        - PCIe remove + rescan  -> "xHCI HW did not halt ... status = 0x1000"
#          (USBSTS bit 12 = HCE, Host Controller Error) -> probe fails -110
#        - sysfs device reset    -> HCE clears (status = 0x0) but the chip still
#          never asserts HCHalted -> "Host halt failed, -110"
#        - PCIe secondary bus reset on bridge 0001:00:00.0 -> no effect;
#          BRIDGE_CONTROL reads 0000 before and after, so SBR is not reaching
#          the card as a PERST#.
#      Only a full reboot (firmware re-runs PCIe init) restores it.
#
# RULED OUT with direct evidence on this unit — do not re-litigate:
#   * Power supply. Proper 5A USB-PD PSU: max_current=5000, throttled=0x0,
#     EXT5V 5.13-5.16V, ZERO undervoltage events — wedge STILL happened.
#   * PCIe signal integrity / ribbon cable. AER correctable, nonfatal AND fatal
#     counters are ALL ZERO. Link stable at 5 GT/s x1 throughout.
#   * ASPM. LnkCap says "ASPM not supported", LnkCtl says "ASPM Disabled", and
#     the kernel applies its VL805 ASPM fixup. Already off; not a lever.
#   * USB autosuspend. 1-1.1 has power/control=on, runtime_suspended_time=0.
#   * Client contention. No pipewire/pulseaudio/wireplumber/jackd.
#   * Startup/enumeration race. Failure occurs minutes into a HEALTHY stream.
#   * Sample rate as a CURE. 48000 (the dongle's native rate) extends survival
#     ~10x but the same HC death still occurs. Mitigation, not a fix.
#
# NOTE: earlier revisions of this file claimed (a) a "single-TT hub scheduling
# collision", (b) a "PSU brownout", and (c) that "the kernel logs NOTHING at
# the moment of failure". (b) is positively DISPROVEN and (c) is simply WRONG —
# the kernel logs a full host-controller death; earlier sessions only grepped
# for usb_set_interface and missed it. usb_set_interface failed (-110/-19) is a
# DOWNSTREAM CONSEQUENCE, never the fault.
# ---------------------------------------------------------------------------
#
# Config: /etc/default/alsa-ltc-watchdog
#   AUTO_RECOVER=0|1          (default 0)
#   MAX_RECOVER_CYCLES=<n>    (default 10)
#
# Deployed as a systemd oneshot + timer — see alsa-ltc-watchdog.service and
# alsa-ltc-watchdog.timer in this same directory.
#
# Safe to run standalone for a one-off check: sh alsa-ltc-usb-wedge-watchdog.sh

LOG=/var/log/alsa-ltc-watchdog.log
INTERVALS=/var/log/piclock-ltc-intervals.log
ARCHIVE=/var/log/piclock-forensics
LIVELOG=/var/log/piclock-ltc-live-stall.log  # captured while the HC is STILL ALIVE
LIVESTATE=/run/alsa-ltc-watchdog.live-stall  # per-boot COUNT of live-stall captures
LIVEMAX=3                                    # max live-stall captures per boot

# Read-only xHCI MMIO register dump (USBCMD/USBSTS/IMAN/CRCR/PORTSC). Prefer a
# copy sitting next to this script so the tool works when run straight from a
# git checkout; fall back to the deployed location.
REGDUMP=$(dirname "$0")/xhci-regs.py
[ -f "$REGDUMP" ] || REGDUMP=/opt/clock8002/xhci-regs.py
STATE=/run/alsa-ltc-watchdog.wedged          # per-boot (tmpfs): act once per boot
COUNTER=/var/lib/alsa-ltc-watchdog/cycles    # persists across reboots
THRESHOLD=3        # usb_set_interface failures in the recent dmesg tail to call it wedged
DMESG_LINES=200    # how many recent dmesg lines to scan each run
CRASHLOOP_RESTARTS=8   # NRestarts while not active => crash-looping
MIN_UPTIME=120     # don't trust "card missing" / "crashloop" before this many seconds

# Capture any explicit overrides from the ENVIRONMENT *before* we assign
# defaults or source the config file, then re-apply them afterwards so the
# environment always wins.
#
# This matters: /etc/default/alsa-ltc-watchdog sets AUTO_RECOVER=1, and sourcing
# it used to clobber an env-supplied AUTO_RECOVER=0. On 2026-07-28 that turned
# an intended dry run into a real reboot of the target. Use DRY_RUN=1 for a
# genuinely side-effect-free check.
_ENV_AUTO_RECOVER=$AUTO_RECOVER
_ENV_DRY_RUN=$DRY_RUN

AUTO_RECOVER=0
MAX_RECOVER_CYCLES=10
DRY_RUN=0

[ -f /etc/default/alsa-ltc-watchdog ] && . /etc/default/alsa-ltc-watchdog

[ -n "$_ENV_AUTO_RECOVER" ] && AUTO_RECOVER=$_ENV_AUTO_RECOVER
[ -n "$_ENV_DRY_RUN" ] && DRY_RUN=$_ENV_DRY_RUN

ts() { date '+%Y-%m-%d %H:%M:%S'; }

dtail=$(dmesg -T 2>/dev/null | tail -n "$DMESG_LINES")
fails=$(printf '%s\n' "$dtail" | grep -c "usb_set_interface failed")
restarts=$(systemctl show alsa-ltc.service -p NRestarts --value 2>/dev/null)
substate=$(systemctl show alsa-ltc.service -p SubState --value 2>/dev/null)
active=$(systemctl is-active alsa-ltc.service 2>/dev/null)
uptime_s=$(cut -d. -f1 /proc/uptime 2>/dev/null)

# ---- multi-signature detection ---------------------------------------------
# Ordered most-conclusive first. REASON is recorded so the intervals log tells
# us WHICH failure mode each episode was, not just that something broke.
REASON=""

# 1. xHCI host controller death. Conclusive, and unrecoverable without reboot.
printf '%s\n' "$dtail" | grep -qE 'HC died|assume dead|Host halt failed|setup: -110|probe with driver xhci_hcd failed' \
    && REASON=hc_dead

# 2. Stranded isochronous URBs — the PRIMARY fault, and the precursor that the
#    teardown path escalates into HC death. Catching this is the early warning.
if [ -z "$REASON" ]; then
    printf '%s\n' "$dtail" | grep -qE 'still [0-9]+ active urbs on EP' && REASON=urb_stall
fi

# 3. USB audio card vanished entirely (the functional symptom of HC death).
#    Guarded by MIN_UPTIME so we never fire during normal boot enumeration.
if [ -z "$REASON" ] && [ "${uptime_s:-0}" -ge "$MIN_UPTIME" ] 2>/dev/null; then
    grep -q 'USB-Audio' /proc/asound/cards 2>/dev/null || REASON=card_gone
fi

# 4. Legacy signature: storm of usb_set_interface failures.
if [ -z "$REASON" ] && [ "${fails:-0}" -ge "$THRESHOLD" ] 2>/dev/null; then
    REASON=setif_storm
fi

# 5. Crash-looping without any of the above (e.g. latched into 'failed').
if [ -z "$REASON" ] && [ "${uptime_s:-0}" -ge "$MIN_UPTIME" ] 2>/dev/null \
   && [ "$active" != "active" ] && [ "${restarts:-0}" -ge "$CRASHLOOP_RESTARTS" ] 2>/dev/null; then
    REASON=crashloop
fi

# ---- EARLY WARNING: capture forensics while the controller is STILL ALIVE ---
# Everything else in this script is a post-mortem, taken after the VL805 is dead
# — at which point every register read is meaningless and the USB topology is
# already gone. The ONLY window in which the stalled isochronous endpoint can be
# inspected is between the first decode gap / -EIO and alsa-ltc's teardown,
# roughly 33-38 s. This branch grabs that window.
#
# It is deliberately CAPTURE-ONLY: it never reboots, and it uses its own
# once-per-boot flag so it does not consume $STATE and does not interfere with
# the real hc_dead handling that follows a few seconds later.
#
# The key measurement is the xHCI interrupt counter sampled twice, 2 s apart.
# If URBs are queued but the IRQ count is FROZEN, the controller has stopped
# generating completion events altogether — that distinguishes "controller went
# silent" from "device stopped sending data".
if [ "$DRY_RUN" != "1" ] && [ "$REASON" != "hc_dead" ] \
   && grep -q 'USB-Audio' /proc/asound/cards 2>/dev/null; then
    # $LIVESTATE now holds a COUNT, not just a flag, so a benign early event can no
    # longer consume the single per-boot capture. Bounded by $LIVEMAX so the log
    # cannot grow without limit.
    _livecount=$(cat "$LIVESTATE" 2>/dev/null || echo 0)
    case "$_livecount" in ''|*[!0-9]*) _livecount=0 ;; esac
  if [ "$_livecount" -lt "$LIVEMAX" ]; then
    _recent=$(journalctl -u alsa-ltc.service --since '-90s' --no-pager 2>/dev/null)
    _eio=$(printf '%s\n' "$_recent" | grep -c 'read from audio interface failed')
    _gap=$(printf '%s\n' "$_recent" | grep -c 'no LTC decoded for')
    # Require an actual -EIO. Do NOT trigger on $_gap alone: a decode gap can occur
    # with a completely healthy controller (clipped input -> LTC decode failure),
    # and on cycle 7 exactly that wasted the one capture we had. $_gap is still
    # recorded in the header for context.
    if [ "${_eio:-0}" -ge 1 ]; then
        echo $((_livecount + 1)) > "$LIVESTATE"
        {
            echo "===== $(ts) LIVE STALL — controller still alive (eio=$_eio gap=$_gap) ====="
            echo "uptime=${uptime_s}s alsa-ltc=$active restarts=$restarts"
            # Registers FIRST — this is the earliest read we can take, and the
            # ONLY chance to see USBSTS/IMAN before the kernel's xhci_halt()
            # clears R/S and overwrites the evidence. The 2026-07-28 capture
            # could not distinguish "controller faulted" from "MSI delivery
            # broke" precisely because this was missing.
            echo "--- xHCI operational registers (DURING stall, pre-halt) ---"
            if [ -f "$REGDUMP" ] && command -v python3 >/dev/null 2>&1; then
                python3 "$REGDUMP" 2>&1 | sed 's/^/  /'
            else
                echo "  unavailable (regdump=$REGDUMP python3=$(command -v python3 || echo missing))"
            fi
            echo "--- xHCI interrupt counters, sample 1 ---"
            grep -iE 'xhci|usb' /proc/interrupts
            sleep 2
            echo "--- xHCI interrupt counters, sample 2 (2s later; FROZEN = HC stopped raising events) ---"
            grep -iE 'xhci|usb' /proc/interrupts
            echo "--- ALSA capture stream state ---"
            for f in status hw_params sw_params; do
                echo "  [pcm0c/sub0/$f]"
                cat /proc/asound/card0/pcm0c/sub0/$f 2>/dev/null | sed 's/^/    /'
            done
            echo "--- USB device state (1-1.1) ---"
            for a in bConfigurationValue bNumInterfaces speed devnum; do
                echo "  $a=$(cat /sys/bus/usb/devices/1-1.1/$a 2>/dev/null)"
            done
            echo "--- VL805 link + config space (valid only while alive) ---"
            lspci -vv -s 0001:01:00.0 2>/dev/null | grep -iE 'LnkSta|LnkCtl|MSI:|Status:'
            lspci -xxx -s 0001:01:00.0 2>/dev/null | head -8
            echo "--- PCIe AER counters ---"
            for f in aer_dev_correctable aer_dev_nonfatal aer_dev_fatal; do
                echo "  [$f]"
                grep -vE ' 0$' "/sys/bus/pci/devices/0001:00:00.0/$f" 2>/dev/null | sed 's/^/    /' || echo "    all zero"
            done
            echo "--- power ---"
            echo "  throttled=$(vcgencmd get_throttled 2>/dev/null | cut -d= -f2) ext5v=$(vcgencmd pmic_read_adc EXT5V_V 2>/dev/null | sed 's/.*=//') temp=$(vcgencmd measure_temp 2>/dev/null)"
            echo "--- dmesg tail (25) ---"
            dmesg -T 2>/dev/null | tail -25
            echo "--- alsa-ltc journal tail (30) ---"
            printf '%s\n' "$_recent" | tail -30
            echo "==================================================================="
        } >> "$LIVELOG" 2>&1
        logger -t alsa-ltc-watchdog "LIVE STALL captured (eio=$_eio gap=$_gap, #$((_livecount + 1))/$LIVEMAX) — HC still alive" 2>/dev/null
    fi
  fi
fi

if [ -z "$REASON" ]; then
    [ "$DRY_RUN" = "1" ] && echo "DRY RUN: healthy — no wedge detected (setif_fails=$fails active=$active restarts=$restarts uptime=${uptime_s}s)"
    rm -f "$STATE"
    exit 0
fi

# DRY_RUN reports and exits WITHOUT touching state, counters, logs or rebooting.
if [ "$DRY_RUN" = "1" ]; then
    echo "DRY RUN: would act — reason=$REASON"
    echo "  setif_fails=$fails active=$active substate=$substate restarts=$restarts uptime=${uptime_s}s"
    echo "  AUTO_RECOVER=$AUTO_RECOVER would_reboot=$([ "$AUTO_RECOVER" = 1 ] && echo yes || echo no)"
    echo "  (no state written, no counter incremented, no reboot)"
    exit 0
fi

# Wedged. Only act once per boot.
[ -f "$STATE" ] && exit 0
touch "$STATE"

STAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p "$ARCHIVE" "$(dirname "$COUNTER")"

# ---- how long did it survive this time? (the key comparison metric) -------
BOOT_EPOCH=$(date -d "$(uptime -s)" +%s 2>/dev/null)
FIRST_EIO=$(journalctl -b -u alsa-ltc.service --no-pager -o short-unix 2>/dev/null \
            | grep -m1 'read from audio interface failed' | cut -d. -f1)
# Fallback 1: the decode gap precedes the first -EIO by ~5 s and is the true
# onset of the URB stall, so prefer it when present.
#
# 2026-07-28: THIS OVERRIDE IS REMOVED. A decode gap is NOT reliably the onset of
# a URB stall. Proven on cycle 7: a 2.1 s `no LTC decoded` gap fired while the HC
# was perfectly healthy (MSI IRQ climbing ~1000/s, hw_ptr advancing) — it was an
# LTC *decode* failure caused by a clipped/railed input (peak=32767), and the
# stream then ran fine for another hour. Preferring the gap made that run report
# survived_s=2350 when it actually failed at 6214 s. Measure to first -EIO only.
FIRST_GAP=$(journalctl -b -u alsa-ltc.service --no-pager -o short-unix 2>/dev/null \
            | grep -m1 'no LTC decoded for' | cut -d. -f1)
# Recorded for reference/diagnostics only — deliberately NOT used for SURVIVED.
# Fallback 2: if userspace logged nothing usable, fall back to the kernel's own
# first sign of trouble (stranded URBs / HC death).
if [ -z "$FIRST_EIO" ]; then
    FIRST_EIO=$(journalctl -b -k --no-pager -o short-unix 2>/dev/null \
                | grep -m1 -E 'active urbs on EP|HC died|assume dead' | cut -d. -f1)
fi
if [ -n "$BOOT_EPOCH" ] && [ -n "$FIRST_EIO" ]; then
    SURVIVED=$((FIRST_EIO - BOOT_EPOCH))
else
    SURVIVED=unknown
fi

# When the first decode gap happened, purely for diagnostics. If this is much
# earlier than SURVIVED, a benign clipping-induced decode gap preceded the real
# stall — useful for telling the two failure modes apart after the fact.
if [ -n "$BOOT_EPOCH" ] && [ -n "$FIRST_GAP" ]; then
    GAP_AT=$((FIRST_GAP - BOOT_EPOCH))
else
    GAP_AT=none
fi

RATE=$(journalctl -b -u alsa-ltc.service --no-pager 2>/dev/null \
       | sed -n 's/.*LTC decoder initialized: sample rate: \([0-9]*\).*/\1/p' | tail -1)
DECODED=$(journalctl -b -u alsa-ltc.service --no-pager 2>/dev/null \
          | sed -n 's/.*\[heartbeat\] [0-9]* frames, \([0-9]*\) ltc decoded.*/\1/p' | tail -1)
PEAK=$(journalctl -b -u alsa-ltc.service --no-pager 2>/dev/null \
       | sed -n 's/.*peak=\([0-9]*\).*/\1/p' | tail -1)

CYCLE=0
[ -f "$COUNTER" ] && CYCLE=$(cat "$COUNTER" 2>/dev/null)
[ -n "$CYCLE" ] || CYCLE=0
CYCLE=$((CYCLE + 1))
echo "$CYCLE" > "$COUNTER"

THR=$(vcgencmd get_throttled 2>/dev/null | cut -d= -f2)
UV=$(dmesg 2>/dev/null | grep -ci 'Undervoltage detected')
E5=$(vcgencmd pmic_read_adc EXT5V_V 2>/dev/null | sed 's/.*=//')

printf '%s cycle=%s reason=%s survived_s=%s rate=%s ltc_decoded=%s peak=%s setif_fails=%s restarts=%s throttled=%s uv_events=%s ext5v=%s\n' \
    "$(ts)" "$CYCLE" "$REASON" "$SURVIVED" "${RATE:-?}" "${DECODED:-?}" "${PEAK:-?}" \
    "$fails" "${restarts:-?}" "$THR" "$UV" "$E5" >> "$INTERVALS"

{
    echo "===== $(ts) WEDGE DETECTED reason=$REASON (cycle $CYCLE/$MAX_RECOVER_CYCLES) ====="
    echo "survived_s=$SURVIVED (to first -EIO) first_decode_gap_s=$GAP_AT rate=$RATE ltc_decoded=$DECODED peak=$PEAK"
    echo "setif_fails(last $DMESG_LINES dmesg lines)=$fails NRestarts=$restarts SubState=$substate"
    echo "power: throttled=$THR ext5v=$E5 uv_events=$UV"
    echo "--- dmesg tail (40) ---";              dmesg -T 2>/dev/null | tail -40
    echo "--- alsa-ltc journal tail (60) ---";   journalctl -b -u alsa-ltc.service -n 60 --no-pager 2>/dev/null
    echo "--- lsusb -t ---";                    lsusb -t 2>/dev/null
    echo "--- /proc/asound/cards ---";          cat /proc/asound/cards 2>/dev/null
    echo "--- usb power state ---"
    for d in usb1 1-1 1-1.1; do
        echo "  $d control=$(cat /sys/bus/usb/devices/$d/power/control 2>/dev/null) status=$(cat /sys/bus/usb/devices/$d/power/runtime_status 2>/dev/null) susp_time=$(cat /sys/bus/usb/devices/$d/power/runtime_suspended_time 2>/dev/null)"
    done
    echo "--- xHCI interrupt counters (compare against live-stall capture) ---"
    grep -iE 'xhci|usb' /proc/interrupts
    # Post-halt register state. Paired with the live-stall dump above, this is
    # what shows whether the controller entered the stuck-not-halted state on
    # its own or only in response to the driver's halt attempt.
    echo "--- xHCI operational registers (POST-mortem, after driver halt) ---"
    if [ -f "$REGDUMP" ] && command -v python3 >/dev/null 2>&1; then
        python3 "$REGDUMP" 2>&1 | sed 's/^/  /'
    else
        echo "  unavailable (regdump=$REGDUMP python3=$(command -v python3 || echo missing))"
    fi
    echo "--- VL805 host controller (0001:01:00.0) ---"
    echo "  present_in_lspci=$(lspci -nn 2>/dev/null | grep -c '1106:3483')"
    echo "  driver=$(readlink /sys/bus/pci/devices/0001:01:00.0/driver 2>/dev/null | sed 's#.*/##' || echo NONE)"
    lspci -vv -s 0001:01:00.0 2>/dev/null | grep -iE 'LnkSta|LnkCtl|MSI:|Status:'
    echo "  config space head (all ff = fell off the bus):"
    lspci -xxx -s 0001:01:00.0 2>/dev/null | head -8
    echo "--- PCIe AER counters (bridge 0001:00:00.0) ---"
    for f in aer_dev_correctable aer_dev_nonfatal aer_dev_fatal; do
        echo "  [$f]"
        grep -vE ' 0$' "/sys/bus/pci/devices/0001:00:00.0/$f" 2>/dev/null || echo "    all zero"
    done
    echo "============================================================"
} >> "$LOG" 2>&1

# Per-episode archive so later cycles don't overwrite earlier evidence.
journalctl -b -u alsa-ltc.service --no-pager > "$ARCHIVE/alsa-ltc-cycle$CYCLE-$STAMP.log" 2>&1
dmesg -T > "$ARCHIVE/dmesg-cycle$CYCLE-$STAMP.log" 2>&1
cp /var/log/piclock-ltc-soak.log "$ARCHIVE/soak-cycle$CYCLE-$STAMP.log" 2>/dev/null
# Preserve the live (pre-death) capture alongside the post-mortem for this cycle.
cp "$LIVELOG" "$ARCHIVE/live-stall-cycle$CYCLE-$STAMP.log" 2>/dev/null

logger -t alsa-ltc-watchdog "USB wedge cycle $CYCLE survived=${SURVIVED}s rate=${RATE}" 2>/dev/null

if [ "$AUTO_RECOVER" != "1" ]; then
    echo "$(ts) AUTO_RECOVER=0 — leaving unit wedged for inspection" >> "$LOG"
    exit 0
fi

if [ "$CYCLE" -ge "$MAX_RECOVER_CYCLES" ] 2>/dev/null; then
    echo "$(ts) cycle cap $MAX_RECOVER_CYCLES reached — NOT rebooting, staying wedged for inspection" >> "$LOG"
    logger -t alsa-ltc-watchdog "cycle cap reached, staying wedged" 2>/dev/null
    exit 0
fi

echo "$(ts) AUTO_RECOVER=1 cycle $CYCLE/$MAX_RECOVER_CYCLES — rebooting to re-arm" >> "$LOG"
logger -t alsa-ltc-watchdog "rebooting to re-arm (cycle $CYCLE/$MAX_RECOVER_CYCLES)" 2>/dev/null
sync
systemctl reboot
