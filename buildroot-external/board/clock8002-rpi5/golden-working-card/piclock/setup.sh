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

# Restore or generate stable machine-id (persisted at /boot/.piclock-machine-id)
mkdir -p /var/lib/dbus
if [ -s /boot/.piclock-machine-id ]; then
	cp -f /boot/.piclock-machine-id /var/lib/dbus/machine-id
else
	tr -d '-' < /proc/sys/kernel/random/uuid | tr 'A-Z' 'a-z' \
		> /var/lib/dbus/machine-id
	cp -f /var/lib/dbus/machine-id /boot/.piclock-machine-id 2>/dev/null || true
fi
chmod 0444 /var/lib/dbus/machine-id 2>/dev/null || true

# Install and start power button handler
if [ -f /boot/power-button.sh ]; then
	cp /boot/power-button.sh /root/power-button.sh
	chmod +x /root/power-button.sh
	/root/power-button.sh &
fi

# Apply network.ini configuration
if [ -f /boot/piclock/network.ini ]; then
	_ini_get() {
		awk -F= -v s="$1" -v k="$2" \
			'/^\[/{cur=substr($0,2,index($0,"]")-2)} cur==s&&$1==k{gsub(/[[:space:]]/,"",$2);print $2;exit}' \
			/boot/piclock/network.ini
	}
	_to_mask() {
		case "$1" in
			*.*.*.*)  echo "$1" ;;
			8)  echo "255.0.0.0" ;;
			16) echo "255.255.0.0" ;;
			24) echo "255.255.255.0" ;;
			32) echo "255.255.255.255" ;;
			*)  echo "255.255.255.0" ;;
		esac
	}

	_net_hostname=$(_ini_get host hostname)
	if [ -n "$_net_hostname" ]; then
		hostname "$_net_hostname"
		# avahi-daemon starts at S50 before this script; restart it to pick up the new hostname
		/etc/init.d/S50avahi-daemon stop 2>/dev/null || true
		/etc/init.d/S50avahi-daemon start 2>/dev/null || true
	fi

	_net_mode=$(_ini_get network mode)
	[ -z "$_net_mode" ] && _net_mode=dhcp
	_net_addr=$(_ini_get network address)
	_net_mask=$(_to_mask "$(_ini_get network netmask)")
	_net_gw=$(_ini_get network gateway)
	_net_dns=$(_ini_get network dns)

	if { [ "$_net_mode" = "static" ] || [ "$_net_mode" = "dual" ]; } && \
	   { [ -z "$_net_addr" ] || [ -z "$_net_mask" ]; }; then
		_net_mode=dhcp
	fi

	{
		echo "# Generated from /boot/piclock/network.ini by setup.sh"
		echo "auto lo"
		echo "iface lo inet loopback"
		echo ""
		case "$_net_mode" in
			static)
				echo "auto eth0"
				echo "iface eth0 inet static"
				echo "    address $_net_addr"
				echo "    netmask $_net_mask"
				[ -n "$_net_gw" ] && echo "    gateway $_net_gw"
				[ -n "$_net_dns" ] && echo "    dns-nameservers $_net_dns"
				;;
			dual)
				echo "auto eth0"
				echo "iface eth0 inet dhcp"
				echo ""
				echo "iface eth0:1 inet static"
				echo "    address $_net_addr"
				echo "    netmask $_net_mask"
				[ -n "$_net_gw" ] && echo "    gateway $_net_gw"
				;;
			*)
				echo "auto eth0"
				echo "iface eth0 inet dhcp"
				;;
		esac
	} > /etc/network/interfaces

	ip addr flush dev eth0 2>/dev/null || true
	ifup eth0 2>/dev/null || true
	# In dual mode, add the static alias directly via ip rather than ifup eth0:1.
	# ifup eth0 spawns udhcpc in the background and returns before eth0 is marked
	# "up" in the ifupdown state file, causing ifup eth0:1 to fail silently at boot.
	if [ "$_net_mode" = "dual" ]; then
		_raw_mask=$(_ini_get network netmask)
		ip addr add "${_net_addr}/${_raw_mask}" dev eth0 2>/dev/null || true
	fi

	# NTP control
	_ntp=$(_ini_get network ntp)
	if [ "$_ntp" = "false" ]; then
		/etc/init.d/S49ntp stop 2>/dev/null || true
	elif [ "$_ntp" = "true" ]; then
		/etc/init.d/S49ntp restart 2>/dev/null || true
	fi
fi
