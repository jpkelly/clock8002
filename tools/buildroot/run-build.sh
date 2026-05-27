#!/bin/sh
# run-build.sh — Self-finalizing Buildroot build wrapper.
#
# Records --start before make, runs make, then always records --finish
# with the actual exit code. Use this from a screen session.
#
# Usage:
#   run-build.sh <session-label> [make-args...]
#
# Examples:
#   run-build.sh 24c605d-dirclean clock8002-dirclean
#   run-build.sh 24c605d-dirclean clock8002-dirclean make
#   BR2_INCLUDE_AUTHORIZED_KEYS=1 BR2_PICLOCKKEY="$(cat ~/.ssh/id_rsa.pub)" \
#       run-build.sh 24c605d-keyed clock8002-dirclean make
#
# Env:
#   SRC_REPO  (default: ~/clock8002-root-ram)
#   BR_DIR    (default: ~/buildroot)
#   OUTPUT_DIR (default: $BR_DIR/output)

set +e

LABEL="${1:-}"
if [ -z "$LABEL" ]; then
    echo "ERROR: session label required" >&2
    echo "Usage: run-build.sh <session-label> [make-args...]" >&2
    exit 2
fi
shift

SRC_REPO="${SRC_REPO:-${HOME}/clock8002-root-ram}"
BR_DIR="${BR_DIR:-${HOME}/buildroot}"
OUTPUT_DIR="${OUTPUT_DIR:-${BR_DIR}/output}"
TOOLS_DIR="${SRC_REPO}/tools/buildroot"
LOG="/tmp/br-${LABEL}.log"
EXIT_MARKER="/tmp/br-${LABEL}.exit"

# Default to a sensible target if none specified
if [ $# -eq 0 ]; then
    set -- make
fi

# Snapshot the build start state
sh "${TOOLS_DIR}/manifest-snapshot.sh" --start "${OUTPUT_DIR}" \
    --src "${SRC_REPO}" --br "${BR_DIR}" \
    --target "$*" 2>&1 | tee -a "${LOG}"

cd "${BR_DIR}" || { echo "Cannot cd to ${BR_DIR}" | tee -a "${LOG}"; exit 1; }

# Run each make target in sequence; bail on first failure
RC=0
for target in "$@"; do
    echo "=== make ${target} ===" | tee -a "${LOG}"
    make "${target}" 2>&1 | tee -a "${LOG}"
    RC=$?
    if [ "$RC" -ne 0 ]; then
        echo "make ${target} FAILED with exit ${RC}" | tee -a "${LOG}"
        break
    fi
done

# Always finalize, even on failure — this is the whole point of the wrapper
sh "${TOOLS_DIR}/manifest-snapshot.sh" --finish "${OUTPUT_DIR}" "${RC}" 2>&1 | tee -a "${LOG}"

echo "BR_BUILD_EXIT:${RC}" | tee -a "${LOG}"
echo "${RC}" > "${EXIT_MARKER}"
exit "${RC}"
