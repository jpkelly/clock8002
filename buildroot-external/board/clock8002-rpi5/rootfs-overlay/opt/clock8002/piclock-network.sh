#!/bin/sh
# Apply network settings from /boot/piclock/network.ini
# Runs at boot via S45piclock-network

NETWORK_INI="/boot/piclock/network.ini"
RUN_DIR="/run/piclock"
AP_PIDFILE="${RUN_DIR}/wpa_ap.pid"
AP_CONF="${RUN_DIR}/wpa_ap.conf"
DNSMASQ_PIDFILE="${RUN_DIR}/dnsmasq-ap.pid"
AP_ADDR="192.168.50.1"
AP_MASK="255.255.255.0"
AP_DHCP_START="192.168.50.10"
AP_DHCP_END="192.168.50.150"

[ -f "$NETWORK_INI" ] || exit 0

# Simple INI parser — reads key=value, ignoring comments and section headers
parse_ini() {
    local file="$1" section="$2" key="$3"
    awk -F= -v section="$section" -v key="$key" '
        /^\[/ { current = substr($0, 2, index($0, "]") - 2) }
        current == section && $1 == key { print $2; exit }
    ' "$file"
}

NET_MODE=$(parse_ini "$NETWORK_INI" network mode)
NET_HOSTNAME=$(parse_ini "$NETWORK_INI" host hostname)

# Apply NTP setting
NTP_ENABLED=$(parse_ini "$NETWORK_INI" network ntp)
if [ "$NTP_ENABLED" = "true" ]; then
    echo "Running one-shot NTP sync via BusyBox ntpd..."
    ntpd -g -q 2>/dev/null || echo "NTP sync failed (not critical)"
elif [ "$NTP_ENABLED" = "false" ]; then
    echo "NTP disabled (by network.ini)"
fi

mkdir -p "$RUN_DIR"

# --- Wi-Fi Access Point (non-NM backend) ---
stop_ap() {
    if [ -f "$DNSMASQ_PIDFILE" ]; then
        kill "$(cat "$DNSMASQ_PIDFILE")" 2>/dev/null || true
        rm -f "$DNSMASQ_PIDFILE"
    fi
    if [ -f "$AP_PIDFILE" ]; then
        kill "$(cat "$AP_PIDFILE")" 2>/dev/null || true
        rm -f "$AP_PIDFILE"
    fi
    ifconfig wlan0 0.0.0.0 2>/dev/null || true
}

channel_to_freq() {
    ch="$1"
    case "$ch" in
        1) echo 2412 ;;
        2) echo 2417 ;;
        3) echo 2422 ;;
        4) echo 2427 ;;
        5) echo 2432 ;;
        6) echo 2437 ;;
        7) echo 2442 ;;
        8) echo 2447 ;;
        9) echo 2452 ;;
        10) echo 2457 ;;
        11) echo 2462 ;;
        *) echo 2437 ;;
    esac
}

# Wait for wlan0 to appear (brcmfmac loads via eudev uevent after boot)
i=0
while [ "$i" -lt 30 ]; do
    ip link show wlan0 >/dev/null 2>&1 && break
    i=$((i + 1))
    sleep 1
done
if ip link show wlan0 >/dev/null 2>&1; then
    echo "wlan0 ready after ${i}s"
else
    echo "Warning: wlan0 not found after 30s"
fi

# Set regulatory domain so AP mode is permitted.
AP_COUNTRY=$(parse_ini "$NETWORK_INI" wifi ap_country)
[ -z "$AP_COUNTRY" ] && AP_COUNTRY="US"
echo "Setting WiFi regulatory domain: ${AP_COUNTRY}"
command -v iw >/dev/null 2>&1 && iw reg set "$AP_COUNTRY" || true
sleep 1

AP_ENABLED=$(parse_ini "$NETWORK_INI" wifi ap_enabled)

if [ "$AP_ENABLED" = "true" ]; then
    AP_SSID=$(parse_ini "$NETWORK_INI" wifi ap_ssid)
    AP_PASSWORD=$(parse_ini "$NETWORK_INI" wifi ap_password)
    AP_CHANNEL=$(parse_ini "$NETWORK_INI" wifi ap_channel)

    [ -z "$AP_SSID" ] && AP_SSID="$(hostname)-ap"
    [ -z "$AP_PASSWORD" ] && AP_PASSWORD="clockwork"
    [ -z "$AP_CHANNEL" ] && AP_CHANNEL="6"
    AP_FREQ=$(channel_to_freq "$AP_CHANNEL")

    echo "Configuring Wi-Fi AP: ${AP_SSID} (ch ${AP_CHANNEL})"
    stop_ap

    cat > "$AP_CONF" <<EOF
ctrl_interface=/run/wpa_supplicant
ap_scan=2
country=${AP_COUNTRY}
network={
    ssid="${AP_SSID}"
    mode=2
    frequency=${AP_FREQ}
    key_mgmt=WPA-PSK
    psk="${AP_PASSWORD}"
    proto=RSN
    pairwise=CCMP
}
EOF

    wpa_supplicant -B -i wlan0 -D nl80211 -c "$AP_CONF" -P "$AP_PIDFILE" >/tmp/piclock-wifi-ap.log 2>&1 || true
    ifconfig wlan0 "$AP_ADDR" netmask "$AP_MASK" up 2>/dev/null || true

    if command -v dnsmasq >/dev/null 2>&1; then
        dnsmasq \
            --interface=wlan0 \
            --bind-interfaces \
            --except-interface=lo \
            --dhcp-range="${AP_DHCP_START},${AP_DHCP_END},12h" \
            --dhcp-option=3,"${AP_ADDR}" \
            --pid-file="$DNSMASQ_PIDFILE" \
            --conf-file= >/tmp/piclock-dnsmasq-ap.log 2>&1 || true
    else
        echo "Warning: dnsmasq not installed; AP clients need manual IP configuration"
    fi
elif [ "$AP_ENABLED" = "false" ]; then
    echo "Disabling Wi-Fi AP"
    stop_ap
fi

# --- Hostname ---
if [ -n "$NET_HOSTNAME" ]; then
    CURRENT_HOSTNAME=$(hostname)
    if [ "$CURRENT_HOSTNAME" != "$NET_HOSTNAME" ]; then
        echo "Setting hostname to ${NET_HOSTNAME}..."
        hostname "$NET_HOSTNAME"
        # /etc/hostname is on squashfs (read-only) — skip write silently
        if [ -w /etc/hostname ]; then
            echo "$NET_HOSTNAME" > /etc/hostname
        fi
        if [ -w /etc/hosts ]; then
            sed -i "s/127\.0\.1\.1.*/127.0.1.1\t${NET_HOSTNAME}/" /etc/hosts
            grep -q "127.0.1.1" /etc/hosts || printf "127.0.1.1\t%s\n" "$NET_HOSTNAME" >> /etc/hosts
        fi
        [ -x /etc/init.d/S50avahi-daemon ] && /etc/init.d/S50avahi-daemon restart 2>/dev/null || true
    fi
fi
