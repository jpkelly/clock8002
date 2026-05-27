Repository workflow note:
- Build machine: pi@cm5.local (CM5, 8GB RAM, NVMe)
- Primary test machine: root@piClock.local (192.168.8.245)
- Reference/research unit only: root@192.168.8.246
- Do not use .246 as a fallback target for normal build validation or deployment.

Active build modes:
- Dev/RC mode:
  - Branch: feature/root-ram
  - Source path: /home/pi/clock8002-root-ram
  - Buildroot external: /home/pi/clock8002-root-ram/buildroot-external
- Release mode:
  - Branch: master
  - Use clean clone or ref-verified working copy at the target release ref

Critical guardrails:
- Never mix source inputs from multiple repos in one build.
- Always build in a screen session and use unique log/exit files.
- Never switch branch/ref in the active source tree until build completion is confirmed by exit marker.
- Treat image filename commit text as label only; provenance requires manifest + hashes.
- Before editing files: verify local branch with `git branch --show-current`.
- Before commit: verify local branch again.
- If running tests, ask first.
- For remote build orchestration, use SSH heredoc form (`ssh host 'sh' <<'REMOTE' ... REMOTE`) by default; avoid inline quoted SSH one-liners except trivial single-literal commands.
- Do not use ad-hoc build wrapper scripts for launch orchestration. Use direct Buildroot commands in documented sequence.
- Before starting any build, run a preflight check and print results: current branch, active screen sessions, bundle path existence, and exact make targets.
- If preflight fails or runtime behavior is unexpected, stop immediately and ask before retrying.
- Before every build, run `tools/buildroot/manifest-preflight.sh <output_dir>` (on cm5) to determine the correct build action. Follow its recommendation unless explicitly overridden. See docs/build-policy.md § Output Directory State Tracking.
- Before launching any build (after preflight and make clean/prep), run `tools/buildroot/manifest-snapshot.sh --start <output_dir> --src ~/clock8002-root-ram --br ~/buildroot --target "<make_target>"` to record the build start state.
- After a build completes (success or failure), run `tools/buildroot/manifest-snapshot.sh --finish <output_dir> <exit_code>` to record the final state.
- Before any `make <pkg>-dirclean`, run `tools/buildroot/manifest-record-dirclean.sh <output_dir> <pkg>` to append the event to the manifest.

Build reproducibility docs:
- Policy: docs/build-policy.md
- Manifest template: docs/build-manifest-template.md

- Operational defaults:
- Prebuilt kernel is default unless explicitly overridden.
- Always use fresh output directory for release builds; use manifest-preflight.sh to determine if an incremental build is safe for dev/RC builds.
- Always provide monitor and exit-check commands after starting a build.
- Keep terminal wrappers no-crash: avoid top-level `set -e`, use explicit `cmd_status`/`rc`, unique `/tmp/<session>.log` + `/tmp/<session>.exit`.

Codebase orientation:
- Active code: v4/ only (v3/ is historical)
- Go module: gitlab.com/clock-8001/clock-8001/v4
- Target config path: /boot/piclock/clock.ini (never rely on baked symlink target)

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
- Buildroot sources sdl3-clock/alsa-ltc directly from `~/clock8002-root-ram/v4` on cm5 — always `git pull --ff-only` in `~/clock8002-root-ram` before any `make`. Do NOT pull `~/clock8002`; that directory is not used by Buildroot.
- Before building, verify cm5 is on the **intended branch** and at the **intended commit**: `ssh pi@cm5.local 'cd ~/clock8002-root-ram && git branch --show-current && git log --oneline -1'` — expected branch is `master` for release/production builds, or the feature branch name for feature-branch builds. Fail if the output does not match. The commit hash shown here is what will be compiled into the image — confirm it matches the intended HEAD before proceeding.
- Dual service file rule: service files that exist in both `v4/` (Trixie) and `buildroot-external/board/clock8002-rpi5/rootfs-overlay/` (Buildroot) must be kept in sync. When editing a service file in `v4/`, always check for a Buildroot overlay copy and update it with the same changes (adjusting for platform differences like `User=root`). The overlay overwrites the package-installed copy at image build time.
- Dev builds (with SSH key): `ssh pi@cm5.local "cd ~/buildroot && BR2_PICLOCKKEY='$(cat ~/.ssh/id_rsa.pub)' make clock8002-dirclean && BR2_PICLOCKKEY='$(cat ~/.ssh/id_rsa.pub)' make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:\$?"`
- Release builds (no SSH key): `ssh pi@cm5.local 'cd ~/buildroot && make clean && make > /tmp/br-build.log 2>&1; echo BR_BUILD_EXIT:$?'`
- Monitor: `ssh pi@cm5.local 'tail -f /tmp/br-build.log'`
- Always provide the monitor command after starting a build.
- Image transfer: `scp pi@cm5.local:~/buildroot/output/images/sdcard.img ~/Desktop/piClock-<COMMIT>-sdcard.img`
- Image naming convention: `piClock-<7-char-commit-hash>-sdcard.img`
- Flash command format (user runs manually — never run dd from agent): `diskutil unmountDisk /dev/diskN && sudo dd if=/Users/jp/Desktop/piClock-<COMMIT>-sdcard.img of=/dev/rdiskN bs=4m status=progress && diskutil eject /dev/diskN`
- Always verify disk number with `diskutil list external physical` before giving flash commands.
- BusyBox on target: no bash (use `sh`), no `tar -z` (use `gzip -d -c | tar x`), no `--ignore-missing` on sha256sum.
- SSH to Buildroot target: `root@piClock.local` with `-o IdentitiesOnly=yes -i ~/.ssh/id_rsa`.
- Deploy binary directly (no install.sh on BR): `/etc/init.d/S99clock stop && cp <binary> /root/sdl-clock && /etc/init.d/S99clock start`. (The running process is `sdl-clock`; `sdl3-clock` is the build-target name but `clock_cmd.sh` exec's `/root/sdl-clock`.)
- After a fresh Buildroot checkout, always run `buildroot-external/scripts/apply-build-host-patches.sh ~/buildroot` before building — this applies the Mesa 25.0.7 upgrade and host-xz libtool workaround (see issue #29).
