################################################################################
# sdl3
################################################################################

SDL3_VERSION = 3.4.0
SDL3_SITE = https://github.com/libsdl-org/SDL/releases/download/release-$(SDL3_VERSION)
SDL3_SOURCE = SDL3-$(SDL3_VERSION).tar.gz
SDL3_LICENSE = Zlib
SDL3_LICENSE_FILES = LICENSE.txt
SDL3_INSTALL_STAGING = YES

SDL3_DEPENDENCIES = libdrm mesa3d eudev

SDL3_CONF_OPTS = \
	-DSDL_SHARED=ON \
	-DSDL_STATIC=OFF \
	-DSDL_TESTS=OFF \
	-DSDL_EXAMPLES=OFF \
	-DSDL_INSTALL_TESTS=OFF \
	-DSDL_UNIX_CONSOLE_BUILD=ON \
	-DSDL_KMSDRM=ON \
	-DSDL_OFFSCREEN=ON \
	-DSDL_X11=OFF \
	-DSDL_WAYLAND=OFF \
	-DSDL_COCOA=OFF \
	-DSDL_DIRECTX=OFF \
	-DSDL_OPENGL=OFF \
	-DSDL_VULKAN=OFF \
	-DSDL_CAMERA=OFF \
	-DSDL_PULSEAUDIO=OFF \
	-DSDL_JACK=OFF \
	-DSDL_OSS=OFF \
	-DSDL_PIPEWIRE=OFF \
	-DSDL_RPATH=OFF \
	-DSDL_ALSA=ON \
	-DSDL_ALSA_SHARED=OFF

$(eval $(cmake-package))
