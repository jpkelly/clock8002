################################################################################
# clock8002 application package
################################################################################

CLOCK8002_VERSION = prototype
CLOCK8002_SITE = $(call qstrip,$(BR2_PACKAGE_CLOCK8002_SOURCE_DIR))
CLOCK8002_SITE_METHOD = local
CLOCK8002_LICENSE = GPL-3.0
CLOCK8002_LICENSE_FILES = LICENSE
CLOCK8002_DEPENDENCIES = host-go sdl3 sdl3-ttf sdl3-image libltc

CLOCK8002_CONFIG_SUFFIX = default

CLOCK8002_GIT_DIR = $(realpath $(CLOCK8002_SITE)/..)
# The root-ram feature line is intentionally labeled as ram-root in runtime UI/config.
CLOCK8002_GIT_TAG = $(shell cd $(CLOCK8002_GIT_DIR) && branch=$$(git branch --show-current 2>/dev/null || true); if [ "$$branch" = "feature/root-ram" ]; then echo "ram-root"; else git describe --tags --abbrev=0 HEAD 2>/dev/null || echo "v0.0.1"; fi)
CLOCK8002_GIT_COMMIT = $(shell cd $(CLOCK8002_GIT_DIR) && git rev-list -1 HEAD 2>/dev/null || echo "unknown")
CLOCK8002_BUILD_DATE = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
CLOCK8002_VERSION_PKG = gitlab.com/clock-8001/clock-8001/v4/clock
CLOCK8002_GO_LD_FLAGS = -X $(CLOCK8002_VERSION_PKG).gitCommit=$(CLOCK8002_GIT_COMMIT) \
	-X $(CLOCK8002_VERSION_PKG).gitTag=$(CLOCK8002_GIT_TAG) \
	-X $(CLOCK8002_VERSION_PKG).buildEnvironment=buildroot
CLOCK8002_ALSA_LTC_CFLAGS = -O2 \
	-DALSA_LTC_GIT_TAG=\"$(CLOCK8002_GIT_TAG)\" \
	-DALSA_LTC_GIT_COMMIT=\"$(CLOCK8002_GIT_COMMIT)\" \
	-DALSA_LTC_BUILD_DATE=\"$(CLOCK8002_BUILD_DATE)\"

define CLOCK8002_BUILD_CMDS
	cd $(@D) && \
		$(HOST_GO_TARGET_ENV) \
		CGO_ENABLED=0 \
		GOFLAGS=-mod=vendor \
		$(HOST_DIR)/bin/go build \
			-v \
			-ldflags "$(CLOCK8002_GO_LD_FLAGS)" \
			-o $(@D)/sdl3-clock \
			./cmd/sdl3-clock
	$(TARGET_CC) $(CLOCK8002_ALSA_LTC_CFLAGS) \
		-I$(STAGING_DIR)/usr/include \
		-o $(@D)/alsa-ltc $(@D)/alsa-ltc.c \
		-L$(STAGING_DIR)/usr/lib -lasound -lltc
	# Keep the legacy deployment name expected by the .244-style boot scripts,
	# but source it from the SDL3 build rather than reviving the old SDL2 binary.
	cp -f $(@D)/sdl3-clock $(@D)/sdl-clock
	# Build oled-daemon frozen binary via PyInstaller (uses host venv on cm5).
	if [ -f $(@D)/oled/oled_daemon.py ] && [ -x /home/pi/oled-build-venv/bin/pyinstaller ]; then \
		/home/pi/oled-build-venv/bin/pyinstaller \
			--onefile \
			--name oled-daemon \
			--hidden-import luma.core.interface.serial \
			--hidden-import luma.core.mixin \
			--hidden-import luma.oled.device \
			--hidden-import smbus2 \
			--distpath $(@D)/oled \
			--workpath /tmp/oled-pyinstaller-build \
			--specpath /tmp \
			$(@D)/oled/oled_daemon.py \
			> /tmp/oled-pyinstaller.log 2>&1; \
	fi
endef

define CLOCK8002_INSTALL_TARGET_CMDS
	$(INSTALL) -d $(TARGET_DIR)/opt/clock8002
	$(INSTALL) -m 0755 $(@D)/sdl-clock $(TARGET_DIR)/opt/clock8002/sdl-clock
	$(INSTALL) -m 0755 $(@D)/sdl3-clock $(TARGET_DIR)/opt/clock8002/sdl3-clock
	$(INSTALL) -m 0755 $(@D)/alsa-ltc $(TARGET_DIR)/opt/clock8002/alsa-ltc
	$(INSTALL) -D -m 0644 $(@D)/clock.ini.$(CLOCK8002_CONFIG_SUFFIX) \
		$(TARGET_DIR)/boot/piclock/clock.ini
	$(INSTALL) -D -m 0644 $(@D)/network.ini.$(CLOCK8002_CONFIG_SUFFIX) \
		$(TARGET_DIR)/boot/piclock/network.ini
	cp -a $(@D)/fonts $(TARGET_DIR)/opt/clock8002/
	cp -a $(@D)/voices $(TARGET_DIR)/opt/clock8002/
	$(INSTALL) -m 0644 $(@D)/ttf_fonts/*.ttf $(TARGET_DIR)/opt/clock8002/
	if [ -f $(@D)/oled/oled_daemon.py ]; then \
		$(INSTALL) -D -m 0755 $(@D)/oled/oled_daemon.py \
			$(TARGET_DIR)/opt/clock8002/oled/oled_daemon.py; \
	fi
	if [ -f $(@D)/oled/oled-daemon ]; then \
		$(INSTALL) -D -m 0755 $(@D)/oled/oled-daemon \
			$(TARGET_DIR)/opt/clock8002/oled/oled-daemon; \
	fi
	if [ -f $(@D)/oled/piclockLogo.bin ]; then \
		$(INSTALL) -D -m 0644 $(@D)/oled/piclockLogo.bin \
			$(TARGET_DIR)/opt/clock8002/piclockLogo.bin; \
	fi
	if [ -f $(@D)/oled/oled.ini ]; then \
		$(INSTALL) -D -m 0644 $(@D)/oled/oled.ini \
			$(TARGET_DIR)/boot/piclock/oled.ini; \
	fi
	$(INSTALL) -D -m 0644 $(@D)/splash/bootsplash.png \
		$(TARGET_DIR)/opt/clock8002/bootsplash.png
	$(INSTALL) -D -m 0755 $(@D)/splash/bootsplash.sh \
		$(TARGET_DIR)/opt/clock8002/bootsplash.sh
endef

$(eval $(generic-package))
