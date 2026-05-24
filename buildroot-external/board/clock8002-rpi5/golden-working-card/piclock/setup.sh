#!/bin/sh
# /boot/piclock/setup.sh — runs at boot as root, /boot already mounted rw.
# Place this file on the FAT boot partition at piclock/setup.sh.

# Install SSH authorized_keys
if [ -f /boot/piclock/authorized_keys ]; then
	mkdir -p /root/.ssh
	chmod 700 /root/.ssh
	cp /boot/piclock/authorized_keys /root/.ssh/authorized_keys
	chmod 600 /root/.ssh/authorized_keys
fi

# Boot splash: write raw RGB565 image directly to framebuffer if enabled.
_piclock_ini_get() {
	[ -f /boot/piclock/piclock.ini ] || return 1
	val=$(awk -F= "/^$1/ { gsub(/[[:space:]]/, \"\", \$2); print \$2 }" /boot/piclock/piclock.ini)
	echo "${val}"
}
if [ "$(_piclock_ini_get splash_enabled)" = "true" ] && [ -f /boot/piclock/bootsplash.raw ]; then
	dd if=/boot/piclock/bootsplash.raw of=/dev/fb0 bs=4096 2>/dev/null || true
fi

# Start OLED daemon
if [ -x /boot/oled-daemon ]; then
	modprobe i2c-dev 2>/dev/null || true
	/boot/oled-daemon &
fi

# Install and start power button handler
if [ -f /boot/power-button.sh ]; then
	cp /boot/power-button.sh /root/power-button.sh
	chmod +x /root/power-button.sh
	/root/power-button.sh &
fi
