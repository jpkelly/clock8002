#!/bin/sh
# sdl-clock pokemon watchdog — restarts sdl3-clock in a tight loop on crash.

start() {
	echo "Starting sdl-clock watchdog"
	export HOME=/root
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

	# Kill bootsplash (fbv) so it releases the framebuffer before we take DRM master.
	kill $(pidof fbv) 2>/dev/null || true

	# Release fbcon's hold on the DRM device so sdl3-clock can acquire master.
	# fbcon binds to vtcon1 at boot via vc4drmfb; without this sdl3-clock
	# cannot get DRM master and exits immediately with code 1.
	echo 0 > /sys/class/vtconsole/vtcon1/bind 2>/dev/null || true

	cd /opt/clock8002
	while true; do
		/root/clock_cmd.sh >> /tmp/clock.log 2>&1
		echo "$(date): sdl-clock exited ($?), restarting in 2s..." >> /tmp/clock.log
		sleep 2
	done
}

stop() {
	echo "Stopping sdl-clock watchdog"
	kill $(cat /var/run/clock.pid 2>/dev/null) 2>/dev/null || true
	for pid in $(ps aux | grep clock_pokemon | grep -v grep | awk '{print $1}'); do
		[ "$pid" != "$$" ] && kill "$pid" 2>/dev/null || true
	done
	kill $(ps aux | grep clock_cmd | grep -v grep | awk '{print $1}') 2>/dev/null || true
	kill $(ps aux | grep sdl3-clock | grep -v grep | awk '{print $1}') 2>/dev/null || true
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
