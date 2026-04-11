## Version 1.2.10 (unreleased)
* alsa-ltc
  * Add `-v` verbose flag — prints activity dot (`.`) for each decoded frame, ALSA hardware params (buffer/period sizes), and peak signal level in heartbeat
  * Card info and 30-second heartbeat are now always on (no flag needed); `-v` adds activity dots, ALSA negotiation details, and signal levels
  * Change default sample rate from 48000 to 44100 to match upstream hardware
  * Add optional `[sample-rate]` argument — overrides the default without rebuilding
  * Print warning when hardware negotiates a different sample rate than requested
  * Detect USB audio disconnect (`SND_PCM_STATE_DISCONNECTED` / `-ENODEV`) and exit immediately instead of retrying — lets systemd restart with a fresh device handle
  * Include ALSA PCM state name in all read-error log lines for USB troubleshooting
  * Add LTC decode count to heartbeat output
  * Add optional `[fps]` argument — configures LTC decoder frame rate (24, 25, or 30; default: 25)
  * Add LTC gap detection — logs `[gap] no LTC decoded for Xms` when timecode decode gap exceeds 1 second (always on)
* Service (`alsa-ltc.service`)
  * Run without `-v` by default — heartbeat and card info are always on; add `-v` to the command line when USB debugging is needed
  * Add `ExecStopPost` USB reset — on failure exit, toggles sysfs `authorized` on C-Media devices to reset the dongle before the next restart attempt
  * Change `Restart=always` to `Restart=on-failure` — only restarts on error exits, not clean stops
  * Reduce `RestartSec` from 30s to 5s — faster recovery now that early disconnect detection avoids wasted retries
  * Add `StartLimitBurst=5` / `StartLimitIntervalSec=60` — prevents infinite crash-loop when USB hardware is deeply locked

## Version 1.2.8
* Install
  * Fix FAT32 `/boot/firmware` remount after fstab update — config saves via web UI now work immediately without requiring a reboot
  * Enable `alsa-ltc.service` during install — was missing, service file was copied but never enabled

## Version 1.2.7
* alsa-ltc
  * Internal retry loop for USB audio device detection — polls for device and PCM handle for up to 15 seconds on startup instead of failing immediately
* Buildroot
  * Pin kernel to Trixie commit `359f37f0` (6.12.47) — prevents untested kernel regressions
  * Fast alsa-ltc restart with USB autosuspend disable in rootfs overlay
  * Fix hostname race — add `RequiresMountsFor=/boot` to rootfs-overlay network service
  * Add `alsa-ucm-conf`, `alsactl`, `amixer` to defconfig for proper ALSA card naming
  * Remove deleted `99-alsa-ltc-usb.rules` from package recipe
* Install
  * Mask `alsa-store.service` and `alsa-restore.service` to prevent corrupt ALSA state file from poisoning C-Media mixer on boot
* Network
  * `piclock-network.service`: add `RequiresMountsFor=/boot` to fix hostname race on Trixie

## Version 1.2.6
* alsa-ltc
  * Return non-zero exit code on error — systemd `Restart=on-failure` now triggers correctly instead of treating all exits as success
  * Add `snd_pcm_drop()` before `snd_pcm_close()` for clean capture teardown — prevents potential hang on broken USB device
* Buildroot
  * Remove deleted `99-alsa-ltc-usb.rules` from package recipe — Buildroot image builds were failing with `install: cannot stat '99-alsa-ltc-usb.rules': No such file or directory`

## Version 1.2.5
* Install
  * Config files moved to `/boot/firmware/piclock/` (FAT32 boot partition) — now accessible from Mac/PC when SD card is inserted (appears as `bootfs` volume with a `piclock/` folder)
  * `install.sh` migrates existing config files from `/boot/piclock/` to `/boot/firmware/piclock/` and updates symlinks on existing units
  * `install.sh` configures `/boot/firmware` with correct `uid`/`gid` mount options so the web UI can write back to `clock.ini` on the FAT32 partition
  * `install.sh` removes stale `/var/lib/alsa/asound.state` on install to prevent alsa-ltc `Input/output error` caused by corrupt ALSA state from a prior crash-loop
  * `install.sh` now uses `$SUDO_USER` to correctly identify the invoking user when run via `sudo bash install.sh` — fixes `User=` in service files, config symlinks, and fstab uid/gid
