Repository workflow note:
- Build machine: pi@pi5start.local
- Test machine: pi@piclock.local
- Keep build machine current: before any build/release task on pi5start.local, use a fresh clone at the target tag/branch or run `git fetch --tags origin` + `git pull --ff-only` in the working copy.
- Source-of-truth rule: for build/test/release on pi5start.local, create a clean clone from `https://github.com/jpkelly/clock8002.git` (or verify the working copy matches that origin and target ref) before running commands.

Dev-deploy workflow note (feature branch testing, NOT a release):
- `install.sh` must be run on the TARGET machine (piclock) from a flat release directory — never from the source tree.
- The Makefile `release` target flattens all assets (including ttf_fonts/*.ttf) into a release dir and tarballs it.
- Reliability rules for remote deploy sessions:
  - Do not use `pkill -f /opt/clock8002/sdl-clock` or `pkill -f /opt/clock8002/alsa-ltc` in one-shot SSH deploy commands; pattern matches can terminate the remote session and cause SSH exit 255.
  - Prefer `sudo systemctl stop ...` and `sudo systemctl kill ...` for process teardown.
  - Run installer as `sudo bash install.sh > /tmp/<install-log>.log 2>&1` and check `INSTALL_EXIT` before restarting services.
  - After installer completion, explicitly start and verify `clock8002`, `alsa-ltc`, and `oled_daemon` service state.
- Steps:
  1. Clone branch and build on pi5start: `ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone --depth 1 --branch BRANCH https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && make release NETWORK_CONFIG=default'`
  2. Relay tarball to piclock (note: GIT_TAG defaults to nearest ancestor tag, e.g. v1.0.3): `scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-*-default-linux-arm64.tar.gz /tmp/ && scp /tmp/clock8002-*-default-linux-arm64.tar.gz pi@piclock.local:/tmp/`
  3. Install on piclock: `ssh pi@piclock.local 'cd /tmp && tar xzf clock8002-*-default-linux-arm64.tar.gz && cd clock8002-*-default-linux-arm64 && sudo bash install.sh'`
- After deploying to the test unit, always report the short GitHub commit hash (first 7 characters) that was deployed.

Issue management note:
- From now on I’ll only close an issue if you explicitly tell me to close it.

Test management note:
- If you are going to run a test, ask first and then I will decide whether or not you should run the test.

Change control note:
- Do not make code changes by default.
- Before editing any repository files, first present findings/status and ask for explicit user approval to proceed with code changes.
- Exception: if the user explicitly asks to implement/fix/change code in the current request, proceed.

Stability decision gate note:
- When investigating suspected memory/resource leaks, do not deploy code fixes until the current baseline run on the fully-populated 2GB piclock unit reaches at least the 24h checkpoint, unless the user explicitly overrides this gate.
- Required decision checkpoints: 12h and 24h with the same metrics (`VmRSS`, `VmSwap`, system swap, service state, temperature/throttle).
- Approve code change only if leak behavior is reproduced on that unit (e.g., materially rising `VmSwap` for `sdl-clock` or sustained RSS growth over time). If metrics are flat by 24h, hold changes and treat prior 1GB findings as non-generalized.

Release management note:
- For each new release, be sure to include both a default and a Gerry version.
- Versioning must follow this repository's own tag line (`v1.x` and onward); ignore inherited upstream `v4.x` tags from the fork source.
- When cutting a new release, update README quick-install download/extract commands to the new release URL/version.

Buildroot image workflow note:
- Branch: `master` (NEVER use `buildroot-prototype` — it is historical only and permanently diverged).
- Test unit: `root@piclockBR.local`. Build host: `pi@pi5start.local` (~/buildroot).
- Buildroot sources sdl-clock/alsa-ltc directly from `~/clock8002/v4` on pi5start — always `git pull --ff-only` in `~/clock8002` before any `make`.
- Before building, verify pi5start is on master: `ssh pi@pi5start.local 'cd ~/clock8002 && git branch --show-current'`
- Dev builds (with SSH key): `ssh pi@pi5start.local "cd ~/buildroot && BR2_PICLOCKKEY='$(cat ~/.ssh/id_rsa.pub)' make clock8002-dirclean && BR2_PICLOCKKEY='$(cat ~/.ssh/id_rsa.pub)' make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:\$?"`
- Release builds (no SSH key): `ssh pi@pi5start.local 'cd ~/buildroot && make clean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$?'`
- Monitor: `ssh pi@pi5start.local 'tail -f /tmp/br-build.log'`
- Always provide the monitor command after starting a build.
- Image transfer: `scp pi@pi5start.local:~/buildroot/output/images/sdcard.img ~/Desktop/piclockBR-<COMMIT>-sdcard.img`
- Image naming convention: `piclockBR-<7-char-commit-hash>-sdcard.img`
- Flash command format (user runs manually — never run dd from agent): `diskutil unmountDisk /dev/diskN && sudo dd if=/Users/jp/Desktop/piclockBR-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress && diskutil eject /dev/diskN`
- Always verify disk number with `diskutil list external physical` before giving flash commands.
- BusyBox on target: no bash (use `sh`), no `tar -z` (use `gzip -d -c | tar x`), no `--ignore-missing` on sha256sum.
- SSH to Buildroot target: `root@piclockBR.local` with `-o IdentitiesOnly=yes -i ~/.ssh/id_ed25519`.
- Deploy binary directly (no install.sh on BR): `systemctl stop clock8002 && cp <binary> /opt/clock8002/sdl-clock && systemctl start clock8002`.
- Mesa 25.0.7 changes on pi5start are NOT in git — must be re-applied after any clean Buildroot checkout on the build host.
- host-xz libtool bug workaround (`acl_cv_wl="-Wl,"` in xz.mk) is also NOT in git — must be re-applied after clean Buildroot checkout.
