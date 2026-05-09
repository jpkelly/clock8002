#!/bin/sh

alsa_card="$(awk '/USB-Audio - USB Audio Device/ { print $1; exit }' /proc/asound/cards 2>/dev/null)"
exec /root/alsa-ltc "plughw:${alsa_card:-2},0" 255.255.255.255 1245