* alsa-ltc
  * Added `[Install]/WantedBy=multi-user.target` to `alsa-ltc.service` — `systemctl enable/disable alsa-ltc` now works correctly
  * Removed `99-alsa-ltc-usb.rules` udev rule — alsa-ltc is now a plain boot-time service
* Buildroot
  * `/boot/piclock/` config files were already on FAT32 in Buildroot (no change needed)
* Documentation
  * Updated all config path references from `/boot/piclock/` to `/boot/firmware/piclock/`
  * Removed `chown` from Gerry deploy commands (FAT32 does not support per-file ownership)

## Version 1.2.4
* Install
  * `install.sh`: seed `network.ini` hostname from system hostname on first install — avoids hardcoded `piClock` default when user sets a custom hostname via RPi Imager
* Buildroot
  * Remove hardcoded dev SSH public key from `post-build.sh`; SSH key now optional via `BR2_PICLOCKKEY` environment variable at build time — release images are password-only
* Documentation
  * README: add "Preparing the SD Card" section covering RPi Imager OS setup (username, password, SSH)
  * README.buildroot.md: add table of contents; separate Release build and Dev build command sections
  * RELEASING.md: document rule that Buildroot release images must never include an SSH key

## Version 1.2.3
* Buildroot
  * Remove boot-partition EEPROM provisioning (`pieeprom.upd` + `recovery.bin` baked into image): approach causes a red screen loop when installed EEPROM firmware is same version or newer than Buildroot's blob — which is the case for any Pi 5 previously booted with Trixie. EEPROM provisioning is now a documented manual step (see README)
* Documentation
  * README: added "EEPROM Provisioning (Pi 5)" section with `rpi-eeprom-update -d -f` command sequence for setting `BOOT_ORDER=0xf1` on factory-fresh units

## Version 1.2.2
* Bugfixes
  * `alsa-ltc`: start/stop via udev instead of boot-time service enable; fixes xHCI controller crash when USB audio device is behind a hub and not yet enumerated at `multi-user.target` start time
  * New file `99-alsa-ltc-usb.rules`: triggers `alsa-ltc.service` start when any `snd-usb-audio` device appears; stops service on device removal

## Version 1.2.1
* Bugfixes
  * `clock8002.service`: add `CAP_SYS_ADMIN` to `AmbientCapabilities` so DRM mirror mode works when running as non-root (`User=pi`); fixes "permission denied" on `DRM_IOCTL_SET_MASTER` / `SetCrtc` for the spare HDMI connector

## Version 1.2.0
* Features
  * Buildroot image variant: single SD card image with full clock8002 stack (sdl-clock, alsa-ltc, oled_daemon, bootsplash, Wi-Fi AP, power button shutdown)
  * DRM/KMS second display: mirror clock to spare HDMI connector via direct DRM dumb buffer (no desktop required)
  * PerfectCue second display mode: show full-screen cue icon on second HDMI
  * Boot splash screen (Buildroot only): fbv PNG splash while DRM/KMS initialises
  * Build environment indicator: `BR` badge on OLED splash and web GUI when running under Buildroot
  * Wi-Fi AP support (Buildroot): dnsmasq + wpa_supplicant D-Bus, persisted US regdomain
  * Power button shutdown (Buildroot): systemd-logind integration
* Bugfixes
  * Info overlay `I` key toggle now uses actual visibility state
  * `Restart=on-failure` so Q-key exit stays stopped
  * Version ldflags correctly passed to sdl-clock Go build under Buildroot
  * Buildroot/Trixie runtime detection via `/etc/os-release` instead of binary string grep or ldflags
  * OLED: restore `subprocess` import removed during `is_buildroot()` refactor

## Version 1.1.5
* Bugfixes
  * Skip timer background preload when `DynamicBG` is disabled
  * Skip static background load when `Background` is empty
* Ops / workflow
  * Documented deploy reliability rules to avoid SSH session drops and partial installs during remote deployment

## Version 1.1.4
* Bugfixes
  * Web config: clarify the full-screen cue icon setting as an overlay on the clock
* Documentation
  * README quick install updated for v1.1.4
  * README release workflow updated to publish both default and Gerry variants

## Version 1.0.3
* Bugfixes
  * Installer consistency report now prints `sdl-clock` version from embedded build metadata when `--version` is unsupported
* Documentation
  * Added reusable release notes template workflow for GitHub releases

