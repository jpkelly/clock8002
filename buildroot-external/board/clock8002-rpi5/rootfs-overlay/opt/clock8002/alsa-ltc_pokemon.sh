#!/bin/sh

start() {
	echo -"Starting audio to OSC SMTP LTC converter"
	echo -"Loading kernel modules for hifiberry ADC+ DAC pro"
	modprobe i2c_bcm2835
	modprobe snd_soc_bcm2835_i2s
	modprobe snd_soc_pcm186x_i2c
	modprobe snd_soc_bcm2835_i2s
	modprobe clk_hifiberry_dacpro
	modprobe snd_soc_pcm512x_i2c
	modprobe snd_soc_hifiberry_dacplusadcpro
	echo -"Loading kernel modules for interspace industries usb aio"
	modprobe snd_usb_audio

	cd /root
	while true
	do
		echo -e "\033[9;0]"
		/root/alsa-ltc_cmd.sh
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
		;;
esac

exit $?
