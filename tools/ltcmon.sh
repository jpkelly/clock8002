#!/bin/sh
# ltcmon v3 — enhanced alsa-ltc/USB/ALSA monitor (reconstructed)
# Tags: GAP/*, KERN/*, PCM/STATE|STALL, URB/STALL, USB/SUSPEND|RESUME,
#       STATE, RESTART, APP_ERR, HEALTH, heartbeat
# Logs to stdout; wrap with nohup ... > /tmp/ltcmon.log 2>&1 &

ts() { date +%H:%M:%S; }
log() { echo "$(ts) $*"; }

detect_card() {
  awk -F'[][]' '/USB-Audio/ {gsub(/ /,"",$1); print $1; exit}' /proc/asound/cards
}
detect_usb_dev() {
  for d in /sys/bus/usb/devices/*/sound; do
    [ -d "$d" ] || continue
    p=$(dirname "$d")
    basename "$p" | awk -F: '{print $1}'
    return
  done
}

CARD=$(detect_card)
USBDEV=$(detect_usb_dev)
[ -z "$CARD" ] && CARD=1
[ -z "$USBDEV" ] && USBDEV=1-1.1
USBPATH="/sys/bus/usb/devices/$USBDEV"
PCMSTAT="/proc/asound/card${CARD}/pcm0c/sub0/status"
URBNUM="$USBPATH/urbnum"
RSTAT="$USBPATH/power/runtime_status"
SUSPTIME="$USBPATH/power/runtime_suspended_time"
CURSOR=/tmp/ltcmon.dmesg.cursor
ALSA_LOG=/tmp/alsa-ltc.log

log "[MON/START] ltcmon v3 card=$CARD usbdev=$USBDEV pcm=$PCMSTAT"
log "[MON/START] pid=$$ alsa_log=$ALSA_LOG"

DMESG_PREV=$(dmesg 2>/dev/null | wc -l)
echo "$DMESG_PREV" > "$CURSOR"

# ---------- Pollers ----------
# 1) PCM state + hw_ptr (1s)
(
  prev_state=""; prev_hw=""; stall=0
  while :; do
    if [ -r "$PCMSTAT" ]; then
      state=$(awk '/^state/ {print $NF}' "$PCMSTAT")
      hw=$(awk '/^hw_ptr/ {print $NF}' "$PCMSTAT")
      [ -z "$state" ] && state="closed"
      [ -z "$hw" ] && hw=0
      if [ "$state" != "$prev_state" ]; then
        log "[PCM/STATE] $prev_state -> $state hw_ptr=$hw"
        prev_state=$state
      fi
      if [ "$state" = "RUNNING" ] && [ "$hw" = "$prev_hw" ]; then
        stall=$((stall+1))
        [ "$stall" = 3 ] && log "[PCM/STALL] hw_ptr frozen at $hw for ${stall}s state=$state"
      else
        stall=0
      fi
      prev_hw=$hw
    else
      if [ -n "$prev_state" ]; then
        log "[PCM/STATE] $prev_state -> MISSING"
        prev_state=""
      fi
    fi
    sleep 1
  done
) &
P_PCM=$!

# 2) URB + runtime_status (1s)
(
  prev_urb=""; prev_rs=""; urb_stall=0
  while :; do
    urb=""; rs=""
    [ -r "$URBNUM" ] && urb=$(cat "$URBNUM" 2>/dev/null)
    [ -r "$RSTAT" ] && rs=$(cat "$RSTAT" 2>/dev/null)
    if [ -n "$rs" ] && [ "$rs" != "$prev_rs" ]; then
      case "$rs" in
        suspended) log "[USB/SUSPEND] prev=$prev_rs" ;;
        active)    [ -n "$prev_rs" ] && log "[USB/RESUME] prev=$prev_rs" ;;
      esac
      prev_rs=$rs
    fi
    if [ -n "$urb" ] && [ -n "$prev_urb" ]; then
      if [ "$urb" = "$prev_urb" ] && pidof alsa-ltc >/dev/null 2>&1; then
        urb_stall=$((urb_stall+1))
        [ "$urb_stall" = 3 ] && log "[URB/STALL] urbnum frozen at $urb for ${urb_stall}s rs=$rs"
      else
        urb_stall=0
      fi
    fi
    prev_urb=$urb
    sleep 1
  done
) &
P_URB=$!

# 3) dmesg delta (5s)
(
  while :; do
    cur=$(dmesg 2>/dev/null | wc -l)
    prev=$(cat "$CURSOR" 2>/dev/null || echo 0)
    if [ "$cur" -gt "$prev" ]; then
      delta=$((cur - prev))
      dmesg 2>/dev/null | tail -n "$delta" | while IFS= read -r line; do
        case "$line" in
          *"cannot set freq"*)                 log "[KERN/FREQ_FAIL] $line" ;;
          *"usb_set_interface failed"*)        log "[KERN/IFACE_FAIL] $line" ;;
          *"underrun"*|*"xrun"*|*"XRUN"*)      log "[KERN/XRUN] $line" ;;
          *"HC died"*)                         log "[KERN/HC_DIED] $line" ;;
          *"xhci"*"dead"*)                     log "[KERN/HC_DIED] $line" ;;
          *"USB disconnect"*)                  log "[KERN/DISCONNECT] $line" ;;
          *"reset high-speed"*|*"reset full-speed"*|*"reset low-speed"*) log "[KERN/RESET] $line" ;;
          *"failed to complete pause on dma"*) log "[KERN/DMA_PAUSE] $line" ;;
        esac
      done
      echo "$cur" > "$CURSOR"
    fi
    sleep 5
  done
) &
P_DMESG=$!

# 4) process state + restart (10s)
(
  prev_pid=""; prev_st=""
  while :; do
    pid=$(pidof alsa-ltc 2>/dev/null | awk '{print $1}')
    if [ -n "$pid" ] && [ "$pid" != "$prev_pid" ]; then
      [ -n "$prev_pid" ] && log "[RESTART] alsa-ltc pid $prev_pid -> $pid"
      prev_pid=$pid
    fi
    if [ -n "$pid" ] && [ -r "/proc/$pid/status" ]; then
      st=$(awk '/^State:/ {print $2}' "/proc/$pid/status")
      if [ "$st" != "$prev_st" ]; then
        log "[STATE] alsa-ltc pid=$pid state=$st"
        prev_st=$st
      fi
    fi
    sleep 10
  done
) &
P_STATE=$!

# 5) HEALTH + heartbeat (10s)
(
  prev_susp=$(cat "$SUSPTIME" 2>/dev/null || echo 0)
  prev_xhci=$(awk '/xhci/ {for(i=2;i<=NF-2;i++)s+=$i} END{print s+0}' /proc/interrupts)
  tick=0
  while :; do
    sleep 10
    pid=$(pidof alsa-ltc 2>/dev/null | awk '{print $1}')
    st="?"
    [ -n "$pid" ] && [ -r "/proc/$pid/status" ] && st=$(awk '/^State:/ {print $2}' "/proc/$pid/status")
    pcm="closed"; hw=0
    if [ -r "$PCMSTAT" ]; then
      pcm=$(awk '/^state/ {print $NF}' "$PCMSTAT")
      hw=$(awk '/^hw_ptr/ {print $NF}' "$PCMSTAT")
    fi
    urb=$(cat "$URBNUM" 2>/dev/null || echo 0)
    rs=$(cat "$RSTAT" 2>/dev/null || echo "?")
    susp=$(cat "$SUSPTIME" 2>/dev/null || echo 0)
    susp_dms=$((susp - prev_susp)); prev_susp=$susp
    xhci_now=$(awk '/xhci/ {for(i=2;i<=NF-2;i++)s+=$i} END{print s+0}' /proc/interrupts)
    xhci_d=$((xhci_now - prev_xhci)); prev_xhci=$xhci_now
    load=$(awk '{print $1}' /proc/loadavg)
    temp=0
    [ -r /sys/class/thermal/thermal_zone0/temp ] && temp=$(( $(cat /sys/class/thermal/thermal_zone0/temp) / 1000 ))
    freq_errs=$(grep -c '\[KERN/FREQ_FAIL\]' /tmp/ltcmon.log 2>/dev/null)
    iface_errs=$(grep -c '\[KERN/IFACE_FAIL\]' /tmp/ltcmon.log 2>/dev/null)
    [ -z "$freq_errs" ] && freq_errs=0
    [ -z "$iface_errs" ] && iface_errs=0
    log "[HEALTH] pid=${pid:-none} state=$st pcm=$pcm hw_ptr=$hw urbn=$urb rs=$rs susp_dms=$susp_dms xhci_d=$xhci_d load=$load temp=${temp}C freq_errs=$freq_errs iface_errs=$iface_errs"
    tick=$((tick+1))
    [ $((tick % 6)) -eq 0 ] && log "[heartbeat] up"
  done
) &
P_HEALTH=$!

cleanup() {
  log "[MON/STOP] shutting down"
  kill $P_PCM $P_URB $P_DMESG $P_STATE $P_HEALTH 2>/dev/null
  exit 0
}
trap cleanup INT TERM

# ---------- Foreground: alsa-ltc.log gap classifier ----------
[ -f "$ALSA_LOG" ] || : > "$ALSA_LOG"
tail -F "$ALSA_LOG" 2>/dev/null | while IFS= read -r line; do
  case "$line" in
    *"[gap]"*)
      peak=$(echo "$line" | sed -n 's/.*peak_during_gap=\([0-9]*\).*/\1/p')
      dur=$(echo  "$line" | sed -n 's/.*no LTC decoded for \([0-9]*\)ms.*/\1/p')
      [ -z "$peak" ] && peak=0
      [ -z "$dur" ] && dur=0
      if [ "$peak" -lt 100 ]; then tag="SILENCE"
      elif [ "$peak" -gt 20000 ]; then tag="INVALID_LTC"
      else tag="PARTIAL"
      fi
      log "[GAP/$tag] dur=${dur}ms peak=$peak"
      ;;
    *ERROR*|*error*|*Err*) log "[APP_ERR] $line" ;;
  esac
done

cleanup
