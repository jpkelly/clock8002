#!/bin/sh
# alsa-ltc pokemon watchdog — mirrors 3rd party alsa-ltc_pokemon.sh pattern.
# Loads snd_usb_audio once, then restarts alsa-ltc in a tight loop on crash.

start() {
	echo "Loading snd_usb_audio kernel module"
	modprobe snd_usb_audio 2>/dev/null || true

	# alsa-ltc is invoked with '-' (auto-detect mode) — it handles device
	# detection and timing internally, polling every 1s to catch the USB
	# audio device within the Pi 5 xHCI 20s isochronous-ready window.
	# Do NOT add a settle delay here; it causes the open to miss that window.

	cd /opt/clock8002
	while true; do
		/root/alsa-ltc_cmd.sh
		echo "alsa-ltc exited, restarting in 2s..."
		sleep 2
	done
}

stop() {
	true
}

case "$1" in
	start)
		start
		;;
	stop|restart|reload)
		stop
		;;
	*)
		echo "Usage: $0 {start|stop|restart}"
		exit 1
		;;
esac

exit $?
