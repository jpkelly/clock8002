# piClock Quick Start

You have a piClock. The SD card is already in the unit. Plug it in and go.

## Power up

1. Plug a display into either HDMI port (the ports are not labeled).
2. Plug Ethernet in if you have a network, or skip it and use the Wi-Fi hotspot.
3. Power the unit.

The clock starts by itself. For about 30 seconds the HDMI screen shows the version and IP address.

The small OLED on the unit shows:

- IP address
- `piclock.local`
- Web user and password
- A Wi-Fi icon in the top-right corner while the hotspot is on

## Get in

Pick whichever is easiest:

| How | Address |
|-----|---------|
| Browser on the same network | http://piclock.local |
| Browser by IP | http://192.168.8.245 |
| SSH | `ssh pi@piclock.local` |

Web login: **admin** / **clockwork**

SSH login: **pi** / **clockworkadmin**

Change the web password before you put the unit on a network you do not control.

### Wi-Fi hotspot

Out of the box the unit broadcasts its own hotspot. Join it from a laptop or phone, then open http://piclock.local.

- Network name: **piClock-ap**
- Password: **clockwork**

The OLED Wi-Fi icon is lit while the hotspot is running.

This is a hotspot only. The clock does not join venue Wi-Fi as a client.

## Ports on the chassis

| Label | What it is |
|-------|------------|
| HDMI x 2 (unlabeled) | Main clock on the first display you plug in. Second display mirrors the clock, or shows full-screen PerfectCue icons if you turn that on in the web UI. |
| PerfectCue | RS-485 for a DSAN Perfect Cue. |
| Limitimer | RS-485 for Limitimer. |
| LTC | USB ADC for LTC timecode. |
| -5V+ | Auxiliary 5 V power (green phoenix connector). |

## What's new compared with clock-8001

- 4 line, and 1-line text clocks
- Second HDMI: PerfectCue full-screen icons, or a mirror of the main clock
- PerfectCue overlay position and size in the web UI
- Font picker in the web UI
- OLED status screen
- Network, hostname, NTP, and Wi-Fi hotspot set from a file on the SD card
- Drop an SSH public key on the SD card for passwordless login

Clock faces, colors, sources, timers, OSC, PerfectCue, and Limitimer are all set in the web UI. You do not need to edit `clock.ini` by hand.

## Configure from the SD card

Power the unit off and put the SD card in a Mac or PC. The boot partition appears as a drive named **piClock**. Open the `piclock` folder.

Edit these files with a plain text editor only (TextEdit in plain text mode, Notepad). Never use Word or Pages. Put the card back and power on for the changes to take effect.

| File | What it does |
|------|----------------|
| `network.ini` | Hostname, IP, NTP, Wi-Fi hotspot |
| `authorized_keys` | SSH public keys (create this file if it is missing) |
| `oled.ini` | OLED on/off and rotation |
| `clock.ini` | Clock settings — use the web UI instead |

There is also a `README` in that folder with the same network details.

### network.ini

Format: `key=value` with no spaces around the `=`. Lines starting with `#` are ignored.

```ini
[network]
mode=static
ntp=false
address=192.168.8.245
netmask=24
#gateway=192.168.8.1
#dns=1.1.1.1

[host]
hostname=piClock

[wifi]
ap_enabled=true
ap_ssid=piClock-ap
ap_password=clockwork
ap_channel=6
```

**mode**

- `dhcp` — the router assigns the address. Find the clock at http://piclock.local.
- `static` — the address you set. Needs `address` and `netmask`.
- `dual` — DHCP plus an extra fixed address. Use when the venue is DHCP but show control needs a known IP. The OLED IP line alternates between the two addresses every 5 seconds.

**ntp** — `true` to sync time from the internet at boot. Leave `false` when LTC or OSC drives the time, otherwise NTP will fight your timecode source.

**netmask** — a number, not a dotted address. `24` means `255.255.255.0`.

**hostname** — the unit's network name. With `hostname=piClock` you reach it at http://piclock.local and `ssh pi@piclock.local`. Letters, digits, and hyphens only. Give each unit a different name if you have more than one.

**Wi-Fi hotspot** — `ap_enabled=true` or `false`. Password must be at least 8 characters.

Factory defaults: hostname `piClock`, static `192.168.8.245`, NTP off, hotspot on.

### Passwordless SSH

1. On your computer, copy your public key. On a Mac or Linux machine that is usually:

   ```bash
   cat ~/.ssh/id_ed25519.pub
   ```

   If that file does not exist, try `~/.ssh/id_rsa.pub`.

2. On the SD card, create or open `piclock/authorized_keys`.
3. Paste the key on its own line. You can add more than one key, one per line.
4. Eject the card, boot the piClock, then:

   ```bash
   ssh pi@piclock.local
   ```

Keys on the card are **added** to the `pi` account on every boot. They are never removed that way.

- Deleting `authorized_keys` from the SD card does **not** turn off passwordless login. Keys already on the piClock stay.
- To revoke a key, SSH in and edit `~/.ssh/authorized_keys` on the unit, or re-image the card.
- Password login stays on either way (`pi` / `clockworkadmin`).

## If you cannot reach it

1. Read the IP off the OLED, or off the HDMI overlay in the first 30 seconds.
2. Try the hotspot: join **piClock-ap**, password **clockwork**, then http://piclock.local.
3. If Ethernet still fails, power off, put the SD card in a computer, set `mode=dhcp` in `network.ini`, comment out `address` and `netmask`, and boot again.

Usual causes: spaces around the `=` in `network.ini`, netmask written as `255.255.255.0`, two units sharing a hostname or IP, or a static address inside the router's DHCP pool.
