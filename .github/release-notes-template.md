# Clock8002 __VERSION__

## What Changed

- Update summary goes here.

## Install (Default)

```bash
wget https://github.com/jpkelly/clock8002/releases/download/__VERSION__/piClock-__VERSION__-linux-arm64.tar.gz
tar xzf piClock-__VERSION__-linux-arm64.tar.gz
cd piClock-__VERSION__-linux-arm64
sudo bash install.sh
```

## Verify

```bash
systemctl is-active clock8002 alsa-ltc oled_daemon
```

## Notes

- Built for Raspberry Pi 5, arm64.
- Single release variant (`default`); the `gerry` variant was removed in `v1.4.0`.
