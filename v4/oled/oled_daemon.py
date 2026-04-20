#!/usr/bin/env python3
import sys
import time
import os
import socket
import subprocess
import configparser
import re
from luma.core.interface.serial import i2c
from luma.oled.device import ssd1306
from PIL import ImageDraw, ImageFont, Image

_INI_CANDIDATES = [
    os.path.expanduser('~/.config/clock-8001/clock.ini'),  # Trixie / dev
    '/boot/piclock/clock.ini',                              # Buildroot
]
INI_PATH = next((p for p in _INI_CANDIDATES if os.path.exists(p)), _INI_CANDIDATES[0])
OLED_INI_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'oled.ini')
LOGO_PATH = os.path.expanduser('~/piclockLogo.bin')
SDL_CLOCK_PATH = '/opt/clock8002/sdl3-clock'
LOGO_SECONDS = 8

# OLED hardware defaults
OLED_ENABLED = True
OLED_I2C_PORT = 1
OLED_I2C_ADDRESS = 0x3c
OLED_ROTATION = 2

# Read OLED config
if os.path.exists(OLED_INI_PATH):
    _cfg = configparser.ConfigParser()
    _cfg.read(OLED_INI_PATH)
    if _cfg.has_section('oled'):
        OLED_ENABLED = _cfg.getboolean('oled', 'enabled', fallback=OLED_ENABLED)
        OLED_I2C_PORT = _cfg.getint('oled', 'i2c_port', fallback=OLED_I2C_PORT)
        _addr = _cfg.get('oled', 'i2c_address', fallback=None)
        if _addr:
            OLED_I2C_ADDRESS = int(_addr, 16)
        OLED_ROTATION = _cfg.getint('oled', 'rotation', fallback=OLED_ROTATION)

if not OLED_ENABLED:
    sys.exit(0)

serial = i2c(port=OLED_I2C_PORT, address=OLED_I2C_ADDRESS)
device = ssd1306(serial, rotate=OLED_ROTATION)

import signal as _signal
def _blank_and_exit(signum=None, frame=None):
    try:
        device.display(Image.new("1", device.size))
    except Exception:
        pass
    sys.exit(0)
_signal.signal(_signal.SIGTERM, _blank_and_exit)
_signal.signal(_signal.SIGINT, _blank_and_exit)

_FONT_CANDIDATES = [
    "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",  # Debian/Trixie
    "/opt/clock8002/DejaVuSans.ttf",                     # Buildroot (bundled)
]
_FONT_PATH = next((p for p in _FONT_CANDIDATES if os.path.exists(p)), None)

font = ImageFont.truetype(_FONT_PATH, 13) if _FONT_PATH else ImageFont.load_default()
try:
    logo_font = ImageFont.truetype(_FONT_PATH, 10) if _FONT_PATH else ImageFont.load_default()
except Exception:
    logo_font = ImageFont.load_default()

# Direct port of Argon logo display logic
OLED_WD = device.bounding_box[2] + 1
OLED_HT = device.bounding_box[3] + 1
OLED_BUFFERIZE = ((OLED_WD * OLED_HT) >> 3)

def load_logo_buffer(path):
    buf = [0] * OLED_BUFFERIZE
    if os.path.exists(path):
        with open(path, 'rb') as f:
            bgbytes = list(f.read())
        ctr = len(bgbytes)
        if ctr == OLED_BUFFERIZE:
            buf[:] = bgbytes
        elif ctr > OLED_BUFFERIZE:
            buf[:] = bgbytes[0:OLED_BUFFERIZE]
        else:
            buf[0:ctr] = bgbytes
            while ctr < OLED_BUFFERIZE:
                buf[ctr] = 0
                ctr += 1
    return buf

