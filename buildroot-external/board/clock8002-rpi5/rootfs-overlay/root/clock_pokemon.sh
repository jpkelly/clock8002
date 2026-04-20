#!/bin/sh
# sdl-clock pokemon watchdog — restarts sdl3-clock in a tight loop on crash.

start() {
	echo "Starting sdl-clock watchdog"
	export SDL_VIDEODRIVER=kmsdrm
	export XDG_RUNTIME_DIR=/run/user/0
	mkdir -p /run/user/0

	# Wait for KMS/DRM device to be ready before first launch.
	_drm_retries=0
	while [ "$_drm_retries" -lt 30 ]; do
		ls /dev/dri/card* >/dev/null 2>&1 && break
		_drm_retries=$((_drm_retries + 1))
		sleep 1
	done
	echo "DRM device ready after ${_drm_retries}s"

	cd /opt/clock8002
	while true; do
		/root/clock_cmd.sh >> /tmp/clock.log 2>&1
		echo "$(date): sdl-clock exited ($?), restarting in 2s..." >> /tmp/clock.log
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
