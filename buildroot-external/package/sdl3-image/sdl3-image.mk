################################################################################
# sdl3-image
################################################################################

SDL3_IMAGE_VERSION = 3.2.6
SDL3_IMAGE_SITE = https://github.com/libsdl-org/SDL_image/releases/download/release-$(SDL3_IMAGE_VERSION)
SDL3_IMAGE_SOURCE = SDL3_image-$(SDL3_IMAGE_VERSION).tar.gz
SDL3_IMAGE_LICENSE = Zlib
SDL3_IMAGE_LICENSE_FILES = LICENSE.txt
SDL3_IMAGE_INSTALL_STAGING = YES

SDL3_IMAGE_DEPENDENCIES = sdl3 libpng jpeg

SDL3_IMAGE_CONF_OPTS = \
	-DSDL3IMAGE_SHARED=ON \
	-DSDL3IMAGE_STATIC=OFF \
	-DSDL3IMAGE_SAMPLES=OFF \
	-DSDL3IMAGE_INSTALL=ON \
	-DSDL3IMAGE_PNG=ON \
	-DSDL3IMAGE_JPG=ON \
	-DSDL3IMAGE_WEBP=OFF \
	-DSDL3IMAGE_AVIF=OFF \
	-DSDL3IMAGE_JXL=OFF \
	-DSDL3IMAGE_TIF=OFF

$(eval $(cmake-package))
