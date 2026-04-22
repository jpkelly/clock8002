# Clock8002 Handoff

Last updated: 2026-04-22

## Active Investigation: LTC dropouts on piclockBR (2026-04-21 → 2026-04-22)

**Symptom:** alsa-ltc logs `[gap] no LTC decoded for Nms, peak_during_gap=P` events
on piclockBR (192.168.8.246, Buildroot image c4847b2). All observed gaps show
`peak≈32750-32767` (full-scale) and duration 2-5s — classified INVALID_LTC (audio
present but biphase unreadable). Source is TouchDesigner → USB-C analog →
C-Media CM108 USB audio (`plughw:0,0`).

**Pi-side evidence (kernel dmesg, when in degraded state):**
```
usb 1-1.1: 2:1: cannot set freq 44100 to ep 0x82
usb 1-1.1: 2:0: usb_set_interface failed (-110)
```
Errors arrive on a fixed **17.92s cadence** — kernel periodically retries sample
rate renegotiation with the CM108 and times out.

**Key test result:** Stopping alsa-ltc for 60s → 0 new errors. Restarting
alsa-ltc → 0 new errors for 60s after. The 18s cadence only runs once the card
is in a degraded state; a clean `alsa-ltc` restart resets the endpoint and
clears it. Once in the bad state, each failed renegotiation can stall audio
capture 1-5s → INVALID_LTC gap.

**Escalation event (21:20 UTC):** On the pre-enhanced run, alsa-ltc hit 10 EIO
errors and exited. Watchdog restart attempts failed with `cannot set parameters
(Connection timed out)`. USB stack fully wedged: `can't set config #1, error
-110`; unbind/rebind of `1-1.1` and parent hub `1-1` failed; `lsusb` hung.
Required warm reboot to recover (matches the documented "ribbon cable" failure
mode in `/memories/repo/clock8002-stability-tests.md`).

**Reboot recovery (21:53 UTC):** Card re-enumerated cleanly as card 0, alsa-ltc
started by S99 init. Monitor re-armed.

**2026-04-22 reflash — new SD card with image `piclockBR-8234252-gerry-sdcard.img`:**
- Fresh card booted; hostname bug diagnosed (`/opt/clock8002/piclock-network.sh`
  missing exec bit in overlay → `Permission denied` at boot → network.ini
  silently skipped). Fix committed as **`58c6d17`** (`git update-index --chmod=+x`)
  and pushed. Hot-patched live unit first; Buildroot image rebuilt for 58c6d17.
- On **cold boot** of 8234252: USB xHCI HC died at 147s (hard failure) on that
  run. Subsequent cold boot is clean; unit stable at 192.168.8.246 as
  `piClockBR` (user renamed hostname in network.ini to avoid mDNS collision
  with another `piclock.local` on LAN).
- USB audio now on **card 1** (HDMI takes card 0/2 on this image); `1-1.1`
  still. alsa-ltc PCM RUNNING, hw_ptr advancing, no errors.

**Monitoring in place (ltcmon v3, 2026-04-22 ~15:59 UTC on new image):**
- Script: `/root/ltcmon.sh` on piclockBR (repo copy: `tools/ltcmon.sh`). Rebuilt
  from the session-memory spec — original file did not survive the new SD card.
- Log: `/tmp/ltcmon.log`. Auto-detects USB audio card + usbdev path, so it
  works regardless of card index.
- Watches:
  - `tail -F /tmp/alsa-ltc.log` (foreground) — classifies `[gap]` events
    (SILENCE / INVALID_LTC / PARTIAL), `[APP_ERR]`. **NOTE:** on this image the
    S99 init does not redirect alsa-ltc stdout, so `/tmp/alsa-ltc.log` is empty
    until alsa-ltc is restarted under a redirect. Gap classification dormant
    until then; all other signals still captured.
  - `dmesg` cursor-based poll @5s — KERN/FREQ_FAIL, IFACE_FAIL, XRUN,
    DISCONNECT, RESET, HC_DIED, DMA_PAUSE
  - `/proc/asound/card<N>/pcm0c/sub0/status` @1s — PCM state transitions
    (PCM/STATE), hw_ptr stall detection (PCM/STALL)
  - `/sys/bus/usb/devices/1-1.1/urbnum` @1s — URB stall detection (URB/STALL)
    when alsa-ltc is running
  - `/sys/bus/usb/devices/1-1.1/power/runtime_status` @1s —
    USB/SUSPEND / USB/RESUME transitions (definitive autosuspend fingerprint)
  - 10s HEALTH snapshot: pid, state, pcm, hw_ptr, urbn, rs, susp_dms (ms
    suspended this interval), xhci_d (xHCI interrupt delta), load, temp,
    cumulative freq/iface error counts

**Baseline (healthy):** hw_ptr +442K/10s, urbn +10K/10s, xhci_d ≈10022,
rs=active, susp_dms=0, no errors.

**Autosuspend status:** `autosuspend=2s` is configured on the CM108 endpoint,
but `runtime_suspended_time=0` since boot — device has never suspended while
alsa-ltc is actively reading. Autosuspend could only fire during an
error-recovery window when alsa-ltc momentarily pauses URB submission. The
new USB/SUSPEND tag will definitively show whether autosuspend is the
trigger when the bug next occurs.

**Commands:**
```bash
# Skip heartbeats, see events
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa root@192.168.8.246 \
  'grep -v heartbeat /tmp/ltcmon.log | tail -40'

# Restart monitor (hard-kills strays, clears log)
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa root@192.168.8.246 '
for p in $(ps | grep ltcmon | grep -v grep | awk "{print \$1}"); do kill -9 $p; done
rm -f /tmp/ltcmon.log
nohup /root/ltcmon.sh > /tmp/ltcmon.log 2>&1 &'
```

