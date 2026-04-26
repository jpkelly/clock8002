# Release Process

This document covers cutting a new clock8002 release. The primary release artifact is the **Buildroot SD card image**. The legacy Trixie/SDL2 install.sh path is archived on the `trixie` branch.

## Versioning

- Active release line: `v1.x` — do not use inherited upstream `v4.x` tags
- Tag format: `v1.x.y`

---

## Soak Gate (Required Before Release)

Run the piClock test unit with `shakemon.sh` or manual monitoring for at least 24h.

**12h checkpoint** — record `VmRSS`, `VmSwap`, system swap, service PIDs, temp/throttle. If clean, continue soak.

**24h checkpoint** — repeat. Approve release only if:
- `VmSwap` for `sdl3-clock` is 0 or flat
- All service PIDs stable (no restarts)
- No throttle events

If any metric is rising or unstable, hold release and investigate.

---

## Buildroot Image Release

See [buildroot-external/README.buildroot.md](buildroot-external/README.buildroot.md) for the full build and flash workflow.

### 1. Update CHANGELOG.md and HANDOFF.md

Add a CHANGELOG entry. Update HANDOFF.md Current State section with the new tag and commit hash.

### 2. Commit, tag, and push

```bash
git add CHANGELOG.md HANDOFF.md
git commit -m "release: vX.X.X"
git tag vX.X.X
git push origin master vX.X.X
```

### 3. Verify cm5 is on master at the new tag

```bash
ssh pi@cm5.local 'cd ~/clock8002 && git fetch --tags origin && git reset --hard origin/master && git branch --show-current && git describe --tags HEAD'
```

### 4. Build on cm5

**Release images must never include an SSH key.** Always build without `BR2_PICLOCKKEY`.

```bash
ssh pi@cm5.local 'cd ~/buildroot && make clean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$?'
```

Monitor: `ssh pi@cm5.local 'tail -f /tmp/br-build.log'`

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

## Notes

- **Release builds must use `make clean`**, not `clock8002-dirclean`. `output/target/` is not wiped by partial cleans — stale files from prior dev builds (including SSH keys) can persist.
- Do not use `pkill -f /opt/clock8002/sdl3-clock` in SSH commands — pattern matches can terminate the SSH session. Use `/etc/init.d/S99clock stop` instead.
- The Gerry deployment is only valid when both `clock.ini` and `network.ini` match Gerry settings.
