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

# Start OLED daemon
if [ -x /opt/clock8002/oled/oled_daemon.py ]; then
	/opt/clock8002/oled/oled_daemon.py &
fi
