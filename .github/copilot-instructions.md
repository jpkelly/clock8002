Repository workflow note:
- Build machine: pi@cm5.local (CM5, 8GB RAM, NVMe — replaces pi5start.local)
- Test machine: pi@piclock.local
- Keep build machine current: before any build/release task on cm5.local, use a fresh clone at the target tag/branch or run `git fetch --tags origin` + `git pull --ff-only` in the working copy.
- Source-of-truth rule: for build/test/release on cm5.local, create a clean clone from `https://github.com/jpkelly/clock8002.git` (or verify the working copy matches that origin and target ref) before running commands.

Dev-deploy workflow note (feature branch testing, NOT a release):
- `install.sh` must be run on the TARGET machine (piclock) from a flat release directory — never from the source tree.
- The Makefile `release` target flattens all assets (including ttf_fonts/*.ttf) into a release dir and tarballs it.
- Reliability rules for remote deploy sessions:
  - Do not use `pkill -f /opt/clock8002/sdl3-clock` or `pkill -f /opt/clock8002/alsa-ltc` in one-shot SSH deploy commands; pattern matches can terminate the remote session and cause SSH exit 255.
  - Prefer `sudo systemctl stop ...` and `sudo systemctl kill ...` for process teardown.
  - Run installer as `sudo bash install.sh > /tmp/<install-log>.log 2>&1` and check `INSTALL_EXIT` before restarting services.
  - After installer completion, explicitly start and verify `clock8002`, `alsa-ltc`, and `oled_daemon` service state.
- Steps:
  1. Clone branch and build on cm5: `ssh pi@cm5.local 'cd /tmp && rm -rf clock8002-build && git clone --depth 1 --branch BRANCH https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && make release NETWORK_CONFIG=default'`
  2. Relay tarball to piclock (note: GIT_TAG defaults to nearest ancestor tag, e.g. v1.0.3): `scp pi@cm5.local:/tmp/clock8002-build/v4/clock8002-*-default-linux-arm64.tar.gz /tmp/ && scp /tmp/clock8002-*-default-linux-arm64.tar.gz pi@piclock.local:/tmp/`
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
- Approve code change only if leak behavior is reproduced on that unit (e.g., materially rising `VmSwap` for `sdl3-clock` or sustained RSS growth over time). If metrics are flat by 24h, hold changes and treat prior 1GB findings as non-generalized.

Branch safety note:
- Before editing any repository file, always verify the local branch: `git branch --show-current`. Confirm it matches the intended branch before proceeding.
- Before any `git commit`, verify the branch again. Never commit to `master` when changes belong on a feature branch, or vice versa.
- `master` = production-ready Buildroot image code only. All experimental/in-progress work lives on a feature branch.
- Active feature branch: `feature/squashfs-readonly` — owns: rootfs-overlay fstab, S02setup-root, launcher script relocation, /root tmpfs changes. Do NOT merge to master until clean-boot image test passes.
- For a feature-branch build on cm5: (1) switch cm5 to the feature branch, (2) build, (3) switch cm5 back to master immediately after. Always restore cm5 to master — leaving it on a feature branch will corrupt the next master build.
- `buildroot-prototype` branch: historical only — never check it out, never build from it, never commit to it.

Release management note:
- The primary release artifact is the **Buildroot SD card image** (`piClock-<commit>-sdcard.img`), built via the Buildroot system on cm5 (`~/buildroot`). This is what gets attached to GitHub releases.
- The `make release` tarballs are a legacy Trixie mechanism; do not build or attach them for Buildroot releases.
- Versioning must follow this repository's own tag line (`v1.x` and onward); ignore inherited upstream `v4.x` tags from the fork source.
- When cutting a new release, update README quick-install download/extract commands to the new release URL/version.
- When tagging a release, always update HANDOFF.md to reflect the new latest tag, commit hash, and any relevant status changes before or as part of the release commit.

Buildroot image workflow note:
- Branch: `master` (NEVER use `buildroot-prototype` — it is historical only and permanently diverged).
- Test unit: `root@piClock.local`. Build host: `pi@cm5.local` (~/buildroot).
- Buildroot sources sdl3-clock/alsa-ltc directly from `~/clock8002/v4` on cm5 — always `git pull --ff-only` in `~/clock8002` before any `make`.
- Before building, verify cm5 is on the **intended branch**: `ssh pi@cm5.local 'cd ~/clock8002 && git branch --show-current'` — expected output is `master` for release/production builds, or the feature branch name for feature-branch builds. Fail if the output does not match.
- MANDATORY: always start cm5 builds inside a `screen` session. Never start a long Buildroot `make` directly in a one-shot SSH command.
- Dual service file rule: service files that exist in both `v4/` (Trixie) and `buildroot-external/board/clock8002-rpi5/rootfs-overlay/` (Buildroot) must be kept in sync. When editing a service file in `v4/`, always check for a Buildroot overlay copy and update it with the same changes (adjusting for platform differences like `User=root`). The overlay overwrites the package-installed copy at image build time.
- Dev builds (with SSH key): `ssh pi@cm5.local 'cd ~/buildroot && screen -S br-build -dm bash -lc "BR2_PICLOCKKEY='"'"'$(cat ~/.ssh/id_rsa.pub)'"'"' make clock8002-dirclean && BR2_PICLOCKKEY='"'"'$(cat ~/.ssh/id_rsa.pub)'"'"' make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$? > /tmp/br-build.exit"'`
- Release builds (no SSH key): `ssh pi@cm5.local 'cd ~/buildroot && screen -S br-build -dm bash -lc "make clean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$? > /tmp/br-build.exit"'`
- feature/root-ram builds (custom output dir): always include `clock8002-dirclean` before the main `make` to force rebuild of Go binaries and oled-daemon. Without it, incremental builds reuse the cached clock8002 package and changes to oled_daemon.py or Go source are silently skipped. Pattern: `screen -S br-build-<commit> -dm bash -lc "{ CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-root-ram-payload-20260509-165344 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot clock8002-dirclean && CLOCK8002_PREBUILT_KERNEL=1 make O=/home/pi/output-root-ram-payload-20260509-165344 BR2_EXTERNAL=/home/pi/clock8002-root-ram/buildroot-external -C /home/pi/buildroot BR2_PICLOCKKEY=\"$(cat ~/.ssh/id_rsa.pub)\"; } > /tmp/br-build-<commit>.log 2>&1; echo BR_BUILD_EXIT:\$? > /tmp/br-build-<commit>.exit"`
- CRITICAL: NEVER run builds with `&` (background) and immediately switch the cm5 branch. Buildroot rsyncs source during `make` — switching to master before rsync runs causes master's source to be built instead of the feature branch. Always run builds synchronously and only `git checkout master` AFTER the build command returns.
- Monitor: `ssh pi@cm5.local 'screen -ls; tail -f /tmp/br-build.log'`
- Always provide the monitor command after starting a build.
- Image transfer: `scp pi@cm5.local:~/buildroot/output/images/sdcard.img ~/Desktop/piClock-<COMMIT>-sdcard.img`
- Image naming convention: `piClock-<7-char-commit-hash>-sdcard.img`
- Flash command format (user runs manually — never run dd from agent): `diskutil unmountDisk /dev/diskN && sudo dd if=/Users/jp/Desktop/piClock-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress && diskutil eject /dev/diskN`
- IMPORTANT: always use the absolute path `/Users/jp/Desktop/` in `dd if=` — never `~` or `$HOME`, as `sudo dd` does not expand tilde.
- Always verify disk number with `diskutil list external physical` before giving flash commands.
- BusyBox on target: no bash (use `sh`), no `tar -z` (use `gzip -d -c | tar x`), no `--ignore-missing` on sha256sum.
- SSH to Buildroot target: `root@piClock.local` with `-o IdentitiesOnly=yes -i ~/.ssh/id_rsa`.
- VPN fallback IPs (use when .local mDNS fails): cm5=`10.0.0.101`, piClock=`192.168.8.246`. SSH to piClock over VPN: `ssh -o IdentitiesOnly=yes -i ~/.ssh/id_rsa root@192.168.8.246`
- Deploy binary directly (no install.sh on BR): `/etc/init.d/S99clock stop && cp <binary> /opt/clock8002/sdl3-clock && /etc/init.d/S99clock start`. WARNING: on squashfs images (`feature/squashfs-readonly` and `master` after merge), `/opt/clock8002` is read-only squashfs — `cp` will fail. Dev deploys require a full reflash on squashfs images.
- After a fresh Buildroot checkout, always run `buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot` before building — this applies the Mesa 25.0.7 upgrade and host-xz libtool workaround (see issue #29).

Terminal reliability note (avoid VS Code zsh exit 128/255 during orchestration):
- For local orchestration wrappers (especially ssh/git/build), default to a no-crash pattern:
  - Do not use top-level `set -e`.
  - Track failures with an explicit non-special variable such as `cmd_status` or `rc`.
  - Capture step logs in `/tmp/<name>.log` and print `*_STATUS` markers.
  - End the wrapper with `exit 0` after reporting the result so the interactive shell session stays alive.
- Prefer path-explicit git commands (`git -C <repo> ...`) or guarded `cd <repo> || cmd_status=1`; do not rely on inherited cwd in long chained commands.
- Before checking out a branch on cm5, check for worktree conflicts using `git -C ~/clock8002 worktree list`.
  - If the target branch is already attached elsewhere, operate in that existing worktree instead of forcing another checkout.
- For long remote operations, always use unique session/log/exit files (for example `/tmp/<session>.log` and `/tmp/<session>.exit`) and provide explicit monitor + exit-check commands.
- For diagnostic probes where failure is acceptable (missing files, no screen session, optional logs), guard commands with `|| true` so probes do not abort orchestration.
- For remote status/probe commands that include regex, globs, or shell variables, do not use inline quoted `ssh "..."` one-liners. Use a heredoc (`ssh <host> 'sh' <<'REMOTE' ...`) or a checked-in helper script (for example `tools/buildroot/cm5-build-status.sh`) to avoid local zsh expansion/quoting breakage.

Codebase orientation note:
- Active code is in `v4/` only. `v3/` is historical — never edit it.
- Go module: `gitlab.com/clock-8001/clock-8001/v4` (Go 1.24). Build: `cd v4 && make build`. Test: `make test` (runs `go test -short ./clock`).
- Key source dirs: `v4/clock/` (engine, OSC dispatch, integrations), `v4/cmd/sdl3-clock/` (entry point, config loading, SDL3 render loop).
- Config on target: `/root/.config/clock-8001/clock.ini` is a symlink → `/boot/piclock/clock.ini` (FAT). Always edit `/boot/piclock/clock.ini` — never the squashfs-baked symlink target.
