#!/bin/sh
# Apply manual build-host patches to a Buildroot 2025.02 source tree.
# See: https://github.com/jpkelly/clock8002/issues/29
#
# Idempotent: safe to run multiple times.
#
# Usage: ./apply-build-host-patches.sh [path-to-buildroot]
# Default buildroot path: ~/buildroot

set -e

BUILDROOT="${1:-$HOME/buildroot}"

if [ ! -f "$BUILDROOT/Makefile" ]; then
    echo "ERROR: Buildroot not found at $BUILDROOT"
    echo "Usage: $0 [path-to-buildroot]"
    exit 1
fi

echo "Applying build-host patches to: $BUILDROOT"

# ---------------------------------------------------------------------------
# Mesa 25.0.7 upgrade (package/mesa3d/)
# ---------------------------------------------------------------------------
MESA3D_MK="$BUILDROOT/package/mesa3d/mesa3d.mk"
MESA3D_HASH="$BUILDROOT/package/mesa3d/mesa3d.hash"

# 1. Version bump
if grep -q "MESA3D_VERSION = 24.0.9" "$MESA3D_MK"; then
    sed -i 's/MESA3D_VERSION = 24.0.9/MESA3D_VERSION = 25.0.7/' "$MESA3D_MK"
    echo "  mesa3d.mk: version bumped to 25.0.7"
else
    echo "  mesa3d.mk: version already updated (skipping)"
fi

# 2. Add host-python-pyyaml dependency (required by Mesa 25.0.7 build system)
if ! grep -q "host-python-pyyaml" "$MESA3D_MK"; then
    # Use awk to insert after host-python-mako line (sed \t is not portable across shells)
    awk '/host-python-mako \\/{print; print "\thost-python-pyyaml \\"; next}1' "$MESA3D_MK" > "${MESA3D_MK}.tmp" && mv "${MESA3D_MK}.tmp" "$MESA3D_MK"
    echo "  mesa3d.mk: host-python-pyyaml dependency added"
else
    echo "  mesa3d.mk: host-python-pyyaml already present (skipping)"
fi

# 3. Remove deprecated meson options (removed in Mesa 25.x)
for opt in "gallium-omx" "dri3" "gallium-vc4-neon"; do
    if grep -q "\-D${opt}=" "$MESA3D_MK"; then
        grep -v "\-D${opt}=" "$MESA3D_MK" > "${MESA3D_MK}.tmp" && mv "${MESA3D_MK}.tmp" "$MESA3D_MK"
        echo "  mesa3d.mk: removed deprecated option -D${opt}="
    else
        echo "  mesa3d.mk: -D${opt}= already absent (skipping)"
    fi
done

# 4. Update hash file
if ! grep -q "mesa-25.0.7.tar.xz" "$MESA3D_HASH"; then
    cat > "$MESA3D_HASH" << 'EOF'
# From https://mesa.freedesktop.org/archive/
sha256  592272df3cf01e85e7db300c449df5061092574d099da275d19e97ef0510f8a6  mesa-25.0.7.tar.xz
sha512  825bbd8bc5507de147488519786c0200afacf97dae621c80ead24b2c5dd55c5a442757ac8452698ae611e9344025465080795cf8f2dc4eb7ce07b5cc521b2b5c  mesa-25.0.7.tar.xz
# License
sha256  a00275a53178e2645fb65be99a785c110513446a5071ff2c698ed260ad917d75  docs/license.rst
EOF
    echo "  mesa3d.hash: updated for 25.0.7"
else
    echo "  mesa3d.hash: already updated (skipping)"
fi

# 5. Remove incompatible patches (OpenCL/uClibc/ARM32 NEON — irrelevant for glibc aarch64)
PATCH_REMOVED=0
for patch in "$BUILDROOT/package/mesa3d/"*.patch; do
    [ -f "$patch" ] || continue
    rm "$patch"
    echo "  mesa3d: removed incompatible patch: $(basename "$patch")"
    PATCH_REMOVED=1
done
[ "$PATCH_REMOVED" -eq 0 ] && echo "  mesa3d: no patches to remove (skipping)"

# ---------------------------------------------------------------------------
# host-xz libtool bug workaround (package/xz/xz.mk)
# ---------------------------------------------------------------------------
XZ_MK="$BUILDROOT/package/xz/xz.mk"

if ! grep -q 'acl_cv_wl' "$XZ_MK"; then
    sed -i 's|CXX="$(HOSTCXX_NOCCACHE)"|CXX="$(HOSTCXX_NOCCACHE)" acl_cv_wl="-Wl,"|' "$XZ_MK"
    echo "  xz.mk: acl_cv_wl workaround applied"
else
    echo "  xz.mk: acl_cv_wl already present (skipping)"
fi

# ---------------------------------------------------------------------------
# Go 1.24.2 upgrade (package/go/)
# Required: v4/go.mod needs go >= 1.24.0; Buildroot 2025.02 ships 1.23.7
# ---------------------------------------------------------------------------
GO_MK="$BUILDROOT/package/go/go.mk"

