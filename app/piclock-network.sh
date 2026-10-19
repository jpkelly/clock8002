#!/bin/sh
# Apply network settings from the boot partition network.ini
# Runs at boot via piclock-network.service (Trixie)
#
# Pi OS Trixie: /boot/firmware is the FAT32 boot partition.
NETWORK_INI="/boot/firmware/piclock/network.ini"

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
if [ "$NTP_ENABLED" = "false" ]; then
    echo "Disabling NTP..."
    if command -v timedatectl >/dev/null 2>&1; then
        timedatectl set-ntp false
    fi
elif [ "$NTP_ENABLED" = "true" ]; then
    echo "Enabling NTP..."
    if command -v timedatectl >/dev/null 2>&1; then
        timedatectl set-ntp true
    fi
fi

# Find the wired Ethernet interface.
# Pi 5 Trixie uses end0 or eth0 depending on configuration.
find_wired_iface() {
    for iface in end0 eth0 enp0s31f6; do
        if [ -e "/sys/class/net/${iface}" ]; then
            echo "$iface"
            return 0
        fi
    done
    return 1
}

# Print the UUID of the NM ethernet connection bound to interface $1, falling
# back to one not bound to any interface.
# This matches on the interface the connection is bound to, not the device it
# is active on: a wired connection only activates once the cable has a link,
# so with no cable plugged in an active-device lookup never matches.
find_wired_con() {
    local uuid bound fallback=""
    for uuid in $(nmcli -g UUID,TYPE con show | grep ':802-3-ethernet$' | cut -d: -f1); do
        bound=$(nmcli -g connection.interface-name con show uuid "$uuid")
        if [ "$bound" = "$1" ]; then
            echo "$uuid"
            return 0
        fi
        [ -z "$bound" ] && [ -z "$fallback" ] && fallback="$uuid"
    done
    [ -n "$fallback" ] && echo "$fallback"
}

# Bring the wired connection up so modified settings take effect now. Without
# a link (no cable, or still negotiating) there is nothing to bring up, and NM
# applies the saved settings by itself when the link comes up, so don't wait
# for it.
activate_wired() {
    if [ "$(cat "/sys/class/net/${WIRED_IFACE}/carrier" 2>/dev/null)" = "1" ]; then
        nmcli --wait 10 con up "$NM_CON"
    else
        echo "No link on ${WIRED_IFACE} yet; NetworkManager will apply the settings when it comes up."
    fi
}

WIRED_IFACE=""
for i in $(seq 1 10); do
    WIRED_IFACE=$(find_wired_iface) && break
    sleep 1
done

NM_CON=""
if [ -n "$WIRED_IFACE" ]; then
    NM_CON=$(find_wired_con "$WIRED_IFACE")
    # With no saved wired connection and no automatic default from NM yet,
    # create one so the settings below have somewhere to go.
    if [ -z "$NM_CON" ] && [ -n "$NET_MODE" ]; then
        echo "Creating wired connection for ${WIRED_IFACE}..."
        nmcli con add type ethernet ifname "$WIRED_IFACE" con-name piclock-wired autoconnect yes
        NM_CON=$(find_wired_con "$WIRED_IFACE")
    fi
fi

if [ -n "$NM_CON" ]; then
    echo "Using wired connection: $(nmcli -g connection.id con show uuid "$NM_CON") on ${WIRED_IFACE}"
fi

if [ -z "$NM_CON" ]; then
    echo "No wired interface or NetworkManager connection found; skipping wired settings."
elif [ "$NET_MODE" = "static" ]; then
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
        activate_wired
    else
        echo "Warning: static mode set but address/netmask missing."
    fi
elif [ "$NET_MODE" = "dhcp" ]; then
    CURRENT_METHOD=$(nmcli -g ipv4.method con show "$NM_CON")
    if [ "$CURRENT_METHOD" != "auto" ]; then
        echo "Switching to DHCP..."
        nmcli con mod "$NM_CON" ipv4.method auto ipv4.addresses "" ipv4.gateway "" ipv4.dns ""
        activate_wired
    else
        echo "Network mode: DHCP (already set)."
    fi
elif [ "$NET_MODE" = "dual" ]; then
    NET_ADDR=$(parse_ini "$NETWORK_INI" network address)
    NET_MASK=$(parse_ini "$NETWORK_INI" network netmask)
    NET_GW=$(parse_ini "$NETWORK_INI" network gateway)
    NET_DNS=$(parse_ini "$NETWORK_INI" network dns)

    if [ -n "$NET_ADDR" ] && [ -n "$NET_MASK" ]; then
        echo "Applying dual mode: DHCP + static alias ${NET_ADDR}/${NET_MASK}..."
        # With ipv4.method=auto, adding a static address keeps the DHCP lease too.
        nmcli con mod "$NM_CON" ipv4.method auto \
            ipv4.addresses "${NET_ADDR}/${NET_MASK}"
        # In dual mode DHCP supplies the default route; leave gateway/dns empty
        # unless the user explicitly provided them.
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
        activate_wired
    else
        echo "Warning: dual mode set but address/netmask missing."
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
    _ap_retries=0
    while [ "$_ap_retries" -lt 3 ]; do
        nmcli --wait 15 con up "$AP_CON" && break
        _ap_retries=$((_ap_retries + 1))
        echo "AP activation attempt ${_ap_retries} failed, retrying in 5s..."
        sleep 5
    done
    [ "$_ap_retries" -ge 3 ] && echo "Warning: AP activation failed after 3 attempts (will autoconnect later)"
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
        if command -v hostnamectl >/dev/null 2>&1; then
            hostnamectl set-hostname "$NET_HOSTNAME"
        else
            hostname "$NET_HOSTNAME"
        fi
        # Update /etc/hosts so sudo and other tools can resolve the hostname
        sed -i "s/127\.0\.1\.1.*/127.0.1.1\t${NET_HOSTNAME}/" /etc/hosts
        grep -q "127.0.1.1" /etc/hosts || printf "127.0.1.1\t%s\n" "$NET_HOSTNAME" >> /etc/hosts
        # Restart mDNS so the new hostname is advertised on the network
        if command -v systemctl >/dev/null 2>&1; then
            systemctl restart avahi-daemon 2>/dev/null || true
        elif [ -x /etc/init.d/S50avahi-daemon ]; then
            /etc/init.d/S50avahi-daemon restart 2>/dev/null || true
        fi
    fi
fi
