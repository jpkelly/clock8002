# Release Process

This document covers cutting a new clock8002 release. Every release includes both a **default** and a **Gerry** variant (Trixie tarballs). A Buildroot SD card image is produced separately — see [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md).

## Versioning

- Active release line: `v1.x` — do not use inherited upstream `v4.x` tags
- Tag format: `v1.x.y`

---

## Trixie Release (install.sh)

### 1. Update CHANGELOG.md

Add an entry for the new version.

### 2. Update README quick-install URL

In `README.md`, update the download URL and directory name in the Quick Install section to point to the new version.

### 3. Commit, tag, and push

```bash
git add CHANGELOG.md README.md
git commit -m "release: bump to vX.X.X"
git tag vX.X.X
git push origin master vX.X.X
```

### 4. Build on pi5start from a fresh clone

**Always build on `pi@pi5start.local` from a fresh clone at the target tag — never from a local Mac build.**

```bash
ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && git checkout vX.X.X && make release-all GIT_TAG=vX.X.X'
```

This produces:
- `clock8002-vX.X.X-default-linux-arm64.tar.gz`
- `clock8002-vX.X.X-gerry-linux-arm64.tar.gz`

### 5. Generate release notes

```bash
VERSION=vX.X.X
sed "s/__VERSION__/${VERSION}/g" .github/release-notes-template.md > /tmp/release-notes-${VERSION}.md
# Edit /tmp/release-notes-${VERSION}.md to fill in the "What Changed" section
```

### 6. Publish GitHub release

```bash
gh release create "${VERSION}" \
    "clock8002-${VERSION}-default-linux-arm64.tar.gz" \
    "clock8002-${VERSION}-gerry-linux-arm64.tar.gz" \
    --title "${VERSION}" \
    --notes-file "/tmp/release-notes-${VERSION}.md"
```

### 7. Deploy to test unit and verify

Relay the tarball to piclock.local and run the installer:

```bash
# Copy default tarball through Mac to piclock
scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-vX.X.X-default-linux-arm64.tar.gz /tmp/
scp /tmp/clock8002-vX.X.X-default-linux-arm64.tar.gz pi@piclock.local:/tmp/

# Install on piclock
ssh pi@piclock.local 'cd /tmp && tar xzf clock8002-vX.X.X-default-linux-arm64.tar.gz && cd clock8002-vX.X.X-default-linux-arm64 && sudo bash install.sh > /tmp/install-vX.X.X.log 2>&1; echo INSTALL_EXIT:$?'

# Start and verify services
ssh pi@piclock.local 'sudo systemctl start clock8002 alsa-ltc oled_daemon && systemctl is-active clock8002 alsa-ltc oled_daemon'
```

Report the deployed short commit hash (first 7 characters).

**For Gerry variant:** force-apply the Gerry config pair after installing the default:

```bash
ssh pi@piclock.local 'sudo cp /tmp/clock8002-vX.X.X-gerry-linux-arm64/clock.ini /boot/firmware/piclock/clock.ini && sudo cp /tmp/clock8002-vX.X.X-gerry-linux-arm64/network.ini /boot/firmware/piclock/network.ini && sudo reboot'
```

---

## Buildroot Image Release

See [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md) for the full build and flash workflow.

Key points:
- Build host: `pi@pi5start.local` — apply manual patches (Mesa 25.0.7, host-xz) before building (issue #29)
- Image naming: `piclockBR-<version>-sdcard.img` (e.g. `piclockBR-v1.2.4-sdcard.img`)
- Both images (default and Gerry) are Trixie tarballs only — the Buildroot image is config-neutral (config lives on the boot partition)
- **Release images must never include an SSH key** — always build without `BR2_PICLOCKKEY` set. Password-only access (`clockworkadmin`) is the only auth method on release images. The `BR2_PICLOCKKEY` mechanism is for dev builds only and must never be used when cutting a release.
- **Release builds must use `make clean`**, not `clock8002-dirclean`. `output/target/` is not wiped by partial cleans — stale files from prior dev builds (including SSH keys) persist. `make clean` ensures a provably clean rootfs. Build command: `cd ~/clock8002 && git checkout vX.X.X && cd ~/buildroot && make clean && make`

---

## Notes

- Do not use `pkill -f /opt/clock8002/sdl-clock` or `pkill -f /opt/clock8002/alsa-ltc` in SSH commands — pattern matches can terminate the SSH session (exit 255). Use `systemctl stop` instead.
- `install.sh` preserves an existing `/boot/piclock/clock.ini` — it only installs the packaged config on a fresh install. Existing units may need an explicit config copy.
- The Gerry deployment is only valid when both `clock.ini` and `network.ini` match Gerry settings.