**Open questions / next steps:**
- Does autosuspend trigger the degraded state? (Monitor v3 will answer: if
  `[USB/SUSPEND]` precedes `[KERN/FREQ_FAIL]` cluster → confirmed; if
  `rs=active` throughout → ruled out)
- Is PCM stall or URB stall the first symptom? (differentiates ALSA buffer
  starve vs. USB URB submission freeze)
- Candidate mitigations (NOT yet attempted — change-control gate):
  - Disable USB autosuspend on the CM108 at runtime:
    `echo -1 > /sys/bus/usb/devices/1-1.1/power/autosuspend` +
    `echo on > /sys/bus/usb/devices/1-1.1/power/control`
  - Watchdog-level PCM reopen on repeated gap events
  - Different USB port / different audio adapter
- Repo rule: no code changes until user explicitly approves.

**Session note saved:** `/memories/session/ltc-dropout-investigation.md`

**Production image policy (2026-04-22):** Diagnostics (ltcmon, verbose
logging) stay OUT of production images — deploy on demand from `tools/`.
See `/memories/repo/clock8002-production-image-policy.md`.

**Source3 "Playback" not displaying (2026-04-22) — resolved:** Mitti was
still configured to send OSC to the previous unit's IP. IP changed with the
new SD card. Not a code/config bug on the clock side. Fix: update Mitti OSC
destination to `192.168.8.246:1245`.

---

## Current State

- Repository: jpkelly/clock8002
- Active release line: v1.x
- Latest tagged release: **v1.3.1** — Trixie tarballs on GitHub (default + gerry)
- Latest Buildroot tag: **v1.3.2** — `buildroot` branch, commit `7dcaa69`
- master HEAD: **`3ed5c9d`** (RELEASING: add fresh-install smoke test step)
- **Active branch: `buildroot`** (renamed from `feature/sdl3-migration`)
- Buildroot SD card image on Mac Desktop: **`piclockBR-b93633c-sdcard.img`** (release build, no SSH key)
- **piclockBR test unit** (192.168.8.246): running `c1fc28a` image (v1.3.2), all features confirmed working
  - OLED logo + stats: **working at boot**
  - sdl3-clock HDMI: **working at boot**
  - WiFi AP (piclockBR-ap): **working**
  - Power button shutdown: **working at boot** (stable `/dev/input/by-path/` symlink)
  - Network config from `network.ini`: **working** (static IP, hostname, AP — all verified after reboot)
  - alsa-ltc: **stable** — 0 USB errors, 0 restarts (PID stable 15+ min), LTC decoding to display confirmed
  - `authorized_keys` from `/boot/piclock/authorized_keys`: **tested and working** — key-based SSH confirmed
  - Static IP from `network.ini`: **working on first boot** — `S43piclock-network-prep` writes NM connection file before NM starts, no DHCP race
  - Build host: pi@cm5.local (10.0.0.101)
- **3rd party reference unit (192.168.8.246 / 10.0.0.131)**: `root` / `clockworkadmin`. BusyBox init. alsa-ltc fixed (was using `-` for device, now uses `plughw:${ALSA_CARD:-2},0`). LTC rolling on display. 0 USB errors.
- **2GB Pi 5 board #1** (piclockTG.local): fresh Trixie 6.12.47, v1.3.1 gerry. EEPROM downgraded to 2025-05-08. Monitor running.
- **2GB Pi 5 board #2** (piclockTD.local): fresh Trixie 6.12.47, v1.3.1 default. Stable. EEPROM 2025-05-08.
- `buildroot-prototype` branch: fully merged into master

## SDL3 Migration Status (branch: buildroot)

### Current state (2026-04-20)
- Branch renamed `feature/sdl3-migration` → `buildroot`; tagged **v1.3.2**
- Branch HEAD: **`c1fc28a`** (pre-tag) → tagged commit includes sdl3-clock feature parity work
- All boot issues fixed: logo, clock, WiFi AP, power button, network config
- **Test unit** (192.168.8.246): live-deployed, all features confirmed working from cold boot
- `piclockBR-c1fc28a-sdcard.img` built on cm5, transferred to Mac Desktop

### Recent commits (session 2026-04-20 — power button + network + BusyBox compat)
- `06d3715`: buildroot: add power button shutdown handler for BusyBox init
- `ab7e80e`: power-button: use nohup to detach handler from init session
- `720d883`: power-button: wait for /dev/input/event0 before reading events
- `855151c`: HANDOFF: add power button, document BusyBox boot-timing pattern
- `c1fc28a`: power-button: use stable by-path symlink instead of hardcoded event0
- `d62f6dd`: piclock-network: BusyBox compatibility for Buildroot

### Prior commits (session 2026-04-19/20 — Buildroot boot fixes)
- `934e43a`: oled: fix SyntaxWarning on regex string literal
- `c05d2c7`: oled: fix INI_PATH to find clock.ini on Buildroot
- `ba63500`: oled: blank display on SIGTERM/SIGINT for clean shutdown
- `ff229c0`: clock_pokemon: unbind fbcon before launching sdl3-clock
- `a9feaf7`: buildroot: add fbcon=map:10 to cmdline to prevent fbcon holding DRM master
- `5609f6d`: fix boot: use absolute paths for logo, export HOME=/root in watchdog

### BusyBox init boot-timing pattern (unified root cause)
All boot failures on BusyBox init share the same root cause: **rcS runs S## scripts
before devices, environment, or services are ready**. Unlike systemd (device units,
After= dependencies), BusyBox rcS is a sequential loop with no dependency tracking.

