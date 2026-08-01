#!/bin/sh
# Read-only system fingerprint for comparing two piClock SD card images.
sec() { echo; echo "########## $1 ##########"; }

sec IDENTITY
hostname
cat /etc/os-release 2>/dev/null | grep -E '^(PRETTY_NAME|VERSION_ID|VERSION_CODENAME)='
echo "uptime_s=$(cut -d. -f1 /proc/uptime)"

sec KERNEL
uname -a
echo "PAGESIZE=$(getconf PAGESIZE)"
echo "cmdline: $(cat /proc/cmdline)"

sec FIRMWARE_BOOTLOADER
vcgencmd version 2>/dev/null
echo "--- bootloader EEPROM ---"
sudo rpi-eeprom-update 2>/dev/null | head -20

sec CONFIG_TXT
grep -vE '^\s*#|^\s*$' /boot/firmware/config.txt 2>/dev/null
sec CMDLINE_TXT
cat /boot/firmware/cmdline.txt 2>/dev/null

sec BOOT_PARTITION_FILES
ls -la /boot/firmware/*.img /boot/firmware/*.elf /boot/firmware/initramfs* 2>/dev/null | awk '{print $5, $9}'

sec PACKAGES_KERNEL_FW_GFX_AUDIO
dpkg -l 2>/dev/null | awk '/^ii/ && ($2 ~ /linux-image|linux-headers|raspi-firmware|rpi-eeprom|libcamera|mesa|libgl|libdrm|alsa|libasound|libsdl|systemd|udev/) {printf "%-45s %s\n", $2, $3}'

sec SDL_LIBS
ls -la /usr/lib/aarch64-linux-gnu/libSDL* 2>/dev/null | awk '{print $5, $9, $10, $11}'
ldconfig -p 2>/dev/null | grep -i sdl

sec MODULES_LOADED
lsmod | sort

sec MODULE_PARAMS_USB_AUDIO
for m in snd_usb_audio usbcore xhci_hcd xhci_pci snd_pcm; do
  echo "--- $m ---"
  if [ -d /sys/module/$m/parameters ]; then
    for p in /sys/module/$m/parameters/*; do
      [ -r "$p" ] && echo "  $(basename $p)=$(cat $p 2>/dev/null)"
    done
  else
    echo "  (not loaded / no params)"
  fi
done

sec MODPROBE_CONF
grep -rH . /etc/modprobe.d/ 2>/dev/null | grep -vE '^\s*#'

sec XHCI_QUIRKS_AND_PCI
sudo dmesg 2>/dev/null | grep -iE 'xhci|quirks' | head -25
echo "--- lspci ---"
lspci -nn 2>/dev/null
echo "--- link status ---"
sudo lspci -vvv -s 0001:01:00.0 2>/dev/null | grep -E 'LnkCap|LnkSta|MaxPayload|MaxReadReq'

sec USB_TOPOLOGY
lsusb 2>/dev/null
echo "--- tree ---"
lsusb -t 2>/dev/null

sec ALSA
cat /proc/asound/cards 2>/dev/null
echo "--- hw_params ---"
for c in /proc/asound/card*/pcm0c/sub0/hw_params; do echo "[$c]"; cat "$c" 2>/dev/null; done
echo "--- status ---"
for c in /proc/asound/card*/pcm0c/sub0/status; do echo "[$c]"; cat "$c" 2>/dev/null; done

sec CPU_THERMAL
echo "governor: $(cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null)"
echo "arm_freq: $(vcgencmd measure_clock arm 2>/dev/null)"
echo "throttled: $(vcgencmd get_throttled 2>/dev/null)"
echo "temp: $(awk '{printf "%.1fC\n", $1/1000}' /sys/class/thermal/thermal_zone0/temp 2>/dev/null)"

sec SERVICES
systemctl list-units --type=service --state=running --no-pager --no-legend 2>/dev/null | awk '{print $1}'
echo "--- enabled, clock-related ---"
systemctl list-unit-files --no-pager --no-legend 2>/dev/null | grep -iE 'clock|ltc|oled|splash|piclock|wedge'

sec UNIT_FILES_LTC_CLOCK
for u in alsa-ltc clock8002 oled_daemon piclock-network; do
  echo "--- $u ---"
  systemctl cat $u 2>/dev/null | grep -vE '^\s*#|^\s*$'
done

sec INSTALLED_BINARIES
ls -la /opt/clock8002/ 2>/dev/null | grep -viE '\.ttf$|fonts|voices|\.png$'
echo "--- versions ---"
for b in /opt/clock8002/sdl-clock /opt/clock8002/sdl3-clock /opt/clock8002/alsa-ltc; do
  [ -x "$b" ] || continue
  echo "[$b]"
  echo "  sha256: $(sha256sum "$b" | cut -d' ' -f1)"
  strings "$b" 2>/dev/null | grep -m2 -oE 'clock\.git(Tag|Commit)=[0-9a-zA-Z.]+'
done

sec BOOT_TIMING
systemd-analyze 2>/dev/null
systemd-analyze blame 2>/dev/null | head -12

sec DMESG_ERRORS
sudo dmesg --level=err,warn 2>/dev/null | head -30
