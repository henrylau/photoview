#!/bin/bash

# Fallback to the latest version if LIBVIPS_VERSION is not set
if [[ -z "${LIBVIPS_VERSION}" ]]; then
  echo "WARN: libvips version is empty, most likely the script runs not on CI."
  echo "Fetching the latest version from libvips repo..."
  LIBVIPS_VERSION=$(curl -fsSL --retry 2 --retry-delay 5 --retry-max-time 60 \
    "https://api.github.com/repos/libvips/libvips/releases/latest" | jq -r '.tag_name')
fi

set -euo pipefail

: "${DEB_HOST_ARCH:=$(dpkg --print-architecture)}"
: "${DEB_HOST_GNU_TYPE:=$(dpkg-architecture -a "$DEB_HOST_ARCH" -qDEB_HOST_GNU_TYPE)}"
CACHE_DIR="${BUILD_CACHE_DIR:-/build-cache}/libvips-${LIBVIPS_VERSION}"
CACHE_MARKER="${CACHE_DIR}/libvips-${LIBVIPS_VERSION}-complete"

# Check if this specific version is already built and cached
if [[ -f "$CACHE_MARKER" ]] && [[ -d "${CACHE_DIR}/output" ]]; then
  echo "libvips ${LIBVIPS_VERSION} found in cache, reusing..."
  mkdir -p /output
  cp -ra "${CACHE_DIR}/output/"* /output/
  exit 0
fi

echo "Building libvips ${LIBVIPS_VERSION} (cache miss)..."

echo Compiler: "${DEB_HOST_GNU_TYPE}" Arch: "${DEB_HOST_ARCH}"

# Install build dependencies
apt-get install -y \
  ninja-build:"${DEB_HOST_ARCH}" \
  python3-pip \
  meson \
  pkg-config:"${DEB_HOST_ARCH}" \
  libglib2.0-dev:"${DEB_HOST_ARCH}" \
  libexpat1-dev:"${DEB_HOST_ARCH}" \
  libfftw3-dev:"${DEB_HOST_ARCH}" \
  libopenexr-dev:"${DEB_HOST_ARCH}" \
  libgsf-1-dev:"${DEB_HOST_ARCH}" \
  liborc-dev:"${DEB_HOST_ARCH}" \
  libopenslide-dev:"${DEB_HOST_ARCH}" \
  libmatio-dev:"${DEB_HOST_ARCH}" \
  libwebp-dev:"${DEB_HOST_ARCH}" \
  libjpeg62-turbo-dev:"${DEB_HOST_ARCH}" \
  libexif-dev:"${DEB_HOST_ARCH}" \
  libtiff-dev:"${DEB_HOST_ARCH}" \
  libcfitsio-dev:"${DEB_HOST_ARCH}" \
  libpoppler-glib-dev:"${DEB_HOST_ARCH}" \
  librsvg2-dev:"${DEB_HOST_ARCH}" \
  libpango1.0-dev:"${DEB_HOST_ARCH}" \
  libopenjp2-7-dev:"${DEB_HOST_ARCH}" \
  liblcms2-dev:"${DEB_HOST_ARCH}" \
  libimagequant-dev:"${DEB_HOST_ARCH}" \
  libjxl-dev:"${DEB_HOST_ARCH}" \
  libheif-dev:"${DEB_HOST_ARCH}" \
  zlib1g-dev:"${DEB_HOST_ARCH}" \
  liblzma-dev:"${DEB_HOST_ARCH}" \
  libbz2-dev:"${DEB_HOST_ARCH}"

URL="https://api.github.com/repos/libvips/libvips/tarball/${LIBVIPS_VERSION}"
echo download libvips from "$URL"
curl -fsSL --retry 2 --retry-delay 5 --retry-max-time 60 -o ./libvips.tar.gz \
  ${GITHUB_TOKEN:+-H "Authorization: Bearer ${GITHUB_TOKEN}"} "$URL"

tar xfv ./libvips.tar.gz
cd libvips-*

# Configure with Meson
meson setup build --libdir=lib --buildtype=release

# Build
cd build
meson compile
meson install
cd ..

mkdir -p /output/bin /output/lib /output/include /output/pkgconfig
cp -a /usr/local/bin/vips /output/bin/
cp -a /usr/local/lib/libvips* /output/lib/
cp -a /usr/local/lib/pkgconfig/vips* /output/pkgconfig/
cp -a /usr/local/include/vips /output/include/
file /usr/local/lib/libvips.so*

# After successful build, cache the results
echo "Caching libvips ${LIBVIPS_VERSION} build results..."
mkdir -p "${CACHE_DIR}/output"
cp -ra /output/* "${CACHE_DIR}/output/"
touch "$CACHE_MARKER"

echo "libvips ${LIBVIPS_VERSION} build complete and cached"