| Issue | Root Cause | Fix |
|-------|-----------|-----|
| OLED logo black at boot | `HOME` unset → `~/piclockLogo.bin` → `/piclockLogo.bin` | Absolute path `/root/piclockLogo.bin` with fallback |
| sdl3-clock crash-loop at boot | `HOME` unset → fonts not found; fbcon held DRM master | `export HOME=/root` + `fbcon=map:10` + device wait |
| Power button not working at boot | `/dev/input/event0` not yet created by eudev | Wait loop (up to 30s) for device node |
| OLED blank user/pass/port | `INI_PATH` hardcoded to Trixie path | Added `/boot/piclock/clock.ini` Buildroot fallback |
| WiFi AP not broadcasting | `piclock-network.sh` ran before wlan0 ready | 3-retry loop with 5s delay |
| SyntaxWarning in oled_daemon | Unescaped `\.` in non-raw string | Changed to `r"..."` |

**Fix pattern**: Always poll/wait for the required resource (device node, env var,
network interface) before using it. Never assume it exists at S## script time.

### Prior commits (session 2026-04-18)
- `96f45ac`: fix ifeq-in-recipe; enable WiFi AP in gerry network.ini
- `8bc6c5e`: add S98oled BusyBox init script for oled_daemon
- `8ad295e`: install authorized_keys from /boot/piclock/ at boot
- `af98d5e`: fix OLED (i2c-dev module, sdl3-clock path) and WiFi AP (S45piclock-network)
- `2b81719`: add CONFIG_I2C_DEV=m; wait for NM before configuring network

### USB audio root cause — BusyBox init / pokemon watchdog approach
- **3rd party unit** (10.0.0.131): kernel `6.12.41-v8` commit `590178d5`, BusyBox inittab — **zero retire_capture_urb errors**
- **Our prior approach**: systemd ExecStartPre with `rmmod`+`modprobe snd_usb_audio` on every alsa-ltc restart — triggers RP1 xHCI endpoint lock → EIO reads → `usb_set_interface -110`
- **New approach** (commit `e247328`): load `snd_usb_audio` once at boot via S11modules, then restart only the userspace `alsa-ltc` process (pokemon watchdog) — never cycles the driver
- **Page size**: Both our old build and 3rd party are 4K pages. `-16k` in kernel CONFIG_LOCALVERSION is misleading — not the issue. DO NOT revisit this.

### Changes in `e247328`
- `defconfig.sample`: kernel commit `359f37f0` → `590178d58b730e981099fdcb405053a000e79820`, `BR2_INIT_SYSTEMD` → `BR2_INIT_BUSYBOX`
- `clock8002.mk`: removed all systemd service file installs
- New rootfs overlay `etc/init.d/`: S11modules, S03copy_alsa-ltc_files, S03copy_clock_files, S99alsa-ltc, S99clock
- New rootfs overlay `root/`: alsa-ltc_pokemon.sh, alsa-ltc_cmd.sh, clock_pokemon.sh, clock_cmd.sh
- `post-build.sh`: fully rewritten — removed all systemd-specific logic
- Deleted orphaned: `etc/systemd/system/piclock-network.service`, `usr/lib/systemd/system/alsa-ltc.service`, `etc/udev/rules.d/99-usb-audio-power.rules`

### What was ported to sdl3-clock (vs sdl-clock master)
- `AppVersion` field + config stamping (data.go, config.ini.go, main.go)
- `CueSecondDisplay` option skeleton (option exists, no DRM implementation)
- `clock.ini.default`: removed hardcoded `app-version`, changed `Face=quad` → `Face=text`

### Features NOT yet ported to sdl3-clock
| # | Feature | sdl-clock files |
|---|---------|-----------------|
| 1 | **Quad face** — 4-source text clock | `text.go`, `data.go`, `http.go`, `config.html.go` |
| 2 | **Dual face** — 2-source text clock | `text.go`, `data.go`, `http.go`, `config.html.go` |
| 3 | **DRM mirror** — second HDMI via KMS/DRM dumb buffer | `drm_mirror_linux.go`, `drm_mirror_other.go`, `second_display_probe.go` |
| 4 | **PerfectCue icon on HDMI-2** — cue icon mode | `drm_cue_linux.go`, `drm_cue_other.go`, `second_display_probe.go` |
| 5 | **Configurable PerfectCue geometry** — cue-pos-x/y, cue-size | `cue.go`, `data.go` |
| 6 | **Sync Time web API** — `/api/settime` + RTC sync | `http.go` |
| 7 | **Cue test API** — `/api/cue` | `http.go` |
| 8 | **Atomic config import** — temp+validate+rename | `http.go` |
| 9 | **SDL resource leak fixes** — destroyTextClock/Audio on hot-reload | `text.go`, `audio.go`, `main.go` |
| 10 | **Row 4 color/alpha** — configurable color + alpha for text clock row 4 | `data.go`, `http.go`, `config.html.go` |
| 11 | **Network-aware info overlay** — shows eth0/wlan0 IPs and WiFi AP SSID on 'I' overlay | `clock/engine.go` |
| 12 | **Config version display** — "Loaded config version" in web UI + `/export` download link | `http.go`, `config.html.go` |
| 13 | **Symlink-safe config path** — `resolvedConfigPath()` resolves symlinks before import/save | `http.go` |
| 14 | **Web UI teal theme** — color scheme changed from pink/magenta to teal (`#006D88`); tab, heading, and form colors updated | `config.html.go` |

### Session 2026-04-20 — alsa-ltc stability + authorized_keys test

**alsa-ltc stability confirmed** (both our Buildroot unit and 3rd party):
- Root cause of prior crash loops: `alsa-ltc_cmd.sh` used `-` (auto-detect) which fails to open CM108 on BusyBox kernels, causing a continuous crash/restart loop that floods VL805 with `usb_set_interface -110` errors
- Our fix (already in image via pokemon watchdog): `ALSA_CARD` detected dynamically from `/proc/asound/cards` → used as `plughw:${ALSA_CARD:-2},0`
- Applied same fix to 3rd party unit (`/root/alsa-ltc_cmd.sh` + `/boot/piclock/authorized_keys` installed)
- Result on both units: 0 USB errors, stable PID over 15+ min, LTC decoding confirmed on display

