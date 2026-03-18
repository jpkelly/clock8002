#!/usr/bin/env python3
import sys
import time
import os
import socket
import subprocess
import configparser
from luma.core.interface.serial import i2c
from luma.oled.device import ssd1306
from PIL import ImageDraw, ImageFont, Image

INI_PATH = os.path.expanduser('~/.config/clock-8001/clock.ini')
OLED_INI_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'oled.ini')
LOGO_PATH = os.path.expanduser('~/piclockLogo.bin')

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
font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 13)

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
    image = Image.frombytes("1", [OLED_WD, OLED_HT], bytes(tmplist))
    device.display(image)
    time.sleep(7)


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
        # Get IP address using hostname -I
        ip = subprocess.check_output(['hostname', '-I']).decode().split()[0]
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

while True:
    image = Image.new("1", device.size)
    draw = ImageDraw.Draw(image)
    hostname, ip, http_user, http_pass = get_stats()
    draw.text((0, 0), ip, font=font, fill=255)
    draw.text((0, 16), hostname, font=font, fill=255)
    draw.text((0, 32), f"User: {http_user}", font=font, fill=255)
    draw.text((0, 48), f"Pass: {http_pass}", font=font, fill=255)
    device.display(image)
    time.sleep(2)

