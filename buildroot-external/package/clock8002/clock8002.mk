################################################################################
# clock8002 application package
################################################################################

CLOCK8002_VERSION = prototype
CLOCK8002_SITE = $(call qstrip,$(BR2_PACKAGE_CLOCK8002_SOURCE_DIR))
CLOCK8002_SITE_METHOD = local
CLOCK8002_LICENSE = GPL-3.0
CLOCK8002_LICENSE_FILES = LICENSE
CLOCK8002_DEPENDENCIES = host-go sdl2 sdl2_gfx sdl2_image sdl2_mixer sdl2_ttf libltc

CLOCK8002_GIT_DIR = $(realpath $(CLOCK8002_SITE)/..)
CLOCK8002_GIT_TAG = $(shell cd $(CLOCK8002_GIT_DIR) && git describe --tags --abbrev=0 HEAD 2>/dev/null || echo "v0.0.1")
CLOCK8002_GIT_COMMIT = $(shell cd $(CLOCK8002_GIT_DIR) && git rev-list -1 HEAD 2>/dev/null || echo "unknown")
CLOCK8002_BUILD_DATE = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
CLOCK8002_VERSION_PKG = gitlab.com/clock-8001/clock-8001/v4/clock
CLOCK8002_GO_LD_FLAGS = -X $(CLOCK8002_VERSION_PKG).gitCommit=$(CLOCK8002_GIT_COMMIT) \
	-X $(CLOCK8002_VERSION_PKG).gitTag=$(CLOCK8002_GIT_TAG) \
	-X $(CLOCK8002_VERSION_PKG).buildEnvironment=buildroot \
	-extldflags '-fuse-ld=bfd'
CLOCK8002_ALSA_LTC_CFLAGS = -O2 \
	-DALSA_LTC_GIT_TAG=\"$(CLOCK8002_GIT_TAG)\" \
	-DALSA_LTC_GIT_COMMIT=\"$(CLOCK8002_GIT_COMMIT)\" \
	-DALSA_LTC_BUILD_DATE=\"$(CLOCK8002_BUILD_DATE)\"

define CLOCK8002_BUILD_CMDS
	cd $(@D) && \
		$(HOST_GO_TARGET_ENV) \
		CGO_ENABLED=1 \
		GOFLAGS=-mod=vendor \
		PKG_CONFIG=$(HOST_DIR)/bin/pkg-config \
		$(HOST_DIR)/bin/go build \
			-v \
			-ldflags "$(CLOCK8002_GO_LD_FLAGS)" \
			-o $(@D)/sdl-clock \
			./cmd/sdl-clock
	$(TARGET_CC) $(CLOCK8002_ALSA_LTC_CFLAGS) \
		-I$(STAGING_DIR)/usr/include \
		-o $(@D)/alsa-ltc $(@D)/alsa-ltc.c \
		-L$(STAGING_DIR)/usr/lib -lasound -lltc
endef

define CLOCK8002_INSTALL_TARGET_CMDS
	$(INSTALL) -d $(TARGET_DIR)/opt/clock8002
	$(INSTALL) -m 0755 $(@D)/sdl-clock $(TARGET_DIR)/opt/clock8002/sdl-clock
	$(INSTALL) -m 0755 $(@D)/alsa-ltc $(TARGET_DIR)/opt/clock8002/alsa-ltc
	$(INSTALL) -D -m 0644 $(@D)/clock8002.service \
		$(TARGET_DIR)/usr/lib/systemd/system/clock8002.service
	$(INSTALL) -D -m 0644 $(@D)/alsa-ltc.service \
		$(TARGET_DIR)/usr/lib/systemd/system/alsa-ltc.service
	$(INSTALL) -D -m 0644 $(@D)/clock.ini.default \
		$(TARGET_DIR)/boot/piclock/clock.ini
	$(INSTALL) -D -m 0644 $(@D)/network.ini.default \
		$(TARGET_DIR)/boot/piclock/network.ini
	cp -a $(@D)/fonts $(TARGET_DIR)/opt/clock8002/
	cp -a $(@D)/voices $(TARGET_DIR)/opt/clock8002/
	$(INSTALL) -m 0644 $(@D)/ttf_fonts/*.ttf $(TARGET_DIR)/opt/clock8002/
	if [ -f $(@D)/oled/oled_daemon.py ]; then \
		$(INSTALL) -D -m 0755 $(@D)/oled/oled_daemon.py \
			$(TARGET_DIR)/opt/clock8002/oled/oled_daemon.py; \
	fi
	if [ -f $(@D)/oled/oled_daemon.service ]; then \
		$(INSTALL) -D -m 0644 $(@D)/oled/oled_daemon.service \
			$(TARGET_DIR)/usr/lib/systemd/system/oled_daemon.service; \
	fi
	if [ -f $(@D)/oled/piclockLogo.bin ]; then \
		$(INSTALL) -D -m 0644 $(@D)/oled/piclockLogo.bin \
			$(TARGET_DIR)/root/piclockLogo.bin; \
	fi
	if [ -f $(@D)/oled/oled.ini ]; then \
		$(INSTALL) -D -m 0644 $(@D)/oled/oled.ini \
			$(TARGET_DIR)/boot/piclock/oled.ini; \
	fi
	$(INSTALL) -D -m 0644 $(@D)/splash/bootsplash.png \
		$(TARGET_DIR)/opt/clock8002/bootsplash.png
	$(INSTALL) -D -m 0755 $(@D)/splash/bootsplash.sh \
		$(TARGET_DIR)/opt/clock8002/bootsplash.sh
	$(INSTALL) -D -m 0644 $(@D)/splash/bootsplash-fbv.service \
		$(TARGET_DIR)/usr/lib/systemd/system/bootsplash.service
endef

$(eval $(generic-package))
