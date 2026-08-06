# Release Process

This document covers cutting a new clock8002 release.

The release artifact is the **Trixie installer tarball**, built from `app/` on the `master` branch. Trixie is the only actively developed platform.

## Versioning

- Active release line: `v1.x` — do not use inherited upstream `v4.x` tags
- Tag format: `v1.x.y` (dropped the `trixie-` prefix starting with `v1.3.16`, since Trixie is
  now the only actively developed platform). Releases before that used `trixie-v1.x.y` —
  those existing tags are historical and are not renamed.

---

## Soak Gate (Required Before Release)

Run the piClock test unit with `shakemon.sh` or manual monitoring for at least 12h.

**6h checkpoint** — record `VmRSS`, `VmSwap`, system swap, service PIDs, temp/throttle. If clean, continue soak.

**12h checkpoint** — repeat. Approve release only if:
- `VmSwap` for `sdl-clock` is 0 or flat
- All service PIDs stable (no restarts)
- No throttle events

If any metric is rising or unstable, hold release and investigate.

---

## Trixie Installer Release

**This is the primary release path.** The `master` branch produces an installer tarball for Raspberry Pi OS / Debian Trixie (arm64).

### Versioning

- Tag format: `v1.x.y` (starting with `v1.3.16`; earlier releases used `trixie-v1.x.y`)
- Example: `v1.3.16`
- These are published as normal (Latest) GitHub releases.

### Build and publish

1. Ensure the `master` branch contains the desired code and HANDOFF.md is updated.
2. **Tag before building.** The Makefile falls back to `git describe --tags --abbrev=0 HEAD`
   when `GIT_TAG` isn't passed, which resolves to the nearest ancestor tag (e.g. a leftover
   `-rc1`/`-rc2`) if HEAD itself isn't tagged yet. Tagging first closes that trap:
   ```bash
   git tag -a vX.X.X -m "Clock 8002 release vX.X.X"
   git push origin vX.X.X
   ```
3. On an ARM64 Trixie builder — `pi@cm5.local` preferred, `pi@pi5start.local` as fallback when cm5 is unavailable:
   ```bash
   cd ~/clock8002 && git fetch --tags origin && git checkout vX.X.X
   rm -rf app/lib && cp -r ~/sdl3-build/sdl3-trixie-lib app/lib
   cd app
   make clean
   make release GIT_TAG=vX.X.X
   ```
   Pass `GIT_TAG` explicitly even though HEAD is now tagged — belt and braces.
4. Verify the built binary reports the tag you expect before publishing, not just that the
   build succeeded:
   ```bash
   tar xzOf piClock-vX.X.X-linux-arm64.tar.gz piClock-vX.X.X-linux-arm64/alsa-ltc > /tmp/a && chmod +x /tmp/a && /tmp/a --version
   ```
5. **Publish directly from the builder (default).** `cm5.local` has `gh` installed and
   authenticated — publish the release from there rather than round-tripping the ~30 MB
   tarball through a laptop:
   ```bash
   gh release create vX.X.X \
       --title "Clock 8002 vX.X.X" \
       --notes-file /tmp/vX.X.X-release-notes.md \
       --latest \
       piClock-vX.X.X-linux-arm64.tar.gz
   ```
   **Fallback (builder has no `gh`, or `gh auth status` fails there):** `scp` the tarball to
   the Mac and run `gh release create` locally instead. If you do this, verify the checksum
   on both ends before publishing — a truncated `scp` produces a corrupt but plausible-sized
   file, and only the checksum catches it:
   ```bash
   scp pi@cm5.local:~/clock8002/app/piClock-vX.X.X-linux-arm64.tar.gz /Users/jp/Desktop/
   shasum -a 256 /Users/jp/Desktop/piClock-vX.X.X-linux-arm64.tar.gz   # compare against sha256sum on the builder
   ```

### Notes

- **Release builds must use `make clean`.** Stale files from prior dev builds can persist otherwise.
- Do not use `pkill -f sdl-clock` in SSH commands — pattern matches can terminate the SSH session. Use `/etc/init.d/S99clock stop` instead.
- There is a single release variant (`default`). The `gerry` variant was removed in `v1.4.0` — do not build or publish a `-gerry-` tarball.
