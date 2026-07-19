#!/bin/bash
set -e

# Build SDL3 3.4.0 + SDL3_image 3.2.6 + SDL3_ttf 3.2.2 for Debian/Raspberry Pi OS Trixie arm64.
# Outputs a relocatable lib/ directory intended to be bundled with the clock8002 release.

SDL3_VERSION=3.4.0
SDL3_IMAGE_VERSION=3.2.6
SDL3_TTF_VERSION=3.2.2

INSTALL_PREFIX="${PWD}/sdl3-trixie-install"
BUILD_DIR="${PWD}/sdl3-trixie-build"
OUTPUT_DIR="${PWD}/sdl3-trixie-lib"
JOBS=$(nproc)

rm -rf "${INSTALL_PREFIX}" "${BUILD_DIR}" "${OUTPUT_DIR}"
mkdir -p "${INSTALL_PREFIX}" "${BUILD_DIR}" "${OUTPUT_DIR}"

cd "${BUILD_DIR}"

# Fetch sources
if [ ! -f "SDL3-${SDL3_VERSION}.tar.gz" ]; then
    wget -q "https://github.com/libsdl-org/SDL/releases/download/release-${SDL3_VERSION}/SDL3-${SDL3_VERSION}.tar.gz"
fi
if [ ! -f "SDL3_image-${SDL3_IMAGE_VERSION}.tar.gz" ]; then
    wget -q "https://github.com/libsdl-org/SDL_image/releases/download/release-${SDL3_IMAGE_VERSION}/SDL3_image-${SDL3_IMAGE_VERSION}.tar.gz"
fi
if [ ! -f "SDL3_ttf-${SDL3_TTF_VERSION}.tar.gz" ]; then
    wget -q "https://github.com/libsdl-org/SDL_ttf/releases/download/release-${SDL3_TTF_VERSION}/SDL3_ttf-${SDL3_TTF_VERSION}.tar.gz"
fi

tar xzf "SDL3-${SDL3_VERSION}.tar.gz"
tar xzf "SDL3_image-${SDL3_IMAGE_VERSION}.tar.gz"
tar xzf "SDL3_ttf-${SDL3_TTF_VERSION}.tar.gz"

# Build SDL3
cmake -S "SDL3-${SDL3_VERSION}" -B "SDL3-${SDL3_VERSION}-build" \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX="${INSTALL_PREFIX}" \
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
    -DSDL_ALSA_SHARED=ON

cmake --build "SDL3-${SDL3_VERSION}-build" --parallel "${JOBS}"
cmake --install "SDL3-${SDL3_VERSION}-build"

# Build SDL3_image
cmake -S "SDL3_image-${SDL3_IMAGE_VERSION}" -B "SDL3_image-${SDL3_IMAGE_VERSION}-build" \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX="${INSTALL_PREFIX}" \
    -DCMAKE_PREFIX_PATH="${INSTALL_PREFIX}" \
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

cmake --build "SDL3_image-${SDL3_IMAGE_VERSION}-build" --parallel "${JOBS}"
cmake --install "SDL3_image-${SDL3_IMAGE_VERSION}-build"

# Build SDL3_ttf
cmake -S "SDL3_ttf-${SDL3_TTF_VERSION}" -B "SDL3_ttf-${SDL3_TTF_VERSION}-build" \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX="${INSTALL_PREFIX}" \
    -DCMAKE_PREFIX_PATH="${INSTALL_PREFIX}" \
    -DSDL3TTF_SHARED=ON \
    -DSDL3TTF_STATIC=OFF \
    -DSDL3TTF_SAMPLES=OFF \
    -DSDL3TTF_INSTALL=ON \
    -DSDL3TTF_HARFBUZZ=ON \
    -DSDL3TTF_FREETYPE=ON

cmake --build "SDL3_ttf-${SDL3_TTF_VERSION}-build" --parallel "${JOBS}"
cmake --install "SDL3_ttf-${SDL3_TTF_VERSION}-build"

# Collect shared libraries into output directory
cp "${INSTALL_PREFIX}/lib/libSDL3.so"* "${OUTPUT_DIR}/"
cp "${INSTALL_PREFIX}/lib/libSDL3_image.so"* "${OUTPUT_DIR}/"
cp "${INSTALL_PREFIX}/lib/libSDL3_ttf.so"* "${OUTPUT_DIR}/"

# Remove absolute RPATH if present
for lib in "${OUTPUT_DIR}"/*.so.*; do
    if command -v patchelf >/dev/null 2>&1; then
        patchelf --remove-rpath "$lib" 2>/dev/null || true
    fi
done

cd "${PWD}"
echo "SDL3 libraries built in: ${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}"
