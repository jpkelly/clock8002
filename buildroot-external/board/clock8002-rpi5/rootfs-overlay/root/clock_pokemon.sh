#!/bin/sh
# sdl-clock pokemon watchdog — restarts sdl3-clock in a tight loop on crash.

start() {
	echo "Starting sdl-clock watchdog"
	export SDL_VIDEODRIVER=kmsdrm
	export XDG_RUNTIME_DIR=/run/user/0
	mkdir -p /run/user/0

	cd /opt/clock8002
	while true; do
		/root/clock_cmd.sh
		echo "sdl-clock exited, restarting in 2s..."
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