**authorized_keys feature tested**:
- `S03copy_clock_files` already implements: copies `/boot/piclock/authorized_keys` → `/root/.ssh/authorized_keys` (700/600 perms) at every boot
- Tested live: wrote key to `/boot/piclock/authorized_keys`, manually triggered copy, passwordless SSH confirmed working
- FAT boot partition makes this accessible from any OS without ext4 support

### Still pending
- Remaining sdl3-clock feature gap items (see table above — items 1–14)
- No blocking issues on the Buildroot platform
- Port features 1–14 from table above (DRM mirror/cue most complex)

### Branch rule
- `v4/` Trixie files are off-limits for Buildroot-only fixes on this branch
- Buildroot-specific changes go in `buildroot-external/` only

### Key credentials / commands
- piclockBR SSH: `sshpass -p 'clockworkadmin' ssh -o StrictHostKeyChecking=no root@10.0.0.128`
- 3rd party unit: `sshpass -p 'clockworkadmin' ssh -o StrictHostKeyChecking=no root@10.0.0.131`
- Monitor build: `ssh pi@cm5.local 'tail -f /tmp/br-build.log'`
- Binary deploy (no reflash): `scp pi@cm5.local:~/buildroot/output/target/opt/clock8002/sdl3-clock root@10.0.0.128:/opt/clock8002/sdl3-clock`
- Image transfer after build: `scp pi@cm5.local:~/buildroot/output/images/sdcard.img ~/Desktop/piclockBR-ab5139d-sdcard.img`

---

### Active Investigation: VL805 xHCI crash on piclockTG (2026-04-14)

**Symptom:** VL805 xHCI controller dies with `Host System Error` / `HC died` at random intervals (~55min, ~6h50m). USB hub and audio device disappear. alsa-ltc crash-loops. Requires hard power cycle to recover.

**Root cause finding:** Two differentiators from piclockTD (stable):
1. `[4.9s] Undervoltage detected!` in TG dmesg on every boot — TD has none
2. TG was running EEPROM firmware **2025-12-08** (`2226a853`); TD has **2025-05-08** (`69471177`)

**Key observation:** After downgrading TG EEPROM to 2025-05-08, the undervoltage event **disappeared** on the same PSU and hardware. This implicates the newer firmware as the primary driver — it likely draws more current during VL805 init (different ASPM / power sequencing), pushing the PSU below threshold and triggering the PCIe bus fault.

**Action taken (2026-04-14):**
- Hard power cycled TG to recover USB
- Downgraded TG EEPROM: `sudo rpi-eeprom-update -d -f /lib/firmware/raspberrypi/bootloader-2712/stable/pieeprom-2025-05-08.bin`
- Rebooted — firmware confirmed `69471177`, no undervoltage, throttle `0x0`, USB healthy
- 1-minute monitor running (`~/monitor.sh` → `~/monitor.log`)

**Decision gates:**
- 6h: check TG monitor log for usb-hub=0 / usb-audio=0 / usb-errors
- 24h: final determination
- If TG survives → firmware was the cause → document and close
- If TG still fails → PSU is the cause → swap PSU between TD and TG

**Monitor restart (after any reboot):** `nohup bash /home/pi/monitor.sh > /dev/null 2>&1 & echo "PID=$!"`

**Bug (`install.sh`):** `~/.config/clock-8001/` was created as root when running `sudo bash install.sh` on a fresh system — `sdl-clock` (running as `User=pi`) could not open the log file and exited immediately on every restart attempt. Only affected fresh installs where the directory didn't already exist.

**Fix (`16302b4`):** Added `chown "${INSTALL_USER}:${INSTALL_USER}" "${CONFIG_DIR}"` after `mkdir -p`. One line.

**Process note:** v1.3.0 was released without catching this because soak test units had the directory from a prior deploy. Fresh-install smoke test step added to RELEASING.md (step 7) to prevent recurrence.

### VL805 xHCI crash — EEPROM firmware implicated (Issue #39, re-opened 2026-04-14)

**Prior resolution (2026-04-12):** Reflashed piclockTG with fresh Trixie — appeared stable. Both boards soaking.

**Re-occurrence (2026-04-14):** piclockTG failed again — VL805 xHCI `HC died` at ~55min. Root cause investigation found new differentiator: TG had EEPROM firmware **2025-12-08** (`2226a853`) vs TD's **2025-05-08** (`69471177`). Pre-downgrade: TG showed `Undervoltage detected!` at boot on same PSU. Post-downgrade: undervoltage gone, throttle `0x0`. EEPROM downgraded to 2025-05-08 and soak restarted. See "Active Investigation" section above.

### Resolved: Static IP timecode not displaying

**Root cause (2026-04-12):** `sendto(255.255.255.255)` returns `ENETUNREACH` when no default gateway is configured. The gerry `network.ini` comments out the gateway line.

**Fix (`f584ec2`):** Added `resolve_subnet_broadcast()` to `alsa-ltc.c` — uses `getifaddrs()` to resolve `255.255.255.255` → actual subnet broadcast address (e.g., `192.168.8.255`). Prefers wired interfaces (eth*/end*) over wireless. Re-resolves on `sendto()` failure and on every 30s heartbeat (catches DHCP subnet changes). Tested: DHCP reboot x2 ✅, Static reboot ✅.

