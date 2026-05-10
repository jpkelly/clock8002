#!/usr/bin/env bash

# Robust cm5 build status probe that avoids fragile inline SSH quoting.
#
# Usage examples:
#   tools/buildroot/cm5-build-status.sh --session br-build-root-ram-payload-20260509-165344
#   tools/buildroot/cm5-build-status.sh --session br-build-root-ram-payload-20260509-165344 --tail 80
#   tools/buildroot/cm5-build-status.sh --session br-build-root-ram-payload-20260509-165344 --output /home/pi/output-root-ram-payload-20260509-165344

set -euo pipefail

HOST="pi@cm5.local"
SESSION=""
OUT_DIR=""
TAIL_LINES=120

usage() {
  cat <<'EOF'
Usage: cm5-build-status.sh --session <session> [options]

Required:
  --session <name>       Build session name, e.g. br-build-root-ram-payload-20260509-165344

Options:
  --output <path>        Output dir on cm5 (default: /home/pi/output-${session#br-build-})
  --host <user@host>     SSH host (default: pi@cm5.local)
  --tail <n>             Log tail line count (default: 120)
  -h, --help             Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --session)
      SESSION="${2:-}"
      shift 2
      ;;
    --output)
      OUT_DIR="${2:-}"
      shift 2
      ;;
    --host)
      HOST="${2:-}"
      shift 2
      ;;
    --tail)
      TAIL_LINES="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$SESSION" ]]; then
  echo "Missing required --session" >&2
  usage
  exit 2
fi

if [[ -z "$OUT_DIR" ]]; then
  OUT_DIR="/home/pi/output-${SESSION#br-build-}"
fi

if ! [[ "$TAIL_LINES" =~ ^[0-9]+$ ]]; then
  echo "--tail must be an integer" >&2
  exit 2
fi

ssh "$HOST" sh -s -- "$SESSION" "$OUT_DIR" "$TAIL_LINES" <<'REMOTE'
set +e

SESSION="$1"
OUT_DIR="$2"
TAIL_LINES="$3"

LOG="/tmp/${SESSION}.log"
EXITF="/tmp/${SESSION}.exit"

echo "SESSION:${SESSION}"
echo "OUTPUT_DIR:${OUT_DIR}"
echo "LOG:${LOG}"
echo "EXIT_FILE:${EXITF}"

echo '---STATE---'
if [ -f "$EXITF" ]; then
  echo DONE
  cat "$EXITF"
else
  echo RUNNING
fi

echo '---TIMING---'
if [ -f "$LOG" ]; then
  START_TIME=$(sed -n 's/^START_TIME://p' "$LOG" | head -n1)
  echo "START_TIME:${START_TIME}"
  NOW_EPOCH=$(date +%s)
  START_EPOCH=$(date -d "$START_TIME" +%s 2>/dev/null || echo 0)
  if [ "$START_EPOCH" -gt 0 ]; then
    ELAPSED_SEC=$((NOW_EPOCH - START_EPOCH))
    echo "ELAPSED_SEC:${ELAPSED_SEC}"
  else
    echo 'ELAPSED_SEC:unknown'
  fi
else
  echo 'START_TIME:unknown'
  echo 'ELAPSED_SEC:unknown'
fi

echo '---ACTIVE_MAKE---'
pgrep -af "make O=${OUT_DIR}" | sed -n '1,80p' || true

echo '---LAST_STAMP---'
LAST_STAMP=$(find "$OUT_DIR/build" -name '.stamp_*' -printf '%TY-%Tm-%Td %TH:%TM:%TS %p\n' 2>/dev/null | sort | tail -n 1)
echo "$LAST_STAMP"
if [ -n "$LAST_STAMP" ]; then
  LAST_PKG=$(echo "$LAST_STAMP" | sed -E 's#.*build/([^/]+)/\.stamp_.*#\1#')
  echo "LAST_PACKAGE:${LAST_PKG}"
fi

echo '---STRICT_KERNEL_COMPILE_LINES---'
grep -nE '^>>>[[:space:]]+linux|^>>>[[:space:]]+linux-custom|[[:space:]]vmlinux[[:space:]]|modules_install|INSTALL_MOD' "$LOG" 2>/dev/null | tail -n 40 || true

echo '---LOG_TAIL---'
tail -n "$TAIL_LINES" "$LOG" 2>/dev/null || true
REMOTE
