#!/bin/sh
set -eu

BINARIES_DIR="$1"

# Placeholder for future image assembly hooks (firmware files, boot partition
# layout, and clock/network config injection).
echo "clock8002 post-image hook complete: ${BINARIES_DIR}"
