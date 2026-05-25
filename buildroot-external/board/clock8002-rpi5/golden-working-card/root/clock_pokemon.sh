#!/bin/sh

# Read a key from /boot/piclock/piclock.ini.
_piclock_get() {
	[ -f /boot/piclock/piclock.ini ] || return 1
	awk -F= "/^$1/ { gsub(/[[:space:]]/, \"\", \$2); print \$2 }" /boot/piclock/piclock.ini
}

start() {
	echo -"Enabling SPI support"
	modprobe spidev
	modprobe spi-bcm2835
	modprobe snd_bcm2835
	echo -"Starting Clock"
	mount -o remount,rw /boot

	# Run user setup script if present (authorized_keys, OLED, etc.)
	if [ -f /boot/piclock/setup.sh ]; then
		sh /boot/piclock/setup.sh
	fi

	cd /root
	while true
	do
		echo -e "\033[9;0]"
		# When bootsplash is enabled, detach the framebuffer console (vtcon1)
		# from fb0 before painting so that no subsequent console output —
		# from concurrent init scripts, module loading, or SDL startup —
		# can overwrite the display.  Then paint the splash image.
		if [ "$(_piclock_get splash_enabled)" = "true" ] && [ -f /boot/bootsplash.raw ]; then
			echo 0 > /sys/class/vtconsole/vtcon1/bind 2>/dev/null || true
			dd if=/boot/bootsplash.raw of=/dev/fb0 bs=4096 2>/dev/null || true
		fi
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
