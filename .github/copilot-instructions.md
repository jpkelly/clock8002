Repository workflow note:
- Build machine: pi@pi5start.local
- Test machine: pi@piclock.local
- Keep build machine current: before any build/release task on pi5start.local, use a fresh clone at the target tag/branch or run `git fetch --tags origin` + `git pull --ff-only` in the working copy.
- Source-of-truth rule: for build/test/release on pi5start.local, create a clean clone from `https://github.com/jpkelly/clock8002.git` (or verify the working copy matches that origin and target ref) before running commands.

Issue management note:
- From now on I’ll only close an issue if you explicitly tell me to close it.

Test management note:
- If you are going to run a test, ask first and then I will decide whether or not you should run the test.

Release management note:
- For each new release, be sure to include both a default and a Gerry version.
- Versioning must follow this repository's own tag line (`v1.x` and onward); ignore inherited upstream `v4.x` tags from the fork source.
- When cutting a new release, update README quick-install download/extract commands to the new release URL/version.