if grep -q "GO_VERSION = 1.23.7" "$GO_MK"; then
    sed -i 's/GO_VERSION = 1.23.7/GO_VERSION = 1.24.2/' "$GO_MK"
    echo "  go.mk: version bumped to 1.24.2"
else
    echo "  go.mk: version already updated (skipping)"
fi

for HFILE in \
    "$BUILDROOT/package/go/go.hash" \
    "$BUILDROOT/package/go/go-bin/go-bin.hash" \
    "$BUILDROOT/package/go/go-src/go-src.hash"; do
    if ! grep -q "go1.24.2.linux-arm64.tar.gz" "$HFILE"; then
        cat >> "$HFILE" << 'GOHASHES'
sha256  9dc77ffadc16d837a1bf32d99c624cb4df0647cee7b119edd9e7b1bcc05f2e00  go1.24.2.src.tar.gz
sha256  4c382776d52313266f3026236297a224a6688751256a2dffa3f524d8d6f6c0ba  go1.24.2.linux-386.tar.gz
sha256  68097bd680839cbc9d464a0edce4f7c333975e27a90246890e9f1078c7e702ad  go1.24.2.linux-amd64.tar.gz
sha256  756274ea4b68fa5535eb9fe2559889287d725a8da63c6aae4d5f23778c229f4b  go1.24.2.linux-arm64.tar.gz
sha256  438d5d3d7dcb239b58d893a715672eabe670b9730b1fd1c8fc858a46722a598a  go1.24.2.linux-armv6l.tar.gz
sha256  5fff857791d541c71d8ea0171c73f6f99770d15ff7e2ad979104856d01f36563  go1.24.2.linux-ppc64le.tar.gz
sha256  1cb3448166d6abb515a85a3ee5afbdf932081fb58ad7143a8fb666fbc06146d9  go1.24.2.linux-s390x.tar.gz
GOHASHES
        echo "  $(basename "$HFILE"): go1.24.2 hashes appended"
    else
        echo "  $(basename "$HFILE"): go1.24.2 already present (skipping)"
    fi
done

# ---------------------------------------------------------------------------
# rpi-firmware upgrade to 2025-12-08 (package/rpi-firmware/)
# Buildroot 2025.02.x ships 5476720d (2025-05-08); that RP1 firmware has a
# bug causing xHCI controller death when USB audio isochronous streaming
# starts (alsa-ltc).  Buildroot master already bumped to 063bcab6 (2025-12-08)
# which matches the working reference system.
# ---------------------------------------------------------------------------
RPI_FW_MK="$BUILDROOT/package/rpi-firmware/rpi-firmware.mk"
RPI_FW_HASH="$BUILDROOT/package/rpi-firmware/rpi-firmware.hash"

OLD_RPI_COMMIT="5476720d52cf579dc1627715262b30ba1242525e"
NEW_RPI_COMMIT="063bcab6c8a90efb0d19f69d88cbbc7ec79cab68"
OLD_RPI_SHA256="00fe5487376e9d5ed14cbc72a9b7a5bd8d6900c84aee10f6d656f192a26d161c"
NEW_RPI_SHA256="812cb64617d6ffbc76f9b8dce8237b31159a1216b31845faec1bb75b6b6b291f"

if grep -q "$OLD_RPI_COMMIT" "$RPI_FW_MK"; then
    sed -i "s/$OLD_RPI_COMMIT/$NEW_RPI_COMMIT/" "$RPI_FW_MK"
    echo "  rpi-firmware.mk: version bumped to 063bcab6 (2025-12-08)"
elif grep -q "$NEW_RPI_COMMIT" "$RPI_FW_MK"; then
    echo "  rpi-firmware.mk: already at 063bcab6 (skipping)"
else
    echo "  rpi-firmware.mk: unexpected version — manual check required"
fi

if grep -q "$OLD_RPI_SHA256" "$RPI_FW_HASH"; then
    sed -i "s/$OLD_RPI_SHA256  rpi-firmware-${OLD_RPI_COMMIT}\.tar\.gz/$NEW_RPI_SHA256  rpi-firmware-${NEW_RPI_COMMIT}.tar.gz/" "$RPI_FW_HASH"
    echo "  rpi-firmware.hash: updated for 063bcab6"
elif grep -q "$NEW_RPI_SHA256" "$RPI_FW_HASH"; then
    echo "  rpi-firmware.hash: already updated (skipping)"
else
    echo "  rpi-firmware.hash: unexpected content — manual check required"
fi

echo ""
echo "Done. All build-host patches applied."
echo "If mesa3d was already in output/build/, run: make mesa3d-dirclean"
echo "If go was already in output/build/, run: make host-go-dirclean"
echo "If rpi-firmware was already in output/build/, run: make rpi-firmware-dirclean"
