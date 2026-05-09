# Buildroot external package includes.
# Skip package makefiles when the base Buildroot tree already provides the same
# package directory name, such as python-smbus2 in Buildroot 2025.11.
CLOCK8002_EXTERNAL_MKS = $(sort $(wildcard $(BR2_EXTERNAL_CLOCK8002_PATH)/package/*/*.mk))
CLOCK8002_EXTERNAL_MKS := $(foreach mk,$(CLOCK8002_EXTERNAL_MKS), \
	$(if $(wildcard $(TOPDIR)/package/$(notdir $(patsubst %/,%,$(dir $(mk))))),, \
	$(mk)))

include $(CLOCK8002_EXTERNAL_MKS)
