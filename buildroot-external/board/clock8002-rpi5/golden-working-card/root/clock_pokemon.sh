#!/bin/sh

start() {
	echo -"Enabling SPI support"
	modprobe spidev
	modprobe spi-bcm2835
	modprobe snd_bcm2835
	echo -"Starting Clock"
	mount -o remount,rw /boot
	cd /root
	while true
	do
		echo -e "\033[9;0]"
		/root/clock_cmd.sh
		echo "CRASHED!"
		echo 15 > /sys/class/gpio/unexport
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