### Fixed in v1.3.0 (2026-04-10/11/12/13 — alsa-ltc enhancements + hardware debugging + subnet broadcast)
- **alsa-ltc grammar fix + OSC suppression** (`666882e`): Fixed `setted` → `set` in 5 hw_params log messages. Suppressed OSC send error flood — logs once on first failure, logs recovery message with count when send succeeds again.
- **`-v` verbose split** (`c7e3a60`): Card info and 30s heartbeat (frames, LTC count, errors) now always on. `-v` adds activity dots, ALSA HW params, and peak signal level.
- **Configurable `[fps]` argument**: LTC decoder frame rate configurable via CLI (default 25). TouchDesigner source is 30fps.
- **LTC gap detection**: Logs warning when no LTC frame decoded for >1 second. Always on, only fires on anomalies.
- **Hardware fault isolated**: piclockBR Board A (Pi 5) has faulty VL805/PCIe — CM108 USB lockup after 30-100s regardless of dongle, ribbon cable, or software image. Board B is stable. Board A retired.
- **Removed `-v` from service files**: Both Trixie and Buildroot overlay service files updated.
- **Subnet broadcast resolution** (`f584ec2`): `resolve_subnet_broadcast()` using `getifaddrs()` — fixes timecode not displaying when static IP has no default gateway. Prefers wired interfaces, re-resolves on failure and every 30s heartbeat.
- **Install reboot prompt** (`38e74d8`): Added "Reboot to finish installation: sudo reboot" message at end of install.sh output.

## Buildroot Status (post-merge)

