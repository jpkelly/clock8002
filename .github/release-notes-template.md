# Clock8002 __VERSION__

## What Changed

- Update summary goes here.

## Install (Default)

```bash
wget https://github.com/jpkelly/clock8002/releases/download/__VERSION__/clock8002-__VERSION__-default-linux-arm64.tar.gz
tar xzf clock8002-__VERSION__-default-linux-arm64.tar.gz
cd clock8002-__VERSION__-default-linux-arm64
sudo bash install.sh
```

## Install (Gerry)

```bash
wget https://github.com/jpkelly/clock8002/releases/download/__VERSION__/clock8002-__VERSION__-gerry-linux-arm64.tar.gz
tar xzf clock8002-__VERSION__-gerry-linux-arm64.tar.gz
cd clock8002-__VERSION__-gerry-linux-arm64
sudo bash install.sh
```

## Verify

```bash
systemctl is-active clock8002 alsa-ltc oled_daemon
```

## Notes

- Built for Raspberry Pi 5, arm64.
- Includes both default and gerry release variants.
