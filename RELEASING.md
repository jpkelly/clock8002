# Release Process

This document covers cutting a new clock8002 release.

The primary release artifact is the **Trixie installer tarball**, built from `v4/` on the `master` branch. See [Trixie Installer Release](#trixie-installer-release) below.

The Buildroot SD card image is a **parked platform**. It is still maintained, but no new units are deployed on it and it is only built on explicit request. Its code lives on the `buildroot` branch.

> Branch note (2026-07-31): what was previously the `trixie` branch is now `master`, and what was previously `master` is now `buildroot`.

## Versioning

- Active release line: `v1.x` — do not use inherited upstream `v4.x` tags
- Tag format: `v1.x.y` (dropped the `trixie-` prefix starting with `v1.3.16`, since Trixie is
  now the only actively developed platform). Releases before that used `trixie-v1.x.y` —
  those existing tags are historical and are not renamed.
- Buildroot tag format: `v1.x.y` (parked; last release `v1.3.7`) — same tag namespace,
  disambiguated by branch (`buildroot`) rather than a prefix.

---

## Soak Gate (Required Before Release)

Run the piClock test unit with `shakemon.sh` or manual monitoring for at least 24h.

**12h checkpoint** — record `VmRSS`, `VmSwap`, system swap, service PIDs, temp/throttle. If clean, continue soak.

**24h checkpoint** — repeat. Approve release only if:
- `VmSwap` for `sdl-clock` is 0 or flat
- All service PIDs stable (no restarts)
- No throttle events

If any metric is rising or unstable, hold release and investigate.

---

## Buildroot Image Release (PARKED — only on explicit request)

See [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md) for the full build and flash workflow.

This platform is dormant. Unless you have specifically been asked for a Buildroot image, use the Trixie process below instead.

### 1. Update CHANGELOG.md and HANDOFF.md

Add a CHANGELOG entry. Update HANDOFF.md Current State section with the new tag and commit hash.

### 2. Commit, tag, and push

```bash
git add CHANGELOG.md HANDOFF.md
git commit -m "release: vX.X.X"
git tag vX.X.X
git push origin buildroot vX.X.X
```

### 3. Verify cm5 is on the buildroot branch at the new tag

```bash
ssh pi@cm5.local 'cd ~/clock8002 && git fetch --tags origin && git reset --hard origin/buildroot && git branch --show-current && git describe --tags HEAD'
```

### 4. Build on cm5

**Release images must never include an SSH key.** Always build without `--key`.

```bash
tools/buildroot/cm5-build-launch.sh --purpose release-<VERSION>
```

The launch script writes a manifest to `docs/manifests/` and prints monitor commands on launch.

### 5. Transfer image

```bash
scp pi@cm5.local:~/buildroot/output/images/sdcard.img /Users/jp/Desktop/piClock-<COMMIT>-sdcard.img
```

Naming convention: `piClock-<7-char-commit-hash>-sdcard.img`

### 6. Publish GitHub release

```bash
gh release create vX.X.X \
    --title "vX.X.X — Buildroot / SDL3" \
    --notes "<release notes>" \
    /Users/jp/Desktop/piClock-<COMMIT>-sdcard.img
```

---

## Trixie Installer Release

**This is the primary release path.** The `master` branch produces an installer tarball for Raspberry Pi OS / Debian Trixie (arm64).

### Versioning

- Tag format: `v1.x.y` (starting with `v1.3.16`; earlier releases used `trixie-v1.x.y`)
- Example: `v1.3.16`
- These are published as normal (Latest) GitHub releases.

### Build and publish

1. Ensure the `master` branch contains the desired code and HANDOFF.md is updated.
2. On an ARM64 Trixie builder — `pi@cm5.local` preferred, `pi@pi5start.local` as fallback when cm5 is unavailable:
   ```bash
   cd ~/clock8002 && git fetch origin && git checkout master && git reset --hard origin/master
   rm -rf v4/lib && cp -r ~/sdl3-build/sdl3-trixie-lib v4/lib
   cd v4
   make clean
   make release GIT_TAG=vX.X.X
   ```
3. Transfer the tarball locally and verify checksums match.
4. Tag and push:
   ```bash
   git tag -a vX.X.X -m "Clock 8002 release vX.X.X"
   git push origin vX.X.X
   ```
5. Create the release:
   ```bash
   gh release create vX.X.X \
       --title "Clock 8002 vX.X.X" \
       --notes-file /tmp/vX.X.X-release-notes.md \
       /Users/jp/Desktop/clock8002-vX.X.X-default-linux-arm64.tar.gz
   ```

### Notes

- **Release builds must use `make clean`**, not `clock8002-dirclean`. `output/target/` is not wiped by partial cleans — stale files from prior dev builds (including SSH keys) can persist.
- Do not use `pkill -f sdl-clock` in SSH commands — pattern matches can terminate the SSH session. Use `/etc/init.d/S99clock stop` instead.
- The Gerry deployment is only valid when both `clock.ini` and `network.ini` match Gerry settings.
