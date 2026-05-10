#!/bin/sh

start() {
	echo -"Enabling SPI support"
	modprobe spidev
	modprobe spi-bcm2835
	modprobe snd_bcm2835
	echo -"Starting Clock"
	mount -o remount,rw /boot

	# Install SSH authorized_keys from boot partition
	if [ -f /boot/piclock/authorized_keys ]; then
		mkdir -p /root/.ssh
		chmod 700 /root/.ssh
		cp /boot/piclock/authorized_keys /root/.ssh/authorized_keys
		chmod 600 /root/.ssh/authorized_keys
	fi

	# Start OLED daemon
	if [ -x /opt/clock8002/oled/oled_daemon.py ]; then
		/opt/clock8002/oled/oled_daemon.py &
	fi

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
