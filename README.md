# OSC controlled simple clock
* Clock binary builds: [![pipeline status](https://gitlab.com/clock-8001/clock-8001/badges/master/pipeline.svg)](https://gitlab.com/clock-8001/clock-8001/commits/master)
* The installation files are available on https://kissa.depili.fi/clock-8001/releases/

Support clock-8001 development by paypal: [![](https://www.paypalobjects.com/en_US/i/btn/btn_donate_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=XUMXUL5RX5MWJ&currency_code=EUR)


## Repository name change 2022-06-02

Due to changes to gitlab OSS licensing the project had to be moved to a new location inside a gitlab clock-8001 organization. So this is the new home for the project, https://gitlab.com/clock-8001/clock-8001/ Unfortunately this also means that go projects importing the clock packages need to be updated to reflect on the new module location and name. Sorry about that, but it had to be done.

## About

This is a simplistic clock written in go that can be used either as a video out with SDL or as a dedicated clock buidt with a 32x32 pixel hub75 led matrix and a ring of 60 addressable leds.

The README has been written for the version 4.x.x clocks, ie. the gitlab.com/clock-8001/clock-8001/v4 go module.

The clock can be controlled with the depili-clock-8001 companion module: https://github.com/bitfocus/companion-module-depili-clock-8001 Module version 5.0.0 is the first to implement V4 clock API.

Features and configuration in greated detail can be found in the [getting started guide in wiki](https://gitlab.com/clock-8001/clock-8001/-/wikis/Getting-started-on-clock-8001-version-4).

Developed in co-operation with Daniel Richert and with grants from FUUG - Finnish Unix User Group.

## Platforms supported

The main target is a custom hardened linux image for raspberry pi 2 & 3 single board computers. It has been made to be as fault tolerant as possible.

Currently the clock is also available on macos Vetura, linux and windows as precompiled installation packages. Compiling from the source should be possible on many other variants.

The installation files are available on https://kissa.depili.fi/clock-8001/releases/

* .msi files are Microsoft installer files for Windows, tested on windows 10 64bit
* .deb files are debian / ubuntu linux packages, currently x86 64bit (amd64) and armhf (raspberry pi os 32bit) packages are being generated. Packages have been tested on raspberry pi os bullseye on rpi 3b+ and rpi4 and on ubuntu 22.4 jammy 64bit.
  * Best way to install the package is to use 'sudo apt install ./clock-8001_1.2.3_arch.deb' command, as it will automatically pull all the dependencies.
* .zip files contain Macos .app files. The earlier unsigned intel releases should work on big sur onwards on x86 macs, the later signed universal releases are for ventura.

## Web configuration interface

The new unified images have a web configuration interface for the clock settings. You can access this interface by pointing your browser to the address of the clock. The default username is `admin` and the default password is `clockwork`. You should change them from the interface or the clock.ini file.

On the raspberry pi images the config interface runs on port 80, on the installable versions on macos, linux and windows the default port is 8080.

You can open a browser pointing to the configuration interface by pressing 'c' key while the clock is running.

## Ready made raspberry pi images

SD-card images for raspberry pi can be found at https://kissa.depili.fi/clock-8001/images

Clock-8001 no longer has multiple different images, they have all been consolidated to one unified image which uses `enable_` files to activate various parts of the clock system as desired. The new images are named as `clock-800-unified-<version>.img`

The images support raspberry pi 2B / 3B / 3B+ boards. They need at least 64Mb SD-cards. Write them to the card like any other raspberry pi sd-card image.

The image tries to get a dhcp address on wired ethernet and also brings up a virtual interface eth0:1 with static ip (default 192.168.10.245 with 255.255.255.0 netmask).

### Customizing the images

You can place the following files on the sd-card FAT partition to customize the installation:
* `clock.ini` main clock configuration file that is used by the default `clock_cmd.sh`
* `hostname` to change the hostname used by the clock, it is available with "hostname.local" for bonjour / mDNS requests
* `interfaces` a replacement for /etc/network/interfaces for custom network configuration
* `ntp.conf` for custom ntp server configuration
* `config.sys` the normal raspberry pi boot configuration for changing video modes etc.
* `sdl-clock` to update the clock binary with this file
* `clock_cmd.sh` is the command line for the clock, it should start with `/root/sdl-clock ` and be followed by any command line parameters you wish to use for the clock.
* `clock_bridge` to update the clock bridge binary file
* `clock_bridge_cmd.sh` to update the clock bridge command line. It should start with `/root/clock-bridge` and be followed by any command line paramaters for the bridge.
* `enable_clock` delete this file and the main clock will not be active
* `enable_bridge` delete this file and the mitti / millumin osc bridge will not be active
* `enable_ssh` delete this file and remote ssh logins to the raspberry pi will not be allowed
* `enable_ltc` delete this file and the LTC audio -> OSC functionality will not be active

### Hyperpixel4 displays

The image supports both the retangular and square hyperpixel4 displays from Pimoroni. To use them you need to rename either `config.txt.hp4_square` or `config.txt.hp4_rect` to `config.txt`

For the rectangular display you need to also modify `clock_cmd.sh` to contain:
```
/hyperpixel4_rect/hyperpixel4-init
/root/sdl-clock -C /boot/clock.ini
```
So that the display is initialized on boot.

## sdl-clock - Output the clock to hdmi on the raspberry pi

You can build the clock binary with `go get gitlab.com/clock-8001/clock-8001/v4/cmd/sdl_clock`. Compiling requires SDL 2 and SDL_GFX 2 libraries. On the raspberry pi the default libraries shipped with rasbian will only output data to X11 window, so for full screen dedicated clock you need to compile the SDL libraries from source. For compiling use `./configure --host=armv7l-raspberry-linux-gnueabihf --disable-pulseaudio --disable-esd --disable-video-mir --disable-video-wayland --disable-video-x11 --disable-video-opengl` for config flags.

### Precompiled binaries

* Latest from git master: [sdl-clock](https://gitlab.com/clock-8001/clock-8001/-/jobs/artifacts/master/raw/sdl-clock?job=build)
* Testing builds: https://kissa.depili.fi/clock-8001/testing/
* Tagged releases: https://kissa.depili.fi/clock-8001/releases/

### LTC timecode support

The clock can be used to display a SMPTE LTC time. This requires a Hifiberry ADC+ DAC Pro hat: https://www.hifiberry.com/shop/boards/hifiberry-dac-adc-pro/ or Interpace Industries USB AiO interface. Currently only these two options for audio input are supported.

For Hifiberry it is recommended to use the pin headers for balanced audio input and to wire the incoming mono LTC signal to both left and right channels.

Other usb audio interfaces might work, but they are completely untested and no support is offered for them.

## matrix-clock - Dedicated led matrix clock

**The matrix clock v4 isn't currently available. You can still use the version 3.**

Bill of materials:
* Raspberry pi
* 32x32 pixel 4mm pixel pitch led matrix
* Led ring with 60 ws2812b leds
* Arduino (nano recommend)
* Adapter hat for the raspberry pi to connect to the led matrix
* 5V 3A power supply
* 12 leds of your choice for the static "hour" markers and current limiting resistors

You need to compile https://gitlab.com/Depili/rpi-matrix for a small program that will listen on udp socket for the led matrix data and handle driving the led matrix.

Compile the led matrix clock binary with `go get gitlab.com/clock-8001/clock-8001/v3/cmd/matrix-clock`

### Precompiled binaries

* Latest from git master: [matrix-clock](https://gitlab.com/clock-8001/clock-8001/-/jobs/artifacts/master/raw/matrix-clock?job=build)

## OSC commands understood by the clock

See https://gitlab.com/clock-8001/clock-8001/-/blob/master/v4/osc.md
