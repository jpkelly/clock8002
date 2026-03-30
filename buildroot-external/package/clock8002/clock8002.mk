################################################################################
# clock8002 package prototype
################################################################################

CLOCK8002_VERSION = prototype
CLOCK8002_SITE = $(call qstrip,$(BR2_PACKAGE_CLOCK8002_SOURCE_DIR))
CLOCK8002_SITE_METHOD = local
CLOCK8002_LICENSE = GPL-3.0
CLOCK8002_LICENSE_FILES = LICENSE

# Prototype install-only package. Build integration for sdl-clock/alsa-ltc
# should be added once toolchain/runtime dependency decisions are finalized.
define CLOCK8002_INSTALL_TARGET_CMDS
	$(INSTALL) -d $(TARGET_DIR)/opt/clock8002
	if [ -f $(@D)/sdl-clock ]; then \
		$(INSTALL) -m 0755 $(@D)/sdl-clock $(TARGET_DIR)/opt/clock8002/sdl-clock; \
	fi
	if [ -f $(@D)/alsa-ltc ]; then \
		$(INSTALL) -m 0755 $(@D)/alsa-ltc $(TARGET_DIR)/opt/clock8002/alsa-ltc; \
	fi
	if [ -f $(@D)/clock8002.service ]; then \
		$(INSTALL) -D -m 0644 $(@D)/clock8002.service \
			$(TARGET_DIR)/usr/lib/systemd/system/clock8002.service; \
	fi
	if [ -f $(@D)/alsa-ltc.service ]; then \
		$(INSTALL) -D -m 0644 $(@D)/alsa-ltc.service \
			$(TARGET_DIR)/usr/lib/systemd/system/alsa-ltc.service; \
	fi
endef

$(eval $(generic-package))
