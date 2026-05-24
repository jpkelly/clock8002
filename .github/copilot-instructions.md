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

Build reproducibility docs:
- Policy: docs/build-policy.md
- Manifest template: docs/build-manifest-template.md

Operational defaults:
- Prebuilt kernel is default unless explicitly overridden.
- Always use fresh output directory for each build.
- Always provide monitor and exit-check commands after starting a build.
- Keep terminal wrappers no-crash: avoid top-level `set -e`, use explicit `cmd_status`/`rc`, unique `/tmp/<session>.log` + `/tmp/<session>.exit`.

Codebase orientation:
- Active code: v4/ only (v3/ is historical)
- Go module: gitlab.com/clock-8001/clock-8001/v4
- Target config path: /boot/piclock/clock.ini (never rely on baked symlink target)
