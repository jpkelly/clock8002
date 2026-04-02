################################################################################
# rpi-eeprom – Raspberry Pi bootloader EEPROM tools
################################################################################

RPI_EEPROM_VERSION = v2025.12.08-2712
RPI_EEPROM_SITE = https://github.com/raspberrypi/rpi-eeprom
RPI_EEPROM_SITE_METHOD = git
RPI_EEPROM_GIT_SUBMODULES = NO
RPI_EEPROM_LICENSE = BSD-3-Clause (tools), Raspberry Pi Ltd (firmware blobs)
RPI_EEPROM_LICENSE_FILES = LICENSE

RPI_EEPROM_DEPENDENCIES = python3

define RPI_EEPROM_INSTALL_TARGET_CMDS
	# Install the three main tools
	$(INSTALL) -D -m 0755 $(@D)/rpi-eeprom-config \
		$(TARGET_DIR)/usr/bin/rpi-eeprom-config
	$(INSTALL) -D -m 0755 $(@D)/rpi-eeprom-update \
		$(TARGET_DIR)/usr/bin/rpi-eeprom-update
	$(INSTALL) -D -m 0755 $(@D)/rpi-eeprom-digest \
		$(TARGET_DIR)/usr/bin/rpi-eeprom-digest

	# Install default config for rpi-eeprom-update
	$(INSTALL) -D -m 0644 $(@D)/rpi-eeprom-update-default \
		$(TARGET_DIR)/etc/default/rpi-eeprom-update

	# Install Pi 5 (BCM2712) firmware blobs – default channel only
	$(INSTALL) -d $(TARGET_DIR)/usr/lib/firmware/raspberrypi/bootloader-2712/default
	cp -a $(@D)/firmware-2712/default/* \
		$(TARGET_DIR)/usr/lib/firmware/raspberrypi/bootloader-2712/default/
endef

$(eval $(generic-package))
