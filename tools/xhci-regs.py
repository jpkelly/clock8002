#!/usr/bin/env python3
# ---------------------------------------------------------------------------
# xhci-regs.py — read-only dump of xHCI operational registers for the VL805.
#
# WHY THIS EXISTS
# On 2026-07-28 the live-stall capture proved the VL805 stops raising MSI
# completion events while the device is still enumerated and the PCIe link is
# clean. The obvious follow-up question — "did the controller fault internally,
# or is MSI delivery broken?" — cannot be answered from /proc/interrupts alone.
# It needs USBSTS / IMAN.
#
# A post-mortem read taken after the wedge showed:
#     USBCMD=0x00000000 (R/S=0)   USBSTS=0x00000000 (HCH=0, HCE=0, HSE=0)
#     CRCR CRR=1                  IMAN IP=0 IE=1
#     PORTSC1 CCS=1 PED=1 PLS=U0
# i.e. the controller neither runs nor halts, latches no error, and has no
# pending interrupt. BUT that read happened AFTER the kernel's xhci_halt()
# cleared R/S, so it cannot tell us the state at the moment data stopped.
# This script exists so the watchdog can sample the same registers DURING the
# stall window, before the driver perturbs anything.
#
# STRICTLY READ-ONLY. It mmaps the BAR PROT_READ and never writes. It must
# never be able to break the capture it is part of, so every failure path
# prints a diagnostic and exits 0.
# ---------------------------------------------------------------------------

import mmap
import os
import struct
import sys
import time

DEFAULT_DEV = '0001:01:00.0'


def open_bar(devpath):
    """Map BAR0 read-only. Returns (mmap, size, how) or (None, 0, reason)."""
    try:
        with open(os.path.join(devpath, 'resource')) as f:
            start_s, end_s, _flags = f.readline().split()[:3]
        start = int(start_s, 16)
        end = int(end_s, 16)
        size = end - start + 1
    except Exception as e:
        return None, 0, 'cannot parse BAR resource: %r' % (e,)

    if size <= 0 or size > 0x100000:
        return None, 0, 'implausible BAR size 0x%x' % size

    # Preferred: sysfs resource0. Refused while a driver holds the region on
    # some kernels, hence the /dev/mem fallback below.
    try:
        fd = os.open(os.path.join(devpath, 'resource0'), os.O_RDONLY | os.O_SYNC)
        try:
            mm = mmap.mmap(fd, size, mmap.MAP_SHARED, mmap.PROT_READ)
            return mm, size, 'sysfs resource0 (0x%x bytes)' % size
        finally:
            os.close(fd)
    except Exception as e:
        err1 = repr(e)

    try:
        fd = os.open('/dev/mem', os.O_RDONLY | os.O_SYNC)
        try:
            mm = mmap.mmap(fd, size, mmap.MAP_SHARED, mmap.PROT_READ, offset=start)
            return mm, size, '/dev/mem @ 0x%x (0x%x bytes)' % (start, size)
        finally:
            os.close(fd)
    except Exception as e:
        return None, 0, 'sysfs: %s ; /dev/mem: %r' % (err1, e)


