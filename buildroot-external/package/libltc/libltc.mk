################################################################################
# libltc
################################################################################

LIBLTC_VERSION = 1.3.2
LIBLTC_SITE = https://github.com/x42/libltc/releases/download/v$(LIBLTC_VERSION)
LIBLTC_SOURCE = libltc-$(LIBLTC_VERSION).tar.gz
LIBLTC_LICENSE = LGPL-3.0+
LIBLTC_LICENSE_FILES = COPYING
LIBLTC_INSTALL_STAGING = YES

$(eval $(autotools-package))
