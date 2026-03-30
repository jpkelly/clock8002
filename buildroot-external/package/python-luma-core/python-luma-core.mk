################################################################################
#
# python-luma-core
#
################################################################################

PYTHON_LUMA_CORE_VERSION = 2.5.3
PYTHON_LUMA_CORE_SOURCE = luma_core-$(PYTHON_LUMA_CORE_VERSION).tar.gz
PYTHON_LUMA_CORE_SITE = https://files.pythonhosted.org/packages/e6/a3/0abb456daf2279483579bed6cf2a7305f93f56ab89f0f238f206fffce303
PYTHON_LUMA_CORE_SETUP_TYPE = setuptools
PYTHON_LUMA_CORE_LICENSE = MIT
PYTHON_LUMA_CORE_LICENSE_FILES = LICENSE.rst
PYTHON_LUMA_CORE_DEPENDENCIES = python-cbor2 python-pillow python-smbus2

$(eval $(python-package))
