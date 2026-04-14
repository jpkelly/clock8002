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

echo ""
echo "Done. All build-host patches applied."
echo "If mesa3d was already in output/build/, run: make mesa3d-dirclean"
