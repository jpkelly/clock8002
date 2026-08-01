piClock - Configuration Folder
==============================

These files configure this piClock unit. They live on the SD card's boot
partition, so you can edit them from a Mac or PC with the card removed, or
over SSH at /boot/firmware/piclock/.

Edit with a plain text editor only (TextEdit in plain text mode, Notepad).
Never use Word or Pages.

  network.ini       Hostname, IP address, NTP, Wi-Fi hotspot
  clock.ini         Clock settings - use the web UI instead of editing this
  oled.ini          OLED display: enabled, i2c_port, i2c_address, rotation

Reboot after changing anything here.


network.ini
===========

Format: key=value, no spaces around the "=". Lines starting with # are
ignored. Keep the [section] headers.


[network]
---------

mode=       dhcp    Router assigns the address. Find the clock by name.
            static  Fixed address you choose. Needs address and netmask.
            dual    DHCP plus an extra fixed address on the same port.
                    Use when the venue is DHCP but show control needs a
                    predictable IP.

ntp=        true    Sync time from the internet at boot.
            false   Don't sync. Use this when LTC or OSC drives the time,
                    otherwise NTP will fight your timecode source.

address=    Fixed IPv4 address. Static and dual modes only.
            Pick something outside the router's DHCP pool.

netmask=    CIDR prefix length - a number, not a dotted address:
              24  =  255.255.255.0
              16  =  255.255.0.0

gateway=    Router address, for internet access. Static mode only.
            Comment it out on an isolated show network.

dns=        Name server. Static mode only. DHCP supplies this otherwise.


[host]
------

hostname=   The unit's network name. With hostname=piClock you reach it at
            http://piClock.local and ssh pi@piClock.local.
            Give each unit a different name if you have more than one.
            Letters, digits and hyphens only.


[wifi]
------

The unit can broadcast its own hotspot alongside the wired connection -
handy for configuring it with no network available.

ap_enabled=     true or false. Default false.
ap_ssid=        Network name. Defaults to <hostname>-ap.
ap_password=    Minimum 8 characters. Default clockwork.
ap_channel=     1-11. Use 1, 6 or 11. Default 6.
ap_country=     The radio's regulatory domain is fixed to US when
                the software is installed, and this setting is
                ignored. Planned for a future release.

Connect to the hotspot, then browse to http://piClock.local. The icon in the
OLED's top right corner is lit while the hotspot is running.


Examples
--------

DHCP, time from the internet:

    [network]
    mode=dhcp
    ntp=true

    [host]
    hostname=piClock

    [wifi]
    ap_enabled=false


Show network, fixed address, time from LTC:

    [network]
    mode=static
    ntp=false
    address=192.168.8.245
    netmask=24

    [host]
    hostname=piClock-Main

    [wifi]
    ap_enabled=false


Venue DHCP, but show control needs a known address:

    [network]
    mode=dual
    ntp=true
    address=10.0.0.245
    netmask=24

    [host]
    hostname=piClock-Stage

    [wifi]
    ap_enabled=false


If you can't reach the unit
---------------------------

Put the SD card in your computer, set mode=dhcp in network.ini, comment out
address and netmask, and boot again. The clock also shows its IP on screen
for 30 seconds after startup.

Usual causes: spaces around the "=", netmask written as 255.255.255.0, two
units sharing a hostname or IP, or a static address inside the DHCP pool.


Default access
==============

  Web    http://piClock.local     admin / clockwork

  SSH    ssh pi@piClock.local

         The password is the one you chose in Raspberry Pi Imager when the
         card was written. There is no factory default, and it is not
         stored anywhere on the card.

         The root account is locked and cannot be used with a password.
         Use sudo from the pi account when you need root.

Change the web password before putting a unit on a network you don't control.