## Version 1.0.2
* Documentation
  * README: specify Raspberry Pi OS Lite (64-bit) based on Debian Trixie

## Version 1.0.1
* Bugfixes
  * OLED splash version display now works with v1.x+ release tags

## Version 1.0.0
* First stable release under clock8002 versioning (v1.x)
* Features
  * OLED splash screen with build version overlay (lower-right corner)
  * alsa-ltc version reporting via `--version` flag

## Version 4.24.2
* Bugfixes
  * `/clock/timer/*/set` and negative seconds
    * Now we display the "-" icon if the provided time is negative

## Version 4.24.1
* Bugfixes
  * `/clock/timer/*/set` misscalculated hours

## Version 4.24.0
* Features
  * `/clock/timer/*/set` OSC command to set a timer to display an external value
  * Experimental "high resolution" timing mode that shows 1/100th second times

## Version 4.23.0
* Features:
  * Stereo LTC decoding. The raspberry pi can now decode both channels of a stereo LTC signal into different time codes and the clock can display them both. This requires a stereo capture interface. Tested with Behringer U-phoria.
    * Caveat: Older clocks will produce glitches with two ltc streams due to bad OSC matching code
* Bugfixes:
  * ToD and LTC display priorities were messed up since 4.19.0

## Version 4.22.0
* Bugfixes
  * Fix images for raspberry pi 2
  * General code cleanup pass on some integrations
    * Mitti
    * Tricaster
    * Vmix
  * Always send hh:mm:ss format strings as OSC feedback
    * This fixes the source time feedback on companion with limitimer
  * Log file creation truncates the existing file on startup


## Version 4.21.1
* Add a list of detected serial devices into the configuration web interface

## Version 4.20.0
* Features:
  * Add support for FTDI chipset based USB serial adapters to the raspberry pi image
    * This allows using RS485 USB adapters to connect to both limitimer and perfect cue simultaneusly. The adapters will be found as '/dev/ttyUSB0' for the first and then '/dev/ttyUSB1' etc for others.

## Version 4.19.2
* Features:
  * Sound cues can be muted per timer
  * Countdown timers can now trigger actions when they end
    * Restart a given timer, can be used to chain timers
    * Send a OSC message
  * Timers now remmeber their last used duration for restarting
  * Backgrounds can be now dynamically changed per the first running timer
    * Each timer can have a default background and timed backgrounds, like the voice files
* Bugfixes:
  * Raspberry pi images: Firmware update in 4.19.0 caused problems with serial communications
    like with limitimer due to renamed device tree overlay.
    * This has been fixed by completely overhauling the image creation.

## Version 4.19.1
* Bugfixes:
  * Clock bridging was using wrong encoding for the /clock/media OSC message, this caused
    excessive logging and weird delays in mitti and millumin timers.

## Version 4.19.0
* New features:
  * Support multiple timers per source
    * Supplied as comma separated list, ie. "3,1,4"
    * First active timer on the list is displayed on the clock
    * This allows for example configuring a primary and backup playout on same source
    * Thank you for Creative Techonlogy Finland for sponsoring this
  * Disguise d3 integration
    * Thank you to Creative Technology Finland for sponsoring this
  * Hyperdeck integration
    * Caveats: Hyperdeck only allows one incoming connection, but the clock can
      act as a relay for other software. This limits the clock to connecting to a
      single hyperdeck.
* Bugfixes:
  * Cleaner shutdown on OSX to avoid leaving ports reserved
  * Cleaner closing of OSC listeners on config reload
  * Updated raspberry-pi firmware to fix issues on some rpi3b+ boards

## Version 4.18.1
* Bugfixes:
  * udptime and overtime countdowns
    * We now respect the overtime count mode for sending the time.

