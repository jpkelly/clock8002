# Release Process

This document covers cutting a new clock8002 release. Every release includes both a **default** and a **Gerry** variant (Trixie tarballs). A Buildroot SD card image is produced separately — see [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md).

## Versioning

- Active release line: `v1.x` — do not use inherited upstream `v4.x` tags
- Tag format: `v1.x.y`

---

## Soak Gate (Required Before Release)

Run both a **default** and **gerry** variant on separate 2GB Pi 5 boards with `vl805-soak-monitor.sh` for at least 12h.

**12h checkpoint** — record status table (uptime, USB errors, LTC decoded, restarts, temp). If both boards are clean, continue soak.

**24h checkpoint** — repeat status table. Approve release only if:
- Both boards: `xhci_err: 0`, `restarts: 0`, no `HC died` events
- Both boards: `alsa-ltc` continuously active with LTC decoding (or stably awaiting a source)

If either board fails the gate, hold release and investigate.

Soak monitor commands:
```bash
ssh pi@piclockTD.local 'tail -5 /tmp/vl805-monitor.log'
ssh pi@piclockTG.local 'tail -5 /tmp/vl805-monitor.log'
```

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

### 4. Build on cm5 from a fresh clone

**Always build on `pi@cm5.local` from a fresh clone at the target tag — never from a local Mac build. Use a full (non-shallow) clone so `git describe` resolves correctly.**

```bash
ssh pi@cm5.local 'cd /tmp && rm -rf clock8002-build && git clone https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && git checkout vX.X.X && make release NETWORK_CONFIG=default && make release NETWORK_CONFIG=gerry'
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

### 7. Fresh-install smoke test (required)

Flash a clean Trixie SD card, boot, and install from the GitHub release exactly as an end user would:

```bash
wget https://github.com/jpkelly/clock8002/releases/download/vX.X.X/clock8002-vX.X.X-default-linux-arm64.tar.gz
tar xzf clock8002-vX.X.X-default-linux-arm64.tar.gz
cd clock8002-vX.X.X-default-linux-arm64
sudo bash install.sh
sudo reboot
```

After reboot, verify:
- Clock face is visible on HDMI (not a console)
- `systemctl is-active clock8002 alsa-ltc oled_daemon` → all `active`
- Web UI responds on port 8080

Do not skip this step. Previous releases missed installer bugs that only appear on fresh systems because test units already had directories with correct ownership.

### 8. Install on test units and verify

After the GitHub release is published, install on both test units from GitHub exactly as an end user would. **The user does this manually — never agent-deploy releases.**

**Default variant on piclockTD:**
```bash
wget https://github.com/jpkelly/clock8002/releases/download/vX.X.X/clock8002-vX.X.X-default-linux-arm64.tar.gz
tar xzf clock8002-vX.X.X-default-linux-arm64.tar.gz
cd clock8002-vX.X.X-default-linux-arm64
sudo bash install.sh
sudo reboot
```

**Gerry variant on piclockTG:**
```bash
wget https://github.com/jpkelly/clock8002/releases/download/vX.X.X/clock8002-vX.X.X-gerry-linux-arm64.tar.gz
tar xzf clock8002-vX.X.X-gerry-linux-arm64.tar.gz
cd clock8002-vX.X.X-gerry-linux-arm64
sudo bash install.sh
sudo reboot
```

After reboot on each unit, verify:
- Clock face is visible on HDMI
- `systemctl is-active clock8002 alsa-ltc oled_daemon` → all `active`
- Web UI responds on port 8080

---

## Buildroot Image Release

See [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md) for the full build and flash workflow.

Key points:
- Build host: `pi@cm5.local` — apply manual patches (Mesa 25.0.7, host-xz) before building (issue #29)
- Image naming: `piclockBR-<version>-sdcard.img` (e.g. `piclockBR-v1.2.4-sdcard.img`)
- Both images (default and Gerry) are Trixie tarballs only — the Buildroot image is config-neutral (config lives on the boot partition)
- **Release images must never include an SSH key** — always build without `BR2_PICLOCKKEY` set. Password-only access (`clockworkadmin`) is the only auth method on release images. The `BR2_PICLOCKKEY` mechanism is for dev builds only and must never be used when cutting a release.
- **Release builds must use `make clean`**, not `clock8002-dirclean`. `output/target/` is not wiped by partial cleans — stale files from prior dev builds (including SSH keys) persist. `make clean` ensures a provably clean rootfs. Build command: `cd ~/clock8002 && git checkout vX.X.X && cd ~/buildroot && make clean && make`

---

## Notes

- Do not use `pkill -f /opt/clock8002/sdl-clock` or `pkill -f /opt/clock8002/alsa-ltc` in SSH commands — pattern matches can terminate the SSH session (exit 255). Use `systemctl stop` instead.
- `install.sh` preserves an existing `/boot/piclock/clock.ini` — it only installs the packaged config on a fresh install. Existing units may need an explicit config copy.
- The Gerry deployment is only valid when both `clock.ini` and `network.ini` match Gerry settings.