- `buildroot-prototype` branch merged into `master` at `afbbc01` — all Buildroot work is now on master
- Tracking issue: **#28** "Buildroot: post-merge validation and remaining work"
- Build host: pi@cm5.local, `~/buildroot` (Buildroot 2025.02)
- Mesa 25.0.7 (upgraded from 24.0.9 — applied via `buildroot-external/scripts/apply-build-host-patches.sh` — see #29)
- SSH: `BR2_PICLOCKKEY` env var for optional key injection; release images are password-only (`clockworkadmin`)
- **After any clean Buildroot checkout**, run: `buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot`

### Fixed this session (2026-04-06/07)
- **alsa-ltc exit code** (`8a4c562`): All `goto cleanup` paths previously returned 0 — systemd never triggered `Restart=on-failure`. Now defaults to exit code 1; set to 0 only after successful setup; set back to 1 on unrecoverable error (10 consecutive read failures).
- **alsa-ltc snd_pcm_drop** (`8a4c562`): Added `snd_pcm_drop(capture)` before `snd_pcm_close(capture)` to prevent potential hang on broken USB device.
- **Buildroot package recipe** (`71c2321`): Removed `99-alsa-ltc-usb.rules` install line from `clock8002.mk` — file was deleted in v1.2.5 but recipe was not updated, causing Buildroot image builds to fail with `install: cannot stat ... No such file or directory`.
- **USB cable root cause**: piClockN and piclockM failures traced to flat ribbon USB cable (no twisted D+/D- pairs). Shorter twisted cable = intermittent; longer cable = fails every boot. Software recovery: sysfs authorized toggle (`echo 0/1 > /sys/bus/usb/devices/1-1.1/authorized`) proven to clear stuck C-Media capture state.
- **usb-monitor service**: Deployed on piClockN and piclockM — `/home/pi/usb-monitor.sh`, 60s interval, logs USB hub + audio device presence and alsa-ltc restart count.

### Open Buildroot issues
- **#28**: Post-merge validation — Trixie regression test, `broadcast.go` leak fix, boot splash, Wi-Fi AP test
- **#29**: Mesa 25.0.7 + host-xz manual patches — now automated via `buildroot-external/scripts/apply-build-host-patches.sh` (closed)

### Fixed this session (2026-04-01 evening)
- **Sync Time button** (`6f76320`): switched `date --set "@epoch"` → `date -u -s "YYYY-MM-DD HH:MM:SS"` and `hwclock --systohc --utc` → `hwclock -w -u`. POSIX short flags work on both GNU coreutils (Trixie) and BusyBox (Buildroot). Verified: system clock and RTC both set correctly.
- **DT overlays** (`3f70e46` + `123d86c`): config.txt now enables i2c_arm, rtc_bbat_vchg, dwc2 host mode, uart0-3. post-image.sh copies both generic and -pi5 overlay .dtbo variants (firmware's overlay_map.dtb redirects e.g. uart1 → uart1-pi5 on Pi 5). All 5 serial devices now visible (ttyAMA0, ttyAMA1, ttyAMA2, ttyAMA3, ttyAMA10).
- **post-image.sh config.txt sync** (`c0ebb96`): post-image.sh now force-copies board config.txt/cmdline.txt over rpi-firmware cached copies before image generation. Eliminates need for `rpi-firmware-dirclean` after config.txt edits.

### Merge-to-main checklist (tracked in issue #24)
1. SDL rendering fixes (PIXELFORMAT_UNKNOWN, surfaceToABGR8888, SetBlendMode) — master panics without these
2. date/hwclock POSIX short flags in http.go
3. install.sh sudoers rule update (`--systohc --utc` → `-w -u`) — must land with item 2
4. clock8002.service `User=root` vs `User=pi` — needs decision (Buildroot=root, Trixie=pi)
5. piclock-network.sh hostname block moved to end — safe for Trixie, fixes DHCP override race
6. Render diagnostic logging (`logRenderDiag`) — decide: keep or strip
7. Import reordering (cosmetic, harmless)
8. `renderSignal()` missing `SetRenderTarget(nil)` — real bug fix
9. DRM mirror files: `drm_mirror_linux.go`, `drm_mirror_other.go` — new files, direct DRM/KMS mirror for second HDMI
10. `second_display_probe.go` — updated mirror path: `isHDMI1Connected` → `isSpareHDMIConnected`, calls `findSpareHDMIConnector` instead of hardcoded HDMI-A-1
11. DRM cue files: `drm_cue_linux.go`, `drm_cue_other.go` — `updateCueDRMBuffer()` writes icons directly into DRM dumb buffer (Option D)
12. `second_display_probe.go` — cue mode now uses `initDRMMirror()` + `updateCueDRMBuffer()` instead of `fbi`+PNG; removes `bytes`, `image/png`, `os/exec` (fbi) dependency

### Fixed this session (2026-04-01 — EEPROM provisioning)
- **rpi-eeprom package** (`772f5b3`): Added Buildroot external package for Pi 5 EEPROM management (rpi-eeprom-config, rpi-eeprom-update, rpi-eeprom-digest + BCM2712 firmware blobs). Reads EEPROM via nvmem — no vcgencmd needed.
- **Red screen fix**: Changed `BOOT_ORDER` from `0xf461` (SD+NVMe+USB+restart) to `0xf1` (SD-only) on both test units via `rpi-eeprom-config --apply`. Eliminated the red PCIe probe screen caused by bootloader NVMe enumeration on quiet boot.
- **1GB Pi 5 validation**: Image boots and runs on Pi 5 1GB (160MB used / 774MB available).

### Fixed this session (2026-04-02 — EEPROM provisioning final resolution)
- **Boot-partition EEPROM approach removed** (`65af133`): After extensive testing, the `pieeprom.upd` + `recovery.bin` approach baked into the image was proven unreliable. Any Pi 5 whose EEPROM firmware is the same version or newer than Buildroot's blob (Dec 2025) will enter a red screen loop every other boot — the firmware repeatedly tries and fails to apply the downgrade. This includes any unit previously booted with Trixie (which auto-updated EEPROM to Jan 2026).
- **Correct approach**: Manual provisioning via `rpi-eeprom-update -d -f` with a custom config blob. Works unconditionally regardless of installed firmware version. One slow boot (~15s), then clean forever. Documented in README "EEPROM Provisioning (Pi 5)" section.
- **piclockt.local (Pi 5 1GB)** provisioned manually to `BOOT_ORDER=0xf1` ✅ — no more red screen.
- **Issue #26**: Closed — EEPROM provisioning documented as manual step; boot-partition approach removed.

### Fixed this session (2026-04-02 — DRM mirror + cue mode)
- **DRM mirror working** (`2ee57fa`): Root cause found — `findHDMI1Connector()` was hardcoded to target HDMI-A-1, but SDL already renders there. Fix: `findSpareHDMIConnector()` scans all connected HDMI outputs, identifies SDL's CRTC (highest fb_id), picks the other. Both displays now show the clock on piclockBR. DRM state confirmed: plane-2→crtc-92 (SDL, fb=685) + plane-3→crtc-104 (mirror, fb=682).
- **DRM cue mode working** (`a5929ef`): Replaced `fbi`+PNG disk cache path with direct DRM dumb buffer writes (Option D). `probeSecondDisplayOutput()` cue branch calls `initDRMMirror()` then `updateCueDRMBuffer(off)`. `syncSecondDisplayCueDisplay()` calls `updateCueDRMBuffer(desired)` — renders icon via `renderCueVisualImage()` and writes XRGB8888 directly into the dumb buffer. No `fbi` binary or `/dev/fb0` required. Web GUI toggle (PerfectCue section) switches modes live without restart. Verified working on piclockBR at `a5929ef`.

### Remaining work
- DRM mirror simplification: master swap dance (DROP/SET_MASTER) may be unnecessary now — targeting fbcon's CRTC, not SDL's
- DRM mirror robustness testing: service restart, HDMI hot-plug, extended runtime
- Trixie regression testing

## Issue #23 Status (Dual HDMI Output)

- **DRM mirror working on Buildroot** (`2ee57fa` on `buildroot-prototype`) ✓
- **DRM cue mode working on Buildroot** (`a5929ef` on `buildroot-prototype`) ✓
- Implementation uses direct DRM/KMS ioctls (not SDL second window) for both modes.
- Behavior:
  - `--cue-second-display`: HDMI-2 shows full-screen PerfectCue icons (forward/reverse/blank) written directly into DRM dumb buffer — no `fbi`, no `/dev/fb0`.
  - No flag (default): HDMI-2 mirrors the main clock via DRM dumb buffer + CPU pixel copy from SDL renderer.
- Web GUI toggle (PerfectCue section checkbox) switches modes live — no restart required.
- Key files: `drm_mirror_linux.go`, `drm_mirror_other.go`, `drm_cue_linux.go`, `drm_cue_other.go`, `second_display_probe.go`.
- `findSpareHDMIConnector()` is fully dynamic — scans all HDMI outputs, identifies SDL's CRTC by highest fb_id, picks the spare. No hardcoded connector/CRTC IDs.
- Issue remains open pending Trixie validation and robustness testing.

## Hot-Plug Policy (Current Expected Behavior)

- Do not crash on HDMI plug/unplug events.
- Main clock on primary output must continue running regardless of HDMI-2 state.
- Icon mode (`fbi`) should tolerate disconnect/reconnect and resume when HDMI-2 is available.
- Mirror mode (DRM/KMS direct) uses a dumb buffer on the spare CRTC. HDMI-2 reconnect likely requires process restart since CRTC/connector assignment is fixed at init time.
- Test matrix still required before closing issue #23:
  - Boot with HDMI-1 only
  - Boot with HDMI-2 only
  - Boot with both connected
  - Unplug/replug HDMI-2 while running in icon mode
  - Unplug/replug HDMI-2 while running in mirror mode

## Recent Release Notes

- v1.1.7 includes:
  - Dynamic app-version stamping for clock.ini via `clock.AppVersionForConfig()` (uses injected git tag; strips leading `v`).
  - Config rewrite/save paths now use dynamic app-version instead of hardcoded `clock.Version`.
  - Gerry profile updates preserved in `clock.ini.gerry` (text face, Limitimer/TOD-LTC/Playback, row colors, limitimer receive mode).
  - README quick-install URLs updated to v1.1.7.
- GitHub release notes continue to use `.github/release-notes-template.md` with `__VERSION__` substitution.

## OLED Splash Version Overlay

- OLED startup logo now overlays build version in lower-right, raised slightly from bottom.
- Regex now supports major versions beyond v0 (`v[0-9]+...`), so v1.x tags display correctly.

## Release Process (Current)

1. Update `CHANGELOG.md`.
2. Ensure `README.md` quick-install commands point to the new version.
3. Commit, tag, and push:
   - `git tag v1.x.y`
   - `git push origin v1.x.y`
4. Build on pi5start from a fresh clone at the tag:
   - `make release-all GIT_TAG=v1.x.y` (produces default + gerry tarballs)
5. Publish GitHub release with both tarballs.
6. Deploy to piclock.local and run installer.
7. Start and verify `clock8002`, `alsa-ltc`, and `oled_daemon`.
8. Report deployed short commit hash.

## Repository Instructions Already Added

- Build machine: `pi@pi5start.local`
- Test machine: `pi@piclock.local`
- Ask before running tests.
- Include both default and gerry artifacts for each release.
- Use repository version line (`v1.x` and onward), ignore inherited upstream `v4.x` tags.
- Update README quick-install URL/version during release cuts.
- Avoid `pkill -f /opt/clock8002/sdl-clock` and `pkill -f /opt/clock8002/alsa-ltc` in one-shot SSH deploy commands (can trigger SSH exit 255).
- Prefer `systemctl stop` + `systemctl kill` for teardown.
- Run installer with log capture and explicit exit check.

## Hard Rules (Do Not Skip)

- Build host rule: build/release artifacts for clock8002 are authoritative only when built on `pi@pi5start.local` from a fresh clone at target ref.
- Do not use Mac local build results for release/deploy validation.
- Gerry variant rule: deployment is only valid when both `/boot/firmware/piclock/clock.ini` and `/boot/firmware/piclock/network.ini` match Gerry settings.
- Installer behavior note: `install.sh` preserves existing `/boot/firmware/piclock/clock.ini` and only installs packaged clock.ini on fresh install; existing units may require explicit config copy.

## Useful Commands

- Check local working tree:
  - `git status --short`
- Build release artifacts on pi5start (fresh clone approach):
  - `ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && git checkout v1.x.y && make release-all GIT_TAG=v1.x.y'`
- Deploy release tarball to piclock.local from local Mac relay:
  - `scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-v1.x.y-default-linux-arm64.tar.gz /tmp/`
  - `scp /tmp/clock8002-v1.x.y-default-linux-arm64.tar.gz pi@piclock.local:/tmp/`
  - `ssh pi@piclock.local 'set -e; sudo systemctl stop clock8002.service alsa-ltc.service oled_daemon.service || true; sudo systemctl kill clock8002.service alsa-ltc.service oled_daemon.service || true; mkdir -p /tmp/clock8002-install && rm -rf /tmp/clock8002-install/clock8002-v1.x.y-default-linux-arm64; tar xzf /tmp/clock8002-v1.x.y-default-linux-arm64.tar.gz -C /tmp/clock8002-install; cd /tmp/clock8002-install/clock8002-v1.x.y-default-linux-arm64; sudo bash install.sh > /tmp/clock8002-install-v1.x.y.log 2>&1; echo INSTALL_EXIT:$?; sudo systemctl start clock8002.service alsa-ltc.service oled_daemon.service'`
- Verify services on piclock:
  - `ssh pi@piclock.local 'systemctl is-active clock8002 alsa-ltc oled_daemon'`
- Force-apply gerry config pair on existing unit:
  - `ssh pi@piclock.local 'sudo cp /tmp/clock8002-v1.x.y-gerry-linux-arm64/clock.ini /boot/firmware/piclock/clock.ini && sudo cp /tmp/clock8002-v1.x.y-gerry-linux-arm64/network.ini /boot/firmware/piclock/network.ini && sudo reboot'`

## Release Notes Template Workflow

- Template file: `.github/release-notes-template.md`
- Placeholder token: `__VERSION__`
- Generate release notes file:
  - `VERSION=v1.x.y; sed "s/__VERSION__/${VERSION}/g" .github/release-notes-template.md > /tmp/release-notes-${VERSION}.md`
- Publish release with templated notes:
  - `gh release create "${VERSION}" "clock8002-${VERSION}-default-linux-arm64.tar.gz" "clock8002-${VERSION}-gerry-linux-arm64.tar.gz" --title "${VERSION}" --notes-file "/tmp/release-notes-${VERSION}.md"`

## Dev-Deploy Workflow (Feature Branch Testing)

Use this when testing a feature branch on piclock — **not** a formal release.
`install.sh` must always run on the target machine from a flat release directory (never from the source tree — `*.ttf` won't be found).

```bash
# 1. Clone branch, build, and package on pi5start
ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone --depth 1 --branch BRANCH https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && make release NETWORK_CONFIG=default'

# 2. Relay tarball through Mac to piclock
#    (GIT_TAG defaults to nearest ancestor tag, e.g. v1.0.3)
scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-*-default-linux-arm64.tar.gz /tmp/
scp /tmp/clock8002-*-default-linux-arm64.tar.gz pi@piclock.local:/tmp/

# 3. Extract and install on piclock
ssh pi@piclock.local 'cd /tmp && tar xzf clock8002-*-default-linux-arm64.tar.gz && cd clock8002-*-default-linux-arm64 && sudo bash install.sh'

# 4. Verify
ssh pi@piclock.local 'systemctl is-active clock8002'
```

> **After every deploy to the test unit, report the short GitHub commit hash (first 7 characters) that was deployed.**

## Buildroot Known Issues

- **Mesa 25.0.7 + host-xz patches**: Automated via `buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot` — run after any clean Buildroot checkout. See #29 (closed).
- **SSH key hardcoded**: Personal SSH public key in `post-build.sh` — should be env-var driven. Tracked in #30.
- **No root password set**: Buildroot images have blank root password → SSH password login blocked on release images. Tracked in #30 (fix: `clockworkadmin`).
- **`broadcast.go` UDP socket leak**: `singleAddr()`/`broadcastAll()` replace connections without closing old ones — ~1.9 MB/hr leak on Buildroot with stable alsa-ltc; much faster with crash-looping alsa-ltc. Tracked in #28.

## Stability Test — piclockT.local (in progress)

- Unit: Pi 5 1GB, Buildroot image `65af133` (v1.2.3), no swap
- Started: 2026-04-03 06:27 UTC
- Baseline RSS: ~44 MB; leak rate: ~1.9 MB/hr
- 21h checkpoint (03:27 UTC): RSS 86 MB, all services active, no OOM
- 24h checkpoint due: ~2026-04-04 06:27 UTC
- Monitor: `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 root@piclockT.local 'cat /root/stability.log'`
- Pass criteria: RSS ±50 MB from baseline, no OOM, mem-avail > 400 MB, all services active

## Documentation updated this session (2026-04-03)

- `README.md`: restructured — Platform folded into Requirements, EEPROM moved to top-level section, Service Operations trimmed, Creating a Release removed, config files table added
- `RELEASING.md`: new — full Trixie + Buildroot release procedure
- `buildroot-external/README.buildroot.md`: new — Buildroot build/flash/SSH/config reference
- `buildroot-external/README.buildroot.txt`: deleted (superseded)
- Release procedure now lives in `RELEASING.md`; see also `buildroot-external/README.buildroot.md`

### README accuracy audit (2026-04-03 afternoon) — all claims now verified
- Fixed: oled.ini description (`1e33012`)
- Fixed: OLED daemon + I2C enable merged into one conditional bullet (`06839ab`)
- Fixed: clock.ini path clarified as `/boot/piclock/clock.ini`; symlink at `~/.config/clock-8001/clock.ini` noted (`8a675fd`)
- Fixed: `--dump-config` moved from Config section to Building from Source section (`f9a0411`)
- Verified correct: web UI credentials (admin/clockwork), network.ini description, installer bullet list, config file paths

## alsa-ltc Enhancements (2026-04-11)

Committed changes in `v4/alsa-ltc.c`, `v4/alsa-ltc.service`, and Buildroot overlay.

### Binary changes (`alsa-ltc.c`)
- **Always-on diagnostics** (no flag needed): ALSA card info at startup, 30s heartbeat with frame count, LTC decode count, and error tally.
- **`-v` verbose flag**: Activity dots (`.` per decoded LTC frame), ALSA hardware params (buffer_size/period_size/periods) at startup, peak audio signal level (0–32767) in heartbeat.
- **Configurable sample rate**: Default changed 48000 → 44100 (matches upstream). Optional 4th positional arg `[sample-rate]`. Prints warning if hardware negotiates different rate.
- **Early DISCONNECT exit**: `snd_pcm_state()` checked on read errors — if `DISCONNECTED` or `-ENODEV`, exits immediately with explicit message instead of burning 10 retries. PCM state name added to all error log lines.
- Usage: `alsa-ltc [-v] <device> <ip> <port> [sample-rate]`

### Service file changes (`alsa-ltc.service` + Buildroot overlay)
- **`-v` removed from production**: Heartbeat and card info are always-on; add `-v` to command line for USB debugging.
- **`ExecStopPost=+` USB reset**: On failure exit, scans for C-Media USB devices and toggles sysfs `authorized` 0→1 to reset the dongle before the next restart.
- **`Restart=on-failure`**: Replaces `always` — restarts only on error exits, not clean stops.
- **`RestartSec=5`**: Reduced from 30s — faster recovery with early DISCONNECT exit.
- **`StartLimitBurst=5` / `StartLimitIntervalSec=60`**: Caps restart attempts on deeply locked hardware.
- **Dual service file rule**: Both `v4/alsa-ltc.service` and `buildroot-external/.../rootfs-overlay/.../alsa-ltc.service` must be kept in sync.

### Hardware debugging (piclockBR CM108 lockup)
- **Root cause**: Faulty PCIe ribbon cable between Pi 5 SoC and VL805 xHCI USB controller.
- Symptom: `usb_set_interface failed (-110/-62)` after 30-100s on every boot.
- Isolation: swapped dongle (still fails), swapped Pi board (works), swapped PCIe ribbon (works) → cable was the fault.
- Resolution: replaced ribbon cable; CM108 stable past 119s+ on original board.

### Documentation
- CHANGELOG.md: "Version 1.2.10 (unreleased)" section.
- README.md: "alsa-ltc command-line options" table, "USB recovery" subsection.

### Upstream binary analysis (root@192.168.8.245)
- Buildroot 2025.11, kernel 6.12.41-v8, BusyBox init, 22KB stripped aarch64
- Sample rate 44100 confirmed via `objdump -s -j .data`
- Activity dot always on; OSC pretty-printer is dead code
- Missing: retry loop, plughw, snd_pcm_drop, error counter, --version, SO_BROADCAST, signal handler

### Stability test (Apr 11, ~6h)
- piclockBR: clean — 0 USB errors
- piclockTG (Trixie): CM108 lockup at ~6h (`usb_set_interface -110`), rebooted to recover

## Next Suggested Release

- No immediate release pending.
- Prerequisites before next release: resolve #28 (Trixie regression test + broadcast.go fix), #29 (build host patches), #30 (SSH/password).
- alsa-ltc enhancements above should be included in next release after build/test validation.
