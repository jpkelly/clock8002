#!/bin/sh
# Wait for vc4 DRM to take over fb0 (replaces simplefb)
i=0
while [ $i -lt 50 ]; do
    if grep -q vc4drmfb /sys/class/graphics/fb0/name 2>/dev/null; then
        exec /usr/bin/fbv -f -i -e /opt/clock8002/bootsplash.png
    fi
    sleep 0.1
    i=$((i + 1))
done
# Fallback: try anyway
exec /usr/bin/fbv -f -i -e /opt/clock8002/bootsplash.png
