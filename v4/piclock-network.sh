#!/bin/bash
# Apply network settings from /boot/piclock/network.ini
# Runs at boot via piclock-network.service

NETWORK_INI="/boot/piclock/network.ini"

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

# Apply hostname if set
if [ -n "$NET_HOSTNAME" ]; then
    CURRENT_HOSTNAME=$(hostname)
    if [ "$CURRENT_HOSTNAME" != "$NET_HOSTNAME" ]; then
        echo "Setting hostname to ${NET_HOSTNAME}..."
        hostnamectl set-hostname "$NET_HOSTNAME"
        # Update /etc/hosts so sudo and other tools can resolve the hostname
        sed -i "s/127\.0\.1\.1.*/127.0.1.1\t${NET_HOSTNAME}/" /etc/hosts
        grep -q "127.0.1.1" /etc/hosts || echo -e "127.0.1.1\t${NET_HOSTNAME}" >> /etc/hosts
        # Restart mDNS so the new hostname is advertised on the network
        systemctl restart avahi-daemon 2>/dev/null || true
    fi
fi

# Wait for NetworkManager to be ready
sleep 5

NM_CON=$(nmcli -t -f NAME,DEVICE con show --active | head -1 | cut -d: -f1)
[ -z "$NM_CON" ] && { echo "No active NetworkManager connection found."; exit 0; }

if [ "$NET_MODE" = "static" ]; then
    NET_ADDR=$(parse_ini "$NETWORK_INI" network address)
    NET_MASK=$(parse_ini "$NETWORK_INI" network netmask)
    NET_GW=$(parse_ini "$NETWORK_INI" network gateway)
    NET_DNS=$(parse_ini "$NETWORK_INI" network dns)

    if [ -n "$NET_ADDR" ] && [ -n "$NET_MASK" ]; then
        echo "Applying static IP: ${NET_ADDR}/${NET_MASK}..."
        nmcli con mod "$NM_CON" ipv4.method manual \
            ipv4.addresses "${NET_ADDR}/${NET_MASK}"
        if [ -n "$NET_GW" ]; then
            nmcli con mod "$NM_CON" ipv4.gateway "$NET_GW"
        else
            nmcli con mod "$NM_CON" ipv4.gateway ""
        fi
        if [ -n "$NET_DNS" ]; then
            nmcli con mod "$NM_CON" ipv4.dns "$NET_DNS"
        else
            nmcli con mod "$NM_CON" ipv4.dns ""
        fi
        nmcli con up "$NM_CON"
    else
        echo "Warning: static mode set but address/netmask missing."
    fi
elif [ "$NET_MODE" = "dhcp" ]; then
    CURRENT_METHOD=$(nmcli -g ipv4.method con show "$NM_CON")
    if [ "$CURRENT_METHOD" != "auto" ]; then
        echo "Switching to DHCP..."
        nmcli con mod "$NM_CON" ipv4.method auto ipv4.addresses "" ipv4.gateway "" ipv4.dns ""
        nmcli con up "$NM_CON"
    else
        echo "Network mode: DHCP (already set)."
    fi
fi

# --- Wi-Fi Access Point ---
AP_ENABLED=$(parse_ini "$NETWORK_INI" wifi ap_enabled)
AP_CON="piclock-ap"

if [ "$AP_ENABLED" = "true" ]; then
    AP_SSID=$(parse_ini "$NETWORK_INI" wifi ap_ssid)
    AP_PASSWORD=$(parse_ini "$NETWORK_INI" wifi ap_password)
    AP_CHANNEL=$(parse_ini "$NETWORK_INI" wifi ap_channel)

    # Default SSID to <hostname>-ap
    [ -z "$AP_SSID" ] && AP_SSID="$(hostname)-ap"
    # Default password
    [ -z "$AP_PASSWORD" ] && AP_PASSWORD="clockwork"
    # Default channel
    [ -z "$AP_CHANNEL" ] && AP_CHANNEL="6"

    if nmcli con show "$AP_CON" >/dev/null 2>&1; then
        echo "Updating Wi-Fi AP: ${AP_SSID}..."
        nmcli con mod "$AP_CON" \
            802-11-wireless.ssid "$AP_SSID" \
            802-11-wireless.channel "$AP_CHANNEL" \
            wifi-sec.psk "$AP_PASSWORD"
    else
        echo "Creating Wi-Fi AP: ${AP_SSID}..."
        nmcli con add type wifi ifname wlan0 con-name "$AP_CON" autoconnect yes \
            ssid "$AP_SSID" \
            802-11-wireless.mode ap \
            802-11-wireless.band bg \
            802-11-wireless.channel "$AP_CHANNEL" \
            ipv4.method shared \
            wifi-sec.key-mgmt wpa-psk \
            wifi-sec.psk "$AP_PASSWORD"
    fi
    nmcli con up "$AP_CON"
elif [ "$AP_ENABLED" = "false" ]; then
    if nmcli con show "$AP_CON" >/dev/null 2>&1; then
        echo "Disabling Wi-Fi AP..."
        nmcli con down "$AP_CON" 2>/dev/null || true
        nmcli con delete "$AP_CON"
    fi
fi
