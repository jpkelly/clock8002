################################################################################
# clock8002 application package
################################################################################

CLOCK8002_VERSION = prototype
CLOCK8002_SITE = $(call qstrip,$(BR2_PACKAGE_CLOCK8002_SOURCE_DIR))
CLOCK8002_SITE_METHOD = local
CLOCK8002_LICENSE = GPL-3.0
CLOCK8002_LICENSE_FILES = LICENSE
CLOCK8002_DEPENDENCIES = host-go sdl2 sdl2_gfx sdl2_image sdl2_mixer sdl2_ttf libltc

define CLOCK8002_BUILD_CMDS
	cd $(@D) && \
		$(HOST_GO_TARGET_ENV) \
		CGO_ENABLED=1 \
		GOFLAGS=-mod=vendor \
		$(HOST_DIR)/bin/go build \
			-v \
			-ldflags "-extldflags '-fuse-ld=bfd'" \
			-o $(@D)/sdl-clock \
			./cmd/sdl-clock
	$(TARGET_CC) -O2 \
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
	if [ -f $(@D)/oled/oled.ini ]; then \
		$(INSTALL) -D -m 0644 $(@D)/oled/oled.ini \
			$(TARGET_DIR)/boot/piclock/oled.ini; \
	fi
endef

$(eval $(generic-package))
