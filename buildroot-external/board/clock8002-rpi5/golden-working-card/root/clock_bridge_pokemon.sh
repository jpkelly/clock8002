#!/bin/sh

start() {
	echo -"Starting Clock mitti and millumin bridge"
	cd /root
	while true
	do
		echo -e "\033[9;0]"
		/root/clock_bridge_cmd.sh
		echo "CRASHED!"
		sleep 2
	done

}

stop() {
	true
}

restart() {
	stop
	start
}

case "$1" in
	start)
		start
		;;
	stop)
		stop
	;;
	restart|reload)
		restart
		;;
	*)
		echo "Usage: $0 {start|stop|restart}"
		exit 1
esac

exit $?