## Version 4.18.0
* New features:
  * Build .deb packages for debian based linux distros
    * Currently building armhf (for raspberry pi's) and amd64 packages
    * use 'sudo apt install ./clock-800_1.2.3_arch.deb' to install
    * Icon to start the clock should be in "Accessories" on your window manager
      start menu
  * Added the "max" clock face.
* Bugfixes:
  * Tally messages not being shown on 288x144 clock face

## Version 4.17.0
* New feature:
  * Added support for DSAN Perfect Cue, see v4/perfectcue.md

## Version 4.16.0
* New features:
  * Mitti, Millumin, vMix and Pictural integrations are now per timer.
    * This means you can get the state from multiple players
* Config page overhaul
  * Better UI widgets to make the page more manageable
  * More contrast between the text and the background

## Version 4.15.0
* New features:
  * Ability to hide hours field on text clock timers
  * Ability to hide icons on text clock timers

## Version 4.14.0
* New features:
  * vMix integration
    * The clock can now show time remaining on videos playing in vMix
  * Tricaster integration
    * The clock can show DDR playback information and event timer info
    * Caveats: tricaster doesn't provide play/pause nor loop info on the API
  * GPIO output module for driving pulse per second/minute clocks
    * Uses two pins per pulse, meant for relays to create the alternating polarity pulses for clocks.
* Bugfixes:
  * Picturall: Additional work on the message parsers
    * Deal with literal \n sequences in text fields
    * Capture floating point values and negative numeric values
  * Window handling improvements:
    * Do not create too large intial window when running on windows
    * Do not exit fullscreen on focus loss

## Version 4.13.3
* Picturall: Better parsers for the messages
  * Previous parsers didn't cope with filenames with spaces (among other things)
  * Rewrote the parsers using regexp-vomit
  * Worked around a picturall bug where media collection messages have unbalanced " in them
* Picturall: Improve looping media filtering
  * Starting and stopping looping media should no longer produce temporary readouts on the clock
* Picturall: Try to get media collection & default_play_mode via telnet
  * This is currently broken in the telnet protocol but might eventually be fixed by AW
  * If we are able to get that data via a command use it instead of the save_media_db hack

## Version 4.13.2
* Picturall improvements and fixes:
  * Autodiscovery kept socket open after returning
  * Fetch the media library and check DefaultPlayMode for looping
    * The media library fetch is bit iffy, once a better way is available it will be migrated

## Version 4.13.1
* Bugfix: picturall autodiscovery on linux and windows

## Version 4.13.0
* Added support for Analog Way Picturall media servers

## Version 4.12.3
* Bugfixes: Some web config values weren't validated as intended
* Added proper ifupdown support in the raspberry pi image
  * This allows setting static dns servers in interfaces file.

## Version 4.12.2
* New project url. The project had to be changed to a gitlab namespace due to OSS project lisencing changes in gitlab.
* Image: Improved the detection of capture interfaces for LTC

## Version 4.12.1
* Bugfixes:
  * Fix linter errors

## Version 4.12.0
* Features:
  * Option to display timer end time. Hint: Assign same timer to two sources to show both remaining and target time
* Bugfixes:
  * Timer feedback when counting up from zero
  * OSC library has been updated

## Version 4.11.4
* Bugfixes:
  * Limitimer: Improve handling of communication errors
  * Limitimer: "Compact" output was out of sync with main output on some cases
  * HTTP: Fix typo in the html for limitimer settings block

## Version 4.11.3
* Bugfixes:
  * Limitimer: Fix count-after-zero skipping 0

## Version 4.11.2
* Bugfixes:
  * Another bug in the limitimer broadcasting feature
  * Another upgrade on go-osc to deal with some bugs that were still present

## Version 4.11.1
* Bugfixes:
  * Limitimer state OSC broadcasting was not working


## Version 4.11.0
* Features:
  * Reloading (most) config options doesn't require a restart anymore
    * HTTP option changes aren't applied until the clock has been restarted
    * Expect there to be still some issues
  * The OSC communication library has been updated for a better performing implementation. There might be some issues with OSC communication
  * Limitimer timers can now be rebroadcast via OSC to other clocks
  * Limitimer timer reception from RS485 can be toggled on and off per limitimer program
* Bugfixes:
  * Flush the config file changes straight to disc for extra robustness
  * Support for older limitimer controllers that don't send message checksums


## Version 4.10.1
* Bugfixes:
  * Missing default on limitimer serial device
  * Typo in limitimer help link in config page (user visible link text only)

## Version 4.10.0
* Features:
  * Limitimer integration ###
    * See `limitimer.md` for usage.
    * Caveats:
      * Companion integration needs version 5.2.2 of the module for the feedback to work
      * Updating the clock config might cause loss-of-signal from limitimer until the raspberry pi is rebooted, investigation ongoing


## Version 4.9.0
* Features:
  * Ability to start in full screen mode on windows / osx via config option
* Bugfixes:
  * Round clock position and scaling with rotated displays

## Version 4.8.0
* Features:
  * Countdown face can now also count up
  * Countdown face displays years to/from target is non-zero
* Bugfixes:
  * SDL texture handling fixes to clear assertions

## Version 4.7.0
Dedicated to Manda and Ilmaleipuri. You were the best cats and companions. I will forever miss you.

* Features:
  * Audio cue support for expiring countdowns
  * Build windows binaries automatically
  * For windows and osx there are several keybindings:
    * Press F to go full screen
    * Press escape to exit full screen
    * Press C to open the web configuration
  * /clock/automation OSC message for controlling signal color automation
  * Config option to use the selected hardware signal group color as clock background color
  * Countdown clock face for countdowns to static date and time
* Bugfixes:
  * Stopping a timer with signal color automation now clears the color


## Version 4.6.0
* Features:
  * 144x144px round clock face
  * 288x144px text clock face
  * Improved signal color handling, You can now override the automation set colors via osc commands
    * If enabled, automation does one color change when timer thresholds are reached, but doesn't prevent manually setting the signal colors.


## Version 4.5.2
* Bugfix: /clock/signal/* OSC command was expecting 4 parameters

## Version 4.5.1
* Bugfixes:
  * Config generation for overtime colors
  * Round clock and overtime visibility modes

## Version 4.5.0
* New: Customizable overtime behaviour
  * Handling of overtime countdowns can now be customized
  * Readout can be chosen to be either: show zeros, show blank, continue counting up
  * The extra visibility can be chosen to be: none, blink, change background color or change both background and blink

## Version 4.4.1
* Bugfix: Load the SPI kernel modules in the clock image for unicorn hats

## Version 4.4.0
* New: Initial hardware signal light support
  * Currently only pimoroni Unicorn HD and Ubercorn hats are supported
  * You can control the color via OSC by groups, or the light color can follow source 1

## Version 4.3.1
* Bugfix for the signal colors

## Version 4.3.0
* New: Color signals for timers
  * `/clock/timer/*/signal` OSC command
  * Optional automation for the signal colors based on timer thresholds

## Version 4.2.0
* New:
  * /clock/flash command, it flashes the screen white for 200ms

## Version 4.1.1
* Fix version information stamping in build automation

## Version 4.1.0
* New:
  * Support for hyperpixel4 square displays
  * Documentation for the displays in README.md

## Version 4.0.3
* Bugfixes:
  * Unhide sources when starting a targeted count

## Version 4.0.2
* Bugfixes:
  * Hide behaviour of the different osc commands to be uniform
  * Starting a timer to given target while clock is paused

## Version 4.0.1
* Fixed hyperpixel4-square configuration

## Version 4.0.0
### New feature highlights

* Support for different clock faces
  * Single round clock
  * Dual round clocks
  * Text clocks
    * 3 and 1 clocks per screen
* New concept of sources and timers
  * A clock face displays a clock source, which can be associated to a timer and different sources for the time, eg. LTC
  * Timers can be used on multiple clock setups
* HTTP configuration editor with input validation
  * import / export for the whole configuration file
* New OSC commands, see osc.md
  * New companion module version supporting the V4 commands and feedback
  * Background selection
  * Timers targeting a certain time-of-day
  * Sending of text messages with custom text and background colors
  * Setting of clock source labels
* Info overlay shown on startup containing version, ip and port information
* Support of LTC input via usb-soundcard / hifiberry dac+ adc pro hat
* Mitti and millumin message processing completely built in
  * You can choose which timers are the destinations of the playback state
* Support for sending and receiving timers as Interspace Industries Countdown2 UDP messages for display units
  * Also supported by StageTimer2 and Irisdown

### Breaking changes

* clock configuration file contents have been changed.
* OSC commands have been changed, compatibility with v3 commands has been kept as close as possible
  * V3 commands will be dropped at a later date
* OSC feedback has been changed
* Command line arguments have been changed
* New dependencies; sdl-ttf, sdl-image, ttf fonts for text rendering

## Version 3.16.3
* BUGFIX: Do not unpause counters when started

## Version 3.16.2
* BUGFIX: Timer pause, resume and turning clock off
* BUGFIX: /clock/countup/modify direction flip to be more natural

## Version 3.16.1
* Fix a linter error preventing automatic builds

## Version 3.16.0
* Start refactoring internal clock engine to implement new clock faces
* Add /clock/countup/modify OSC command
* Add /boot/config.txt editing to the web configuration

## Version 3.15.0
* Add experimental support for Interspace Industries USB AiO soundcard for LTC to the generated images

## Version 3.14.1
* BUGFIX: Clock scaling on the official 7" display

## Version 3.14.0
* Experimental background image support

## Version 3.13.0
* Add an option to disable detection and aspect ratio correction of official raspberry pi displays
* BUGFIX: setting 12h format from the http interface didn't work
* Internal refactoring and code cleanup

## Version 3.12.1
* BUGFIX: missing colon from default clock.ini for port 80

## Version 3.12.0
* Add support for 12 hour format on time-of-day display

## Version 3.11.1
* LTC Bugfix, we had a wrong audio device in the default configuration

## Version 3.11.0
* LTC added options:
  * Toggle between frames and seconds on the clock ring
  * Toggle for loss of signal handling; either blank the clock or continue on internal time
  * Toggle for disabling the LTC handling
* Images
  * Build raspberry pi 2 compatible images
  * Use config file generated by the build automation

## Version 3.10.2
* LTC reception & Hifiberry ADC+ DAC Pro
  * Bugfix on sdl-clock to display the hour part of LTC timestamp
  * Added configuration for the hifiberry to the sd-card image

## Version 3.10.1
* Little more black space between dual clock text display "pixels"


## Version 3.10.0
* Add /clock/dual/text OSC message for setting a text field on dual clock mode

## Version 3.9.0
* Build automation:
  * Build one unified image
* Clock:
  * Web configuration interface
  * Dual-clock mode

## Version 3.8.2
* Build automation improvements
  * Read version information from runtime/debug.ReadBuildInfo()
  * Include pimoroni hyperpixel4 square display support

## Version 3.8.0
* multi-clock: Implement new display of up to 40 clocks
* config file support: use -C to read config flags from a file
* osx support: Remove unused rpi gpio stuff from sdl-clock and handle sdl2 events

## Version 3.7.1
* clock-bridge:
  * BUGFIX: Millumin osc tally and layer renaming. Renaming an active layer caused the internal state to become corrupt. The clock-bridge now filters layer states that haven't been updated in 1 second to remove ghost layers from the renames.

## Version 3.7.0
* BUGFIX: Correct the monitor pixel aspect ratio on the official raspberry pi display. The clock is now round instead of oval.
* FEATURE: Detect if a monitor is rotated 90 or 270 degrees and move the clock to the top portion of the screen.

## Version 3.6.2
* BUGFIX: Make /clock/time/set timezone aware.

## Version 3.6.1
* BUGFIX: /clock/time/set OSC command and busybox date command.

## Version 3.6.0
* Adds /clock/time/set osc command for setting system time, see README.md for details

## Version 3.5.1
* BUFIX: build issue on 3.5.0

## Version 3.5.0
* OSC commands for hiding/showing the numerical seconds field
  * /clock/seconds/off - hides the second display
  * /clock/seconds/on - shows the second display
* Add --debug command line parameter to enable verbose logging

## Version 3.4.1
* Greatly reduced the amount of logging that gets printed on the console. This caused slowdowns on raspberry pi

## Version 3.4.0
* Automatically rescan network interfaces and addresses every 5 seconds
  * This removes the race condition with dhcp lease on startup
  * Now if feedback address is 255.255.255.255 the feedback is sent to broadcast on all configured ipv4 interfaces
* Added --disable-feedback command line switch
  * This disables the osc feedback but the clock will still listen for osc commands

## Version 3.3.0
* Add --disable-osc command line parameter

## Version 3.2.0
* Implemented resolution scaling.
  * This adds support for the official raspberry pi 7" display
* Fixes to the 192x192 small mode

## Version 3.1.1rc1
* CI environment now builds raspberry pi sd card images

## Version 3.1.0
* Clock now shows its version number before acquiring the correct time

## Version 3.0.2
* BUGIX: "tally" text formatting for /qmsk/clock/count osc messages

## Version 3.0.1
* Implement CI environment for testing and automated builds

## Version 3.0.0
* Breaking change to OSC feedback, clocks now send paused/running state
* Countdowns can now be paused and resumed with /clock/pause and /clock/resume

## Version 2.1.0
* Disable flashing when countdown reaches zero with --flash=0 parameter
* Print version to console when started

## Version 2.0.0
* Added countdown and count up timer modes
* Added OSC commands for timers
* Added OSC feedback of clock mode and display date
