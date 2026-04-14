################################################################################
# sdl3-ttf
################################################################################

SDL3_TTF_VERSION = 3.2.2
SDL3_TTF_SITE = https://github.com/libsdl-org/SDL_ttf/releases/download/release-$(SDL3_TTF_VERSION)
SDL3_TTF_SOURCE = SDL3_ttf-$(SDL3_TTF_VERSION).tar.gz
SDL3_TTF_LICENSE = Zlib
SDL3_TTF_LICENSE_FILES = LICENSE.txt
SDL3_TTF_INSTALL_STAGING = YES

SDL3_TTF_DEPENDENCIES = sdl3 freetype harfbuzz

SDL3_TTF_CONF_OPTS = \
	-DSDL3TTF_SHARED=ON \
	-DSDL3TTF_STATIC=OFF \
	-DSDL3TTF_SAMPLES=OFF \
	-DSDL3TTF_INSTALL=ON \
	-DSDL3TTF_HARFBUZZ=ON \
	-DSDL3TTF_FREETYPE=ON

$(eval $(cmake-package))
