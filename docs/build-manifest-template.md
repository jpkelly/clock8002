# Build Manifest Template

Use this file for every candidate build. Fill all fields.

## Identity
- Build name:
- Date/time (UTC):
- Build host:
- Operator:

## Source Inputs
- Source repo path:
- Source repo branch:
- Source repo commit (full SHA):
- External repo path:
- External repo branch:
- External repo commit (full SHA):

## Kernel Inputs
- Prebuilt kernel enabled (`0|1`):
- Kernel bundle absolute path:
- Kernel bundle hash (optional but recommended):

## Build Command
- Defconfig command:
- Dirclean command:
- Main make command:
- Output directory:
- Screen session:
- Log file:
- Exit file:

## Output Artifacts
- `sdcard.img` path:
- `sdcard.img` sha256:
- `Image` sha256:
- `rootfs.cpio.gz` sha256:

## Runtime Binary Hashes (from built image or running unit)
- `sdl-clock` sha256:
- `alsa-ltc` sha256:
- `setup.sh` sha256:
- `config.txt` sha256:
- `cmdline.txt` sha256:

## Validation Results
- Boot status:
- Services status (`clock8002`, `alsa-ltc`, `oled-daemon`):
- LTC status:
- Network/serial notes:
- Tester:
- Test date/time:

## Verdict
- Classification: `candidate | known-good | rejected`
- Notes:
