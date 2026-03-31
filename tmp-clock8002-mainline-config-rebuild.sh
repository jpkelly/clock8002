#!/usr/bin/env bash
set -euo pipefail

cd /Users/jp/Documents/GitHub/clock8002
sh -n buildroot-external/board/clock8002-rpi5/post-image.sh

git add buildroot-external/board/clock8002-rpi5/post-image.sh
git commit -m "Buildroot Pi5: keep generated config/cmdline from firmware settings" || true
git push origin buildroot-prototype

ssh pi@pi5start.local '
set -euo pipefail
cd /home/pi/clock8002
git fetch --tags origin
git checkout buildroot-prototype
git pull --ff-only
echo DEPLOY_HASH:$(git rev-parse --short HEAD)

cd /home/pi/buildroot
cp -f /home/pi/clock8002/buildroot-external/configs/clock8002_rpi5_defconfig.sample configs/clock8002_rpi5_defconfig
make BR2_EXTERNAL=/home/pi/clock8002/buildroot-external clock8002_rpi5_defconfig
make BR2_EXTERNAL=/home/pi/clock8002/buildroot-external > /home/pi/buildroot-clock8002.log 2>&1

echo BUILD_EXIT:$?
echo ---
echo "images/config.txt"; sed -n "1,40p" /home/pi/buildroot/output/images/config.txt || true
echo ---
echo "images/cmdline.txt"; sed -n "1,20p" /home/pi/buildroot/output/images/cmdline.txt || true
echo ---
ls -lh /home/pi/buildroot/output/images/sdcard.img
echo ---
tail -n 40 /home/pi/buildroot-clock8002.log
'
