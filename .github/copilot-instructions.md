Platform status (updated 2026-08-05):
- **Raspberry Pi OS Trixie is the ONLY platform.** All releases ship from it.
- **Buildroot has been removed/diverged** (2026-08-05). The `buildroot` branch was deleted; its history is preserved by the `buildroot-final` tag. Do not attempt Buildroot builds or reference Buildroot files/tools — they no longer exist in this repo.
- Latest release candidate: `v1.4.1-rc3` (2026-08-05), built from `v4/` on `master`. Install validated on a fresh Trixie reflash.
- Last full release: `v1.4.0`.

Branch layout:
- `master` — the only active branch. Production-ready Trixie code, default branch, source of all releases.
- Historical, do not check out, build, or commit to: `archive/trixie-sdl2`, `trixie-sdl3`, `buildroot-prototype`, `feature/root-ram-build-a`, `feature/squashfs-readonly`. The `buildroot` branch was deleted (tagged `buildroot-final`).
- All experimental/in-progress work lives on a feature branch, never directly on `master`.

Machines:
- Primary test unit: piClock.local (192.168.8.245) — log in as `pi` (`ssh pi@piClock.local`). The `root` account is locked and cannot be used with a password; use sudo.
- Build/release machine: pi@cm5.local (CM5, 8GB RAM, NVMe) — Trixie release builder. Fall back to pi@pi5start.local when cm5 is unavailable.
- Reference/research unit only: 192.168.8.246 — never use as a fallback build or deploy target.

Critical guardrails:
- Before editing files: verify local branch with `git branch --show-current`.
- Before commit: verify local branch again. Never commit to `master` when changes belong on a feature branch, or vice versa.
- For remote build orchestration, use SSH heredoc form (`ssh host 'sh' <<'REMOTE' ... REMOTE`) by default; avoid inline quoted SSH one-liners except trivial single-literal commands. When a command needs multiple quotes/heredocs, write the file locally and `scp` it rather than nesting heredocs inside SSH.
- If running tests, ask first.

Codebase orientation:
- Active code: `app/` only. It is the sole Go module and the single source of the product.
- The Go module path is `github.com/jpkelly/clock8002/app` (rehomed from the upstream `gitlab.com/clock-8001/clock-8001/v4`). The old `v4/` directory was renamed to `app/`; the `v4` in the upstream path was the Go module major-version, unrelated to this repository's `v1.x` tag line.
- The root-level Go packages, `v3/`, and the duplicate root `fonts/`/`arduino/` were removed (2026-08-05) as dead upstream code. Do not expect them to exist.
- Config path (Trixie): `/boot/firmware/piclock/`. The old `/boot/piclock/` Buildroot path is gone.

Stability decision gate note:
- When investigating suspected memory/resource leaks, do not deploy code fixes until the current baseline run on the fully-populated 2GB piclock unit reaches at least the 24h checkpoint, unless the user explicitly overrides this gate.
- Required decision checkpoints: 12h and 24h with the same metrics (`VmRSS`, `VmSwap`, system swap, service state, temperature/throttle).
- Approve code change only if leak behavior is reproduced on that unit (e.g., materially rising `VmSwap` for `sdl3-clock` or sustained RSS growth over time). If metrics are flat by 24h, hold changes and treat prior 1GB findings as non-generalized.

Release management note:
- The primary release artifact is the **installer tarball** (`piClock-v1.x.y-linux-arm64.tar.gz`), produced by `make release` in `app/`. This is what gets attached to GitHub releases.
- Tag format: `v1.x.y`, starting with `v1.3.16`. Releases before that used `trixie-v1.x.y` (e.g. `trixie-v1.3.15`) — those existing tags are historical and are not renamed.
- Trixie builder: `pi@cm5.local` is preferred; fall back to `pi@pi5start.local` when cm5 is unavailable. Record which host was used in the release notes.
- Builder prerequisite: the release step copies `~/sdl3-build/sdl3-trixie-lib` into `app/lib`. As of 2026-08-02 that bundle exists on both cm5 and pi5start. Verify the path exists before starting a build on either host.
- There is a single release variant (`default`). The `gerry` variant was dropped in `v1.4.0`.
- Full procedure: see RELEASING.md § Trixie Installer Release.
- Versioning must follow this repository's own tag line (`v1.x` and onward); ignore inherited upstream `v4.x` tags from the fork source.
- When cutting a new release, update README quick-install download/extract commands to the new release URL/version.
- When tagging a release, always update HANDOFF.md to reflect the new latest tag, commit hash, and any relevant status changes before or as part of the release commit.
