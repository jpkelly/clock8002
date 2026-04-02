################################################################################
#
# python-luma-oled
#
################################################################################

PYTHON_LUMA_OLED_VERSION = 3.15.0
PYTHON_LUMA_OLED_SOURCE = luma_oled-$(PYTHON_LUMA_OLED_VERSION).tar.gz
PYTHON_LUMA_OLED_SITE = https://files.pythonhosted.org/packages/f9/36/cad8c85b0206ffbbbb7d2609fdb376666521a503837b6e853a4600f09d5f
PYTHON_LUMA_OLED_SETUP_TYPE = setuptools
PYTHON_LUMA_OLED_LICENSE = MIT
PYTHON_LUMA_OLED_LICENSE_FILES = LICENSE.rst
PYTHON_LUMA_OLED_DEPENDENCIES = python-luma-core

$(eval $(python-package))
