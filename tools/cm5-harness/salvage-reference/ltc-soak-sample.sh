#!/bin/sh
# One soak sample. Run by ltc-soak.timer. Appends a row to the soak CSV so the
# record survives a disconnected laptop.
BASE=/opt/clock8002
CSV=${SOAK_CSV:-$BASE/logs/ltc-soak.csv}
STATE=/run/ltc-soak.last
LABEL=${SOAK_LABEL:-unlabelled}

mkdir -p "$(dirname "$CSV")"
[ -f "$CSV" ] || echo "timestamp,label,board_serial,board_ram_mb,uptime_s,ltc_pid,ltc_nrestarts,pcm_state,hw_ptr,hw_delta,advancing,frames,ltc_decoded,errors,display,avail_mb,swap_mb,temp_c,throttled,clock_rss_kb,clock_swap_kb,ltc_rss_kb,ltc_swap_kb" > "$CSV"

# Per-process VmRSS/VmSwap (KB) for a pid; prints 0 if the pid is missing/gone.
vmfield() {
    pid="$1"; field="$2"
    [ -n "$pid" ] && [ "$pid" != "0" ] && [ -r "/proc/$pid/status" ] || { echo 0; return; }
    awk -v f="$field:" '$1==f{print $2; found=1} END{if(!found) print 0}' "/proc/$pid/status"
}

bserial=$(awk '/^Serial/{print $3}' /proc/cpuinfo 2>/dev/null)
bram=$(awk '/MemTotal/{print int($2/1024)}' /proc/meminfo)

card=$(awk '/USB Audio/{print $1; exit}' /proc/asound/cards 2>/dev/null)
[ -z "$card" ] && card=0
ST=/proc/asound/card$card/pcm0c/sub0/status

pcm_state=$(awk '/^state/{print $2}' "$ST" 2>/dev/null); : "${pcm_state:=CLOSED}"
hw=$(awk '/hw_ptr/{print $3}' "$ST" 2>/dev/null); : "${hw:=-1}"

prev=$(cat "$STATE" 2>/dev/null || echo -1)
echo "$hw" > "$STATE"
if [ "$prev" -ge 0 ] && [ "$hw" -ge 0 ]; then
    delta=$((hw - prev))
else
    delta=-1
fi
# 60s at 44100 Hz should advance ~2.6M frames; anything near zero means stalled
if [ "$delta" -gt 100000 ]; then advancing=yes
elif [ "$delta" -lt 0 ]; then advancing=unknown
else advancing=NO; fi

hb=$(journalctl -u alsa-ltc -b --no-pager 2>/dev/null | grep '\[heartbeat\]' | tail -1)
frames=$(echo "$hb" | sed -n 's/.*\[heartbeat\] \([0-9]*\) frames.*/\1/p')
ltc=$(echo "$hb"    | sed -n 's/.*frames, \([0-9]*\) ltc.*/\1/p')
errs=$(echo "$hb"   | sed -n 's/.*decoded, \([0-9]*\) errors.*/\1/p')

ltc_pid=$(systemctl show -p MainPID --value alsa-ltc)
clock_pid=$(systemctl show -p MainPID --value clock8002)
clock_rss=$(vmfield "$clock_pid" VmRSS)
clock_swap=$(vmfield "$clock_pid" VmSwap)
ltc_rss=$(vmfield "$ltc_pid" VmRSS)
ltc_swap=$(vmfield "$ltc_pid" VmSwap)

echo "$(date '+%Y-%m-%d %H:%M:%S'),\"$LABEL\",$bserial,$bram,$(cut -d. -f1 /proc/uptime),$ltc_pid,$(systemctl show -p NRestarts --value alsa-ltc),$pcm_state,$hw,$delta,$advancing,$frames,$ltc,$errs,$(systemctl is-active clock8002),$(awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo),$(awk '/SwapTotal/{t=$2} /SwapFree/{f=$2} END{print int((t-f)/1024)}' /proc/meminfo),$(awk '{printf "%.1f", $1/1000}' /sys/class/thermal/thermal_zone0/temp 2>/dev/null),$(vcgencmd get_throttled 2>/dev/null | cut -d= -f2),$clock_rss,$clock_swap,$ltc_rss,$ltc_swap" >> "$CSV"
