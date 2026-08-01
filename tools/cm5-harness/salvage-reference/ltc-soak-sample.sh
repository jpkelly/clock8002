#!/bin/sh
# One soak sample. Run by ltc-soak.timer. Appends a row to the soak CSV so the
# record survives a disconnected laptop.
BASE=/opt/clock8002
CSV=${SOAK_CSV:-$BASE/logs/ltc-soak.csv}
STATE=/run/ltc-soak.last
LABEL=${SOAK_LABEL:-unlabelled}

mkdir -p "$(dirname "$CSV")"
[ -f "$CSV" ] || echo "timestamp,label,board_serial,board_ram_mb,uptime_s,ltc_pid,ltc_nrestarts,pcm_state,hw_ptr,hw_delta,advancing,frames,ltc_decoded,errors,display,avail_mb,swap_mb,temp_c,throttled" > "$CSV"

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

echo "$(date '+%Y-%m-%d %H:%M:%S'),\"$LABEL\",$bserial,$bram,$(cut -d. -f1 /proc/uptime),$(systemctl show -p MainPID --value alsa-ltc),$(systemctl show -p NRestarts --value alsa-ltc),$pcm_state,$hw,$delta,$advancing,$frames,$ltc,$errs,$(systemctl is-active clock8002),$(awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo),$(awk '/SwapTotal/{t=$2} /SwapFree/{f=$2} END{print int((t-f)/1024)}' /proc/meminfo),$(awk '{printf "%.1f", $1/1000}' /sys/class/thermal/thermal_zone0/temp 2>/dev/null),$(vcgencmd get_throttled 2>/dev/null | cut -d= -f2)" >> "$CSV"
