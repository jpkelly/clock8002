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

# SSH host key persistence — keep the same Dropbear host key across reboots.
# /boot is already mounted rw. The embedded init generates a fresh random key
# every boot; on first boot we save it to FAT and on subsequent boots we
# restore it and restart the listener so known_hosts stays valid.
_SSH_STORE="/boot/piclock/ssh"
_SSH_KEY="dropbear_ed25519_host_key"
_DB_KEYDIR="/etc/dropbear"
if [ -f "$_SSH_STORE/$_SSH_KEY" ]; then
	# Restore persisted key and restart the Dropbear listener with it
	cp "$_SSH_STORE/$_SSH_KEY" "$_DB_KEYDIR/$_SSH_KEY"
	chmod 600 "$_DB_KEYDIR/$_SSH_KEY"
	_dbpid=$(pidof dropbear 2>/dev/null | tr ' ' '\n' | sort -n | head -1)
	[ -n "$_dbpid" ] && kill "$_dbpid" 2>/dev/null || true
	sleep 1
	dropbear -R
else
	# First boot — Dropbear with -R generates its key lazily on first connection,
	# so we generate it explicitly here, persist to FAT, and restart the listener.
	mkdir -p "$_DB_KEYDIR"
	[ -f "$_DB_KEYDIR/$_SSH_KEY" ] || dropbearkey -t ed25519 -f "$_DB_KEYDIR/$_SSH_KEY" 2>/dev/null || true
	if [ -f "$_DB_KEYDIR/$_SSH_KEY" ]; then
		chmod 600 "$_DB_KEYDIR/$_SSH_KEY"
		mkdir -p "$_SSH_STORE"
		cp "$_DB_KEYDIR/$_SSH_KEY" "$_SSH_STORE/$_SSH_KEY"
		chmod 600 "$_SSH_STORE/$_SSH_KEY"
	fi
	_dbpid=$(pidof dropbear 2>/dev/null | tr ' ' '\n' | sort -n | head -1)
	[ -n "$_dbpid" ] && kill "$_dbpid" 2>/dev/null || true
	sleep 1
	dropbear -R
fi

# Boot splash: write raw RGB565 image directly to framebuffer if enabled.
_piclock_ini_get() {
	[ -f /boot/piclock/piclock.ini ] || return 1
	val=$(awk -F= "/^$1/ { gsub(/[[:space:]]/, \"\", \$2); print \$2 }" /boot/piclock/piclock.ini)
	echo "${val}"
}
if [ "$(_piclock_ini_get splash_enabled)" = "true" ] && [ -f /boot/bootsplash.raw ]; then
	echo 0 > /sys/class/vtconsole/vtcon1/bind 2>/dev/null || true
	dd if=/boot/bootsplash.raw of=/dev/fb0 bs=4096 2>/dev/null || true
fi

# Update cmdline.txt for the next boot based on splash_enabled.
# splash=true  -> quiet + loglevel=0: kernel produces no text output.
# splash=false -> normal cmdline: all kernel and console text visible.
if [ "$(_piclock_ini_get splash_enabled)" = "true" ]; then
	echo 'quiet loglevel=0 logo.nologo consoleblank=0' > /boot/cmdline.txt
else
	echo 'logo.nologo consoleblank=0 console=tty1' > /boot/cmdline.txt
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
	_raw_mask=$(_ini_get network netmask)
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
	# BusyBox ifup does not derive the broadcast address from address/netmask.
	# Set it explicitly: delete then re-add with broadcast + so the kernel
	# derives the correct subnet broadcast (e.g. 192.168.8.255 for /24).
	# 'ip addr change ... broadcast +' silently does nothing on this build.
	if [ "$_net_mode" = "static" ]; then
		ip addr del "${_net_addr}/${_raw_mask}" dev eth0 2>/dev/null || true
		ip addr add "${_net_addr}/${_raw_mask}" broadcast + dev eth0 2>/dev/null || true
		# Add a host route for the limited broadcast address (255.255.255.255)
		# so services like alsa-ltc can send to it without a default gateway.
		ip route replace 255.255.255.255/32 dev eth0 2>/dev/null || true
	fi
	# In dual mode, add the static alias directly via ip rather than ifup eth0:1.
	# ifup eth0 spawns udhcpc in the background and returns before eth0 is marked
	# "up" in the ifupdown state file, causing ifup eth0:1 to fail silently at boot.
	if [ "$_net_mode" = "dual" ]; then
		ip addr add "${_net_addr}/${_raw_mask}" dev eth0 2>/dev/null || true
	fi

	# NTP control
	_ntp=$(_ini_get network ntp)
	if [ "$_ntp" = "false" ]; then
		/etc/init.d/S49ntp stop 2>/dev/null || true
	elif [ "$_ntp" = "true" ]; then
		/etc/init.d/S49ntp restart 2>/dev/null || true
	fi

	# Wi-Fi AP mode
	_wifi_ap=$(_ini_get wifi ap_enabled)
	if [ "$_wifi_ap" = "true" ]; then
		_wifi_ssid=$(_ini_get wifi ap_ssid)
		_wifi_pass=$(_ini_get wifi ap_password)
		_wifi_chan=$(_ini_get wifi ap_channel)
		[ -z "$_wifi_ssid" ]    && _wifi_ssid="piClock-ap"
		[ -z "$_wifi_chan" ]    && _wifi_chan="6"

		# Ensure wireless module is loaded
		modprobe brcmfmac 2>/dev/null || true
		sleep 1

		# Stop any client-mode wpa_supplicant / udhcpc started by S40wifi
		kill "$(cat /var/run/udhcpc.wlan0.pid 2>/dev/null)" 2>/dev/null || true
		killall wpa_supplicant 2>/dev/null || true
		sleep 1

		# Write hostapd config
		cat > /tmp/hostapd.conf <<HAEOF
interface=wlan0
driver=nl80211
ssid=$_wifi_ssid
hw_mode=g
channel=$_wifi_chan
auth_algs=1
wpa=2
wpa_passphrase=$_wifi_pass
wpa_key_mgmt=WPA-PSK
rsn_pairwise=CCMP
HAEOF

		ifconfig wlan0 up 2>/dev/null || true
		/boot/hostapd -B /tmp/hostapd.conf 2>/dev/null || true
		sleep 2

		# Assign static IP to AP interface
		ifconfig wlan0 192.168.50.1 netmask 255.255.255.0 2>/dev/null || true

		# Start dnsmasq for DHCP on AP subnet
		if [ -x /boot/dnsmasq ]; then
			/boot/dnsmasq \
				--interface=wlan0 \
				--bind-interfaces \
				--dhcp-range=192.168.50.10,192.168.50.100,12h \
				--no-resolv \
				--pid-file=/var/run/dnsmasq-ap.pid \
				2>/dev/null || true
		fi
	fi
fi