def flush_logo_buffer(buf):
    tmplist = [0] * OLED_BUFFERIZE
    srcidx = 0
    while srcidx < OLED_BUFFERIZE:
        outyoffset = 0
        yidx = 0
        ymask = 1
        while yidx < 8:
            outoffsetidx = 0
            outbyte = 0
            xmask = 0x80
            xidx = 0
            xoffset = 0
            while xoffset < OLED_WD:
                if buf[srcidx + xoffset] & ymask:
                    outbyte = outbyte | xmask
                xmask = xmask >> 1
                xidx = xidx + 1
                if xidx >= 8:
                    tmplist[srcidx + outoffsetidx + outyoffset] = outbyte
                    xmask = 0x80
                    xidx = 0
                    outbyte = 0
                    outoffsetidx = outoffsetidx + 1
                xoffset = xoffset + 1
            outyoffset = outyoffset + (OLED_WD >> 3)
            yidx = yidx + 1
            ymask = ymask << 1
        srcidx = srcidx + OLED_WD
    # Write logo pixels directly into a fresh mode-1 image. This avoids
    # Image.frombytes("1",...) whose packed-bit interpretation varies across
    # Pillow versions, and avoids frombytes("L",...).convert("1") which can
    # produce an all-black image on some Pillow 11 / luma.oled 3.15 builds.
    image = Image.new("1", (OLED_WD, OLED_HT))
    pix = image.load()
    bytes_per_row = OLED_WD // 8
    for byte_idx, byte in enumerate(tmplist):
        if byte == 0:
            continue
        row = byte_idx // bytes_per_row
        col_base = (byte_idx % bytes_per_row) * 8
        for b in range(8):
            if byte & (0x80 >> b):
                pix[col_base + b, row] = 1
    draw = ImageDraw.Draw(image)
    version_text = get_build_version()
    if version_text:
        # Keep version readable over the logo with a small black backing box.
        try:
            l, t, r, b = draw.textbbox((0, 0), version_text, font=logo_font)
            text_w = r - l
            text_h = b - t
        except Exception:
            text_w, text_h = draw.textsize(version_text, font=logo_font)
        x = max(0, OLED_WD - text_w - 2)
        y = max(0, OLED_HT - text_h - 6)
        draw.rectangle((x - 1, y - 1, OLED_WD - 1, OLED_HT - 1), fill=0)
        draw.text((x, y), version_text, font=logo_font, fill=255)
    device.display(image)
    time.sleep(LOGO_SECONDS)


def get_build_version():
    # Primary source: parse embedded build ldflags from the binary itself.
    # Use grep -m1 so command exits quickly after first match.
    try:
        out = subprocess.check_output(
            [
                'sh',
                '-c',
                r"strings /opt/clock8002/sdl3-clock | grep -m1 -oE 'clock\.gitTag=v[0-9]+\.[0-9]+\.[0-9]+'"
            ],
            stderr=subprocess.STDOUT,
            timeout=1.0,
        )
        text = out.decode(errors='ignore').strip()
        match = re.search(r'clock\.gitTag=(v[0-9]+\.\d+\.\d+)', text)
        if match:
            return match.group(1)
    except Exception:
        pass

    # Fallback for builds that support --version directly.
    try:
        out = subprocess.check_output([SDL_CLOCK_PATH, '--version'], stderr=subprocess.STDOUT, timeout=1.5)
        text = out.decode(errors='ignore').strip()
    except Exception:
        return ''

    match = re.search(r'v[0-9]+\.\d+\.\d+', text)
    if match:
        return match.group(0)

    first_line = text.splitlines()[0] if text else ''
    return first_line[:18]


def parse_ini_settings():
    settings = {'HTTPPort': '', 'HTTPUser': '', 'HTTPPassword': ''}
    if os.path.exists(INI_PATH):
        try:
            config = configparser.ConfigParser()
            config.read(INI_PATH)
            for section in config.sections():
                for key in settings:
                    if config.has_option(section, key):
                        settings[key] = config.get(section, key)
        except configparser.MissingSectionHeaderError:
            # Flat INI format (no section headers)
            with open(INI_PATH, 'r') as f:
                for line in f:
                    line = line.strip()
                    for key in settings:
                        if line.startswith(f'{key}='):
                            settings[key] = line.split('=', 1)[1]
    return settings

def get_ip():
    try:
        out = subprocess.check_output(
            ['ip', '-4', '-o', 'addr', 'show', 'scope', 'global'],
            stderr=subprocess.DEVNULL).decode()
        ip = out.split('inet ')[1].split('/')[0]
    except Exception:
        ip = 'No IP'
    return ip

def get_stats():
    settings = parse_ini_settings()
    port = settings['HTTPPort']
    http_user = settings['HTTPUser']
    http_pass = settings['HTTPPassword']
    hostname = socket.gethostname() + '.local'
    if port:
        hostname = f"{hostname}:{port.lstrip(':')}"
    ip = get_ip()
    return hostname, ip, http_user, http_pass

# Show logo at startup
logo_buf = load_logo_buffer(LOGO_PATH)
flush_logo_buffer(logo_buf)

def is_buildroot():
    try:
        with open('/etc/os-release') as f:
            return any(line.strip() in ('NAME=Buildroot', 'NAME="Buildroot"') for line in f)
    except Exception:
        return False

BUILDROOT = is_buildroot()

while True:
    image = Image.new("1", device.size)
    draw = ImageDraw.Draw(image)
    hostname, ip, http_user, http_pass = get_stats()
    draw.text((0, 0), ip, font=font, fill=255)
    draw.text((0, 16), hostname, font=font, fill=255)
    draw.text((0, 32), f"User: {http_user}", font=font, fill=255)
    draw.text((0, 48), f"Pass: {http_pass}", font=font, fill=255)
    if BUILDROOT:
        draw.ellipse((OLED_WD - 5, 2, OLED_WD - 2, 5), fill=255)
    device.display(image)
    time.sleep(2)

