Repository workflow note:
- Build machine: pi@pi5start.local
- Test machine: pi@piclock.local
- Keep build machine current: before any build/release task on pi5start.local, use a fresh clone at the target tag/branch or run `git fetch --tags origin` + `git pull --ff-only` in the working copy.
- Source-of-truth rule: for build/test/release on pi5start.local, create a clean clone from `https://github.com/jpkelly/clock8002.git` (or verify the working copy matches that origin and target ref) before running commands.

Dev-deploy workflow note (feature branch testing, NOT a release):
- `install.sh` must be run on the TARGET machine (piclock) from a flat release directory — never from the source tree.
- The Makefile `release` target flattens all assets (including ttf_fonts/*.ttf) into a release dir and tarballs it.
- Steps:
  1. Clone branch and build on pi5start: `ssh pi@pi5start.local 'cd /tmp && rm -rf clock8002-build && git clone --depth 1 --branch BRANCH https://github.com/jpkelly/clock8002.git clock8002-build && cd clock8002-build/v4 && make release NETWORK_CONFIG=default'`
  2. Relay tarball to piclock (note: GIT_TAG defaults to nearest ancestor tag, e.g. v1.0.3): `scp pi@pi5start.local:/tmp/clock8002-build/v4/clock8002-*-default-linux-arm64.tar.gz /tmp/ && scp /tmp/clock8002-*-default-linux-arm64.tar.gz pi@piclock.local:/tmp/`
  3. Install on piclock: `ssh pi@piclock.local 'cd /tmp && tar xzf clock8002-*-default-linux-arm64.tar.gz && cd clock8002-*-default-linux-arm64 && sudo bash install.sh'`
- After deploying to the test unit, always report the short GitHub commit hash (first 7 characters) that was deployed.

Issue management note:
- From now on I’ll only close an issue if you explicitly tell me to close it.

Test management note:
- If you are going to run a test, ask first and then I will decide whether or not you should run the test.

Release management note:
- For each new release, be sure to include both a default and a Gerry version.
- Versioning must follow this repository's own tag line (`v1.x` and onward); ignore inherited upstream `v4.x` tags from the fork source.
- When cutting a new release, update README quick-install download/extract commands to the new release URL/version.
