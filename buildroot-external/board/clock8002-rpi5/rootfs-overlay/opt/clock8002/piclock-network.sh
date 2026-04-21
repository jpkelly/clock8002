#!/bin/sh
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

# Apply NTP setting
# Note: timedatectl is a systemd tool; guard for BusyBox init images.
NTP_ENABLED=$(parse_ini "$NETWORK_INI" network ntp)
if [ "$NTP_ENABLED" = "false" ]; then
    echo "Disabling NTP..."
    command -v timedatectl >/dev/null 2>&1 && timedatectl set-ntp false || true
elif [ "$NTP_ENABLED" = "true" ]; then
    echo "Enabling NTP..."
    command -v timedatectl >/dev/null 2>&1 && timedatectl set-ntp true || true
fi

# --- Wi-Fi Access Point ---
# Note: wired IP config is handled pre-NM by S43piclock-network-prep.

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
    echo "Warning: wlan0 not found after 30s — AP may not start"
fi

# Set regulatory domain so AP mode is permitted.
AP_COUNTRY=$(parse_ini "$NETWORK_INI" wifi ap_country)
[ -z "$AP_COUNTRY" ] && AP_COUNTRY="US"
echo "Setting WiFi regulatory domain: ${AP_COUNTRY}"
iw reg set "$AP_COUNTRY"
sleep 1

AP_ENABLED=$(parse_ini "$NETWORK_INI" wifi ap_enabled)
AP_CON="piclock-ap"

if [ "$AP_ENABLED" = "true" ]; then
    AP_SSID=$(parse_ini "$NETWORK_INI" wifi ap_ssid)
    AP_PASSWORD=$(parse_ini "$NETWORK_INI" wifi ap_password)
    AP_CHANNEL=$(parse_ini "$NETWORK_INI" wifi ap_channel)

    [ -z "$AP_SSID" ] && AP_SSID="$(hostname)-ap"
    [ -z "$AP_PASSWORD" ] && AP_PASSWORD="clockwork"
    [ -z "$AP_CHANNEL" ] && AP_CHANNEL="6"

    if nmcli con show "$AP_CON" >/dev/null 2>&1; then
        echo "Updating Wi-Fi AP: ${AP_SSID}..."
        nmcli con mod "$AP_CON" \
            802-11-wireless.ssid "$AP_SSID" \
            802-11-wireless.channel "$AP_CHANNEL" \
            wifi-sec.psk "$AP_PASSWORD" \
            ipv4.method shared
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
    nmcli --wait 10 con up "$AP_CON" || echo "Warning: AP activation timed out (will autoconnect later)"
elif [ "$AP_ENABLED" = "false" ]; then
    if nmcli con show "$AP_CON" >/dev/null 2>&1; then
        echo "Disabling Wi-Fi AP..."
        nmcli con down "$AP_CON" 2>/dev/null || true
        nmcli con delete "$AP_CON"
    fi
fi

# --- Hostname ---
# Applied last so NetworkManager connection activations above cannot override it.
if [ -n "$NET_HOSTNAME" ]; then
    CURRENT_HOSTNAME=$(hostname)
    if [ "$CURRENT_HOSTNAME" != "$NET_HOSTNAME" ]; then
        echo "Setting hostname to ${NET_HOSTNAME}..."
        hostname "$NET_HOSTNAME"
        echo "$NET_HOSTNAME" > /etc/hostname
        sed -i "s/127\.0\.1\.1.*/127.0.1.1\t${NET_HOSTNAME}/" /etc/hosts
        grep -q "127.0.1.1" /etc/hosts || echo -e "127.0.1.1\t${NET_HOSTNAME}" >> /etc/hosts
        systemctl restart avahi-daemon 2>/dev/null || true
    fi
fi
