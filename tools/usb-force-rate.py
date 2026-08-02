#!/usr/bin/env python3
"""Force the capture endpoint's sample rate via a raw usbfs control transfer.

Bypasses ALSA entirely so the rate can be changed while alsa-ltc still holds the
PCM open. This issues the UAC1 SET_CUR SAMPLING_FREQ_CONTROL request to endpoint
0x82 - the same request that reports "cannot set freq 44100 to ep 0x82" when the
controller wedges.
"""
import ctypes
import fcntl
import os
import sys
import time

USBDEVFS_CONTROL = 0xC0185500
SET_CUR = 0x01
GET_CUR = 0x81
SAMPLING_FREQ_CONTROL = 0x01


class CtrlTransfer(ctypes.Structure):
    _fields_ = [
        ('bRequestType', ctypes.c_uint8),
        ('bRequest', ctypes.c_uint8),
        ('wValue', ctypes.c_uint16),
        ('wIndex', ctypes.c_uint16),
        ('wLength', ctypes.c_uint16),
        ('timeout', ctypes.c_uint32),
        ('data', ctypes.c_void_p),
    ]


def control(fd, req_type, request, value, index, buf, timeout=1000):
    xfer = CtrlTransfer(req_type, request, value, index, len(buf), timeout,
                        ctypes.cast(buf, ctypes.c_void_p))
    return fcntl.ioctl(fd, USBDEVFS_CONTROL, xfer)


def set_rate(fd, ep, rate):
    buf = ctypes.create_string_buffer(rate.to_bytes(3, 'little'), 3)
    # 0x22 = class request, recipient endpoint, host-to-device.
    return control(fd, 0x22, SET_CUR, SAMPLING_FREQ_CONTROL << 8, ep, buf)


def get_rate(fd, ep):
    buf = ctypes.create_string_buffer(3)
    control(fd, 0xa2, GET_CUR, SAMPLING_FREQ_CONTROL << 8, ep, buf)
    return int.from_bytes(buf.raw[:3], 'little')


def main():
    dev = sys.argv[1] if len(sys.argv) > 1 else '/dev/bus/usb/001/003'
    ep = int(sys.argv[2], 0) if len(sys.argv) > 2 else 0x82
    rates = [int(r) for r in sys.argv[3:]] or [48000]

    fd = os.open(dev, os.O_RDWR)
    try:
        try:
            print('current rate on ep 0x%02x: %d' % (ep, get_rate(fd, ep)), flush=True)
        except OSError as e:
            print('GET_CUR failed: %r' % (e,), flush=True)
        for rate in rates:
            t = time.strftime('%H:%M:%S')
            try:
                n = set_rate(fd, ep, rate)
                print('%s SET_CUR ep 0x%02x -> %d Hz: ok (%d bytes)' % (t, ep, rate, n), flush=True)
            except OSError as e:
                print('%s SET_CUR ep 0x%02x -> %d Hz: FAILED %r' % (t, ep, rate, e), flush=True)
            time.sleep(0.2)
            try:
                print('   readback: %d' % get_rate(fd, ep), flush=True)
            except OSError as e:
                print('   readback failed: %r' % (e,), flush=True)
    finally:
        os.close(fd)


if __name__ == '__main__':
    main()
