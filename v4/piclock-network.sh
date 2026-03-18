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
        [ -n "$NET_GW" ] && nmcli con mod "$NM_CON" ipv4.gateway "$NET_GW"
        [ -n "$NET_DNS" ] && nmcli con mod "$NM_CON" ipv4.dns "$NET_DNS"
        nmcli con up "$NM_CON"
    else
        echo "Warning: static mode set but address/netmask missing."
    fi
elif [ "$NET_MODE" = "dhcp" ]; then
    echo "Network mode: DHCP (no changes needed)."
fi