def main():
    dev = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_DEV
    devpath = '/sys/bus/pci/devices/' + dev

    if not os.path.isdir(devpath):
        print('device %s not present on the PCI bus' % dev)
        return

    mm, size, how = open_bar(devpath)
    if mm is None:
        print('MMIO unavailable: %s' % how)
        return

    print('mapped via: %s' % how)

    def r32(off):
        if off + 4 > size:
            return None
        return struct.unpack('<I', mm[off:off + 4])[0]

    def decode(val, table):
        if val is None:
            return '(unreadable)'
        names = [n for b, n in table if val & (1 << b)]
        return ', '.join(names) if names else '(none)'

    caplength = mm[0]
    hciversion = (r32(0x00) or 0) >> 16
    hcsparams1 = r32(0x04) or 0
    rtsoff = (r32(0x18) or 0) & ~0x1F
    maxports = (hcsparams1 >> 24) & 0xFF
    op = caplength

    print('CAPLENGTH=0x%02x HCIVERSION=0x%04x HCSPARAMS1=0x%08x maxports=%d RTSOFF=0x%x'
          % (caplength, hciversion, hcsparams1, maxports, rtsoff))

    if caplength in (0x00, 0xFF) or hciversion in (0x0000, 0xFFFF):
        print('!! capability registers look bogus — BAR is not decoding '
              '(card has fallen off the bus); remaining values are meaningless')
        return

    usbcmd_bits = [(0, 'R/S run'), (1, 'HCRST'), (2, 'INTE'), (3, 'HSEE'),
                   (7, 'LHCRST'), (8, 'CSS'), (9, 'CRS'), (10, 'EWE')]
    usbsts_bits = [(0, 'HCH HCHalted'), (2, 'HSE HostSystemError'),
                   (3, 'EINT EventInt'), (4, 'PCD PortChange'), (8, 'SSS'),
                   (9, 'RSS'), (10, 'SRE SaveRestoreErr'),
                   (11, 'CNR ControllerNotReady'), (12, 'HCE HostControllerError')]

    last = {}
    for sample in (1, 2):
        usbcmd = r32(op + 0x00)
        usbsts = r32(op + 0x04)
        crcr_lo = r32(op + 0x18) or 0
        crcr_hi = r32(op + 0x1C) or 0
        crcr = crcr_lo | (crcr_hi << 32)
        iman = r32(rtsoff + 0x20)
        imod = r32(rtsoff + 0x24)
        erdp = (r32(rtsoff + 0x38) or 0) | ((r32(rtsoff + 0x3C) or 0) << 32)

        print('--- register sample %d ---' % sample)
        print('  USBCMD = 0x%08x  [%s]' % (usbcmd or 0, decode(usbcmd, usbcmd_bits)))
        print('  USBSTS = 0x%08x  [%s]' % (usbsts or 0, decode(usbsts, usbsts_bits)))
        print('  CRCR   = 0x%016x  (CRR=%d)' % (crcr, (crcr >> 3) & 1))
        print('  IMAN   = 0x%08x  [IP=%d IE=%d]  IMOD=0x%08x'
              % (iman or 0, (iman or 0) & 1, ((iman or 0) >> 1) & 1, imod or 0))
        print('  ERDP   = 0x%016x  (EHB=%d)' % (erdp, (erdp >> 3) & 1))

        if sample == 1:
            last = {'erdp': erdp, 'usbsts': usbsts}
            time.sleep(2)
        else:
            if last.get('erdp') == erdp:
                print('  ERDP unchanged over 2 s (software-advanced, so this only'
                      ' means the driver processed no events)')

    # Interpretation, so the log is readable without a spec to hand.
    usbcmd = r32(op + 0x00) or 0
    usbsts = r32(op + 0x04) or 0
    iman = r32(rtsoff + 0x20) or 0
    print('--- interpretation ---')
    running = usbcmd & 1
    halted = usbsts & 1
    if not running and not halted:
        print('  R/S=0 AND HCH=0 => controller neither runs nor halts '
              '(this is the "Host halt failed, -110" state)')
    elif running and not halted:
        print('  R/S=1, HCH=0 => controller believes it is still running')
    elif halted:
        print('  HCHalted set => controller is cleanly halted')
    if usbsts & (1 << 12):
        print('  HCE set => internal Host Controller Error latched')
    if usbsts & (1 << 2):
        print('  HSE set => host system error (PCIe/DMA fault)')
    if not (usbsts & (1 << 12)) and not (usbsts & (1 << 2)):
        print('  no HCE/HSE => no internal error latched')
    if (iman & 1) and (iman & 2):
        print('  IMAN IP=1 with IE=1 => event pending but NOT delivered '
              '=> suspect MSI delivery path')
    elif not (iman & 1):
        print('  IMAN IP=0 => no undelivered event; MSI delivery failure NOT indicated')

    print('--- PORTSC ---')
    for p in range(1, maxports + 1):
        v = r32(op + 0x400 + 0x10 * (p - 1))
        if v is None:
            print('  port %2d: (beyond mapped BAR)' % p)
            continue
        if v == 0xFFFFFFFF:
            print('  port %2d: 0xffffffff (not decoding)' % p)
            continue
        print('  port %2d: 0x%08x CCS=%d PED=%d OCA=%d PR=%d PLS=%d PP=%d speed=%d'
              % (p, v, v & 1, (v >> 1) & 1, (v >> 3) & 1, (v >> 4) & 1,
                 (v >> 5) & 0xF, (v >> 9) & 1, (v >> 10) & 0xF))


if __name__ == '__main__':
    try:
        main()
    except Exception as e:  # never let a forensics helper break the capture
        print('xhci-regs.py failed: %r' % (e,))
