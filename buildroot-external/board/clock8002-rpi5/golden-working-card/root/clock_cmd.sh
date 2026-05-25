#!/bin/sh
# Populate SDL_EVDEV_DEVICES so SDL3 (built with -DSDL_LIBUDEV=OFF) discovers
# keyboard/input devices.  Without libudev, SDL_EVDEV_Init() has a no-op
# device scanner; this hint is the only enumeration path available.
#
# Class 258 = 0x0102 = SDL_UDEV_DEVICE_KEYBOARD(0x02) | SDL_UDEV_DEVICE_HAS_KEYS(0x100)
_evdev_list=""
for _ev in /dev/input/event*; do
    [ -e "$_ev" ] || continue
    if [ -z "$_evdev_list" ]; then
        _evdev_list="258:${_ev}"
    else
        _evdev_list="${_evdev_list},258:${_ev}"
    fi
done
[ -n "$_evdev_list" ] && export SDL_EVDEV_DEVICES="$_evdev_list"
unset _evdev_list _ev

exec /root/sdl-clock -C /boot/piclock/clock.ini

