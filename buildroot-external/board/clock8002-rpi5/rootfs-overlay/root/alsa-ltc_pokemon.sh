#!/bin/sh
# alsa-ltc pokemon watchdog — mirrors 3rd party alsa-ltc_pokemon.sh pattern.
# Loads snd_usb_audio once, then restarts alsa-ltc in a tight loop on crash.

start() {
	echo "Loading snd_usb_audio kernel module"
	modprobe snd_usb_audio 2>/dev/null || true

	# Wait for USB audio card to appear
	i=0
	while [ $i -lt 30 ]; do
		grep -qE "USB.Audio|USB Audio" /proc/asound/cards 2>/dev/null && break
		sleep 1
		i=$((i + 1))
	done

	# Detect USB audio card number dynamically (avoids hardcoded plughw:N,0)
	ALSA_CARD=$(grep -E "^ *[0-9]" /proc/asound/cards 2>/dev/null \
		| grep -E "USB.Audio|USB Audio" \
		| awk '{print $1}' | head -1)
	[ -z "$ALSA_CARD" ] && ALSA_CARD=2
	export ALSA_CARD
	echo "Using USB audio card: $ALSA_CARD"

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
