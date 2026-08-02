#!/usr/bin/env python3
# ---------------------------------------------------------------------------
# ltc-forensics.py — continuous high-resolution recorder for the alsa-ltc /
# USB-audio / VL805 xHCI path on the Trixie Pi 5 target.
#
# WHY THIS EXISTS
# The existing tools each see part of the picture:
#   - alsa-ltc-usb-wedge-watchdog.sh  runs on a timer, so it only ever observes
#     the aftermath; by then xhci_halt() has already rewritten the registers.
#   - ltcmon.sh                       polls at 1 s and classifies dmesg, but it
#     tails /tmp/alsa-ltc.log, which does not exist on Trixie (alsa-ltc logs to
#     journald), so its gap classifier never fires here.
#   - xhci-regs.py                    reads the operational registers via mmap,
#     but only as a one-shot.
# None of them record the xHCI *transfer ring* state, which is the only place
# the difference between "the device stopped sending" and "the controller
# stopped collecting" is visible.
#
# The 2026-08-01 wedge made that gap concrete. It reproduced stages 1-4 of the
# chain documented in alsa-ltc-usb-wedge-watchdog.sh (clean stream -> decode gap
# with peak_during_gap=32767 -> -EIO every ~3.33 s with pcm_state=RUNNING ->
# teardown at the 10-error limit) but then did NOT proceed to the documented
# stage 5. There was no "still N active urbs", no "HC died", no xhci line in
# dmesg at all, the card stayed in /proc/asound/cards and devnum never changed.
# Only 103x usb_set_interface failed (-110) and 51x cannot set freq, both of
# which are restart-loop consequences rather than the fault. So this target has
# at least two distinct failure modes and the coarse tooling cannot tell them
# apart.
#
# WHAT THIS RECORDS
# A fast loop (default 10 Hz, measured at 0.098 ms/sample on this unit) keeps a
# rolling in-memory ring buffer. Nothing is written while things are healthy
# beyond a decimated 1 Hz trend CSV. When a trigger fires, the entire
# pre-trigger ring buffer is flushed to disk alongside a full forensic snapshot,
# and fast sampling continues into a post-trigger file. That gives the seconds
# BEFORE the failure at full resolution, which is the part every previous
# capture has been missing.
#
# The discriminating signals, and what a freeze in each one means:
#   MFINDEX          controller microframe timer. Frozen => the HC itself has
#                    stopped, not the device.
#   ep deq / enq     isoc IN transfer ring pointers. deq frozen while enq keeps
#                    advancing => the driver is still queueing but the HC has
#                    stopped consuming: URBs are stranding. This is the exact
#                    condition that later produces "still N active urbs".
#   ep-context state running / halted / error for the isoc IN endpoint.
#   USBSTS           HCE (bit 12) / HSE (bit 2) / HCH (bit 0).
#   IR0_ERDP         event ring dequeue pointer. Static while transfers are
#                    outstanding => events are not being processed.
#   xhci MSI count   from /proc/interrupts; separates "no events generated"
#                    from "events generated but not delivered".
#
# STRICTLY READ-ONLY. It opens debugfs, sysfs and procfs for reading only and
# never writes to hardware, never resets anything, and never touches alsa-ltc.
# It must not be able to perturb the fault it exists to observe.
#
# Usage:
#   sudo ./ltc-forensics.py                      # run in foreground
#   sudo ./ltc-forensics.py --outdir /var/log/ltc-forensics --hz 10
#   sudo ./ltc-forensics.py --selftest            # verify probes then exit
# Normally deployed as ltc-forensics.service.
# ---------------------------------------------------------------------------

import argparse
import collections
import glob
import gzip
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import threading
import time

DEBUG_XHCI_ROOT = '/sys/kernel/debug/usb/xhci'
DEFAULT_OUTDIR = '/var/log/ltc-forensics'


def read_text(path, limit=None):
    try:
        with open(path, 'rb') as f:
            data = f.read() if limit is None else f.read(limit)
        return data.decode('utf-8', 'replace')
    except Exception:
        return ''


def read_int(path, base=10):
    raw = read_text(path).strip()
    if not raw:
        return None
    try:
        return int(raw, base)
    except ValueError:
        try:
            return int(raw, 0)
        except ValueError:
            return None


def parse_kv_regs(text):
    """reg-op / reg-runtime are 'NAME = 0xVALUE' lines."""
    out = {}
    for line in text.splitlines():
        if '=' not in line:
            continue
        k, _, v = line.partition('=')
        v = v.strip()
        try:
            out[k.strip()] = int(v, 0)
        except ValueError:
            pass
    return out


# ---------------------------------------------------------------------------
# Topology discovery
# ---------------------------------------------------------------------------

class Topology:
    """Locates the USB audio card, its USB device, and its xHCI debugfs slot."""

    def __init__(self):
        self.card = None            # ALSA card number
        self.pcm_status = None      # /proc/asound/cardN/pcm0c/sub0/status
        self.usb_dev = None         # e.g. 1-1.1
        self.usb_path = None        # /sys/bus/usb/devices/1-1.1
        self.xhci = None            # /sys/kernel/debug/usb/xhci/0001:01:00.0
        self.slot = None            # e.g. 02
        self.slot_dir = None
        self.iso_in_dir = None      # .../devices/02/ep04
        self.iso_in_index = None
        self.pci_bdf = None
        self.pci_bridge = None
        self.errors = []

    def discover(self):
        self._find_card()
        self._find_usb_device()
        self._find_xhci()
        self._find_slot()
        self._find_iso_endpoint()
        self._find_pci()
        return self

    def _find_card(self):
        cards = read_text('/proc/asound/cards')
        for line in cards.splitlines():
            m = re.match(r'\s*(\d+)\s+\[', line)
            if m and 'USB' in line.upper():
                self.card = int(m.group(1))
                break
        if self.card is None:
            self.errors.append('no USB audio card in /proc/asound/cards')
            return
        cand = '/proc/asound/card%d/pcm0c/sub0/status' % self.card
        if os.path.exists(cand):
            self.pcm_status = cand
        else:
            self.errors.append('no capture pcm status at %s' % cand)

    def _find_usb_device(self):
        if self.card is None:
            return
        # /sys/class/sound/cardN/device is the USB *interface*; its parent is
        # the USB device itself.
        iface = os.path.realpath('/sys/class/sound/card%d/device' % self.card)
        parent = os.path.dirname(iface)
        if os.path.exists(os.path.join(parent, 'idVendor')):
            self.usb_path = parent
            self.usb_dev = os.path.basename(parent)
        else:
            self.errors.append('could not resolve USB device from card %s' % self.card)

    def _find_xhci(self):
        if not os.path.isdir(DEBUG_XHCI_ROOT):
            self.errors.append('xhci debugfs missing (need debugfs mounted + root)')
            return
        # Prefer the controller whose PCI BDF appears in our device's sysfs path.
        try:
            entries = sorted(os.listdir(DEBUG_XHCI_ROOT))
        except PermissionError:
            self.errors.append('cannot read %s (need root)' % DEBUG_XHCI_ROOT)
            return
        real = os.path.realpath(self.usb_path) if self.usb_path else ''
        for name in entries:
            if ':' in name and name in real:
                self.xhci = os.path.join(DEBUG_XHCI_ROOT, name)
                self.pci_bdf = name
                return
        for name in entries:
            if ':' in name:
                self.xhci = os.path.join(DEBUG_XHCI_ROOT, name)
                self.pci_bdf = name
                return
        self.errors.append('no PCI xHCI controller under %s' % DEBUG_XHCI_ROOT)

    def _find_slot(self):
        if not self.xhci or not self.usb_dev:
            return
        devroot = os.path.join(self.xhci, 'devices')
        try:
            slots = sorted(os.listdir(devroot))
        except Exception:
            self.errors.append('cannot list %s' % devroot)
            return
        for s in slots:
            name = read_text(os.path.join(devroot, s, 'name')).strip()
            if name == self.usb_dev:
                self.slot = s
                self.slot_dir = os.path.join(devroot, s)
                return
        self.errors.append('no xHCI slot named %s' % self.usb_dev)

    def _find_iso_endpoint(self):
        if not self.slot_dir:
            return
        ctx = read_text(os.path.join(self.slot_dir, 'ep-context'))
        # One line per endpoint context, in ep_index order starting at 0.
        for idx, line in enumerate(ctx.splitlines()):
            if 'Isoc IN' in line:
                cand = os.path.join(self.slot_dir, 'ep%02d' % idx)
                if os.path.isdir(cand):
                    self.iso_in_dir = cand
                    self.iso_in_index = idx
                    return
        self.errors.append('no Isoc IN endpoint found in ep-context')

    def _find_pci(self):
        if not self.pci_bdf:
            return
        dev = '/sys/bus/pci/devices/%s' % self.pci_bdf
        if os.path.isdir(dev):
            parent = os.path.dirname(os.path.realpath(dev))
            if os.path.exists(os.path.join(parent, 'aer_dev_correctable')):
                self.pci_bridge = parent

    def summary(self):
        return {
            'card': self.card,
            'pcm_status': self.pcm_status,
            'usb_dev': self.usb_dev,
            'usb_path': self.usb_path,
            'xhci': self.xhci,
            'pci_bdf': self.pci_bdf,
            'pci_bridge': self.pci_bridge,
            'slot': self.slot,
            'iso_in_dir': self.iso_in_dir,
            'iso_in_index': self.iso_in_index,
            'errors': self.errors,
        }


# ---------------------------------------------------------------------------
# Fast sampler
# ---------------------------------------------------------------------------

FAST_FIELDS = [
    'mono', 'wall', 'mfindex', 'usbcmd', 'usbsts', 'crcr', 'iman', 'erdp',
    'ep_deq', 'ep_enq', 'ep_cycle', 'ep_inflight', 'urbnum', 'xhci_irq',
    'pcm_state', 'hw_ptr', 'appl_ptr', 'avail', 'ce_brg', 'ce_dev',
    'ue_brg', 'lnk_brg', 'dev_brg', 'pcm_p', 'alts',
]

# Offsets within the AER extended capability.
AER_UE_STATUS = 0x04
AER_CE_STATUS = 0x10
# Offsets within the PCI Express capability.
EXP_DEVSTA = 0x0a
EXP_LNKSTA = 0x12


def _aer_cap_offset(bdf):
    """Byte offset of the AER extended capability in config space, or None."""
    try:
        with open('/sys/bus/pci/devices/%s/config' % bdf, 'rb') as f:
            cfg = f.read(0x1000)
    except OSError:
        return None
    if len(cfg) < 0x1000:
        return None
    off = 0x100
    while 0x100 <= off < 0x1000:
        hdr = int.from_bytes(cfg[off:off + 4], 'little')
        if hdr == 0 or hdr == 0xffffffff:
            return None
        if (hdr & 0xffff) == 0x0001:      # Advanced Error Reporting
            return off
        off = (hdr >> 20) & 0xffc
    return None


def _pcie_cap_offset(bdf):
    """Byte offset of the PCI Express capability (ID 0x10), or None."""
    try:
        with open('/sys/bus/pci/devices/%s/config' % bdf, 'rb') as f:
            cfg = f.read(0x100)
    except OSError:
        return None
    if len(cfg) < 0x100:
        return None
    ptr = cfg[0x34]
    seen = set()
    while ptr and ptr != 0xff and ptr not in seen and ptr + 1 < 0x100:
        seen.add(ptr)
        if cfg[ptr] == 0x10:
            return ptr
        ptr = cfg[ptr + 1]
    return None


class PciErrMon:
    """Samples error/link state from one PCI device's config space.

    Status registers are read-and-cleared so each sample reports what happened
    since the last one; a sticky bit latches on first occurrence and then tells
    you nothing about rate. LnkSta is read-only and is the key one: when the
    VL805 stops answering, the bridge still does, so its link status says
    whether the link itself dropped or the device hung with the link up.
    """

    def __init__(self, bdf):
        self.path = '/sys/bus/pci/devices/%s/config' % bdf
        aer = _aer_cap_offset(bdf)
        exp = _pcie_cap_offset(bdf)
        self.ce_off = (aer + AER_CE_STATUS) if aer else None
        self.ue_off = (aer + AER_UE_STATUS) if aer else None
        self.lnk_off = (exp + EXP_LNKSTA) if exp else None
        self.dev_off = (exp + EXP_DEVSTA) if exp else None

    def _rw(self, f, off, width, clear, mask=None):
        f.seek(off)
        val = int.from_bytes(f.read(width), 'little')
        # Only write back bits that are actually write-1-to-clear. Writing read
        # only bits back (DevSta AuxPwr, for one) is a no-op that still puts
        # config traffic on the link to the device being investigated.
        wr = val & mask if mask is not None else val
        if clear and wr:
            f.seek(off)
            f.write(wr.to_bytes(width, 'little'))
        return val

    def sample(self):
        """Returns (ce, ue, lnksta, devsta); each element None if unavailable."""
        ce = ue = lnk = dev = None
        try:
            with open(self.path, 'r+b', buffering=0) as f:
                if self.ce_off:
                    ce = self._rw(f, self.ce_off, 4, True)
                if self.ue_off:
                    ue = self._rw(f, self.ue_off, 4, True)
                if self.lnk_off:
                    lnk = self._rw(f, self.lnk_off, 2, False)
                if self.dev_off:
                    dev = self._rw(f, self.dev_off, 2, True, mask=0x000f)
        except OSError:
            pass
        return ce, ue, lnk, dev


class Sampler:
    def __init__(self, topo):
        self.t = topo
        self._irq_re = re.compile(r'^\s*\d+:\s+(.*?)\s+\S+\s+\S+\s+xhci_hcd\s*$')
        self.aer_brg = PciErrMon(topo.pci_bridge.rsplit('/', 1)[-1]) if topo.pci_bridge else None
        self.aer_dev = PciErrMon(topo.pci_bdf) if topo.pci_bdf else None
        self.pcm_p = ('/proc/asound/card%d/pcm0p/sub0/status' % topo.card
                      if topo.card is not None else None)
        # Alt setting flips to 1 only while an interface's isoc endpoint is
        # active, so this catches a playback stream opening mid-capture even if
        # it closes again between samples.
        self.alt_paths = sorted(glob.glob(
            '/sys/bus/usb/devices/%s:*/bAlternateSetting' % topo.usb_dev
        )) if topo.usb_dev else []

    def xhci_irq_total(self):
        total = 0
        for line in read_text('/proc/interrupts').splitlines():
            if 'xhci_hcd' not in line:
                continue
            parts = line.split()
            for p in parts[1:]:
                if p.isdigit():
                    total += int(p)
                else:
                    break
        return total

    def sample(self):
        t = self.t
        s = {k: None for k in FAST_FIELDS}
        s['mono'] = time.monotonic()
        s['wall'] = time.time()

        if t.xhci:
            rt = parse_kv_regs(read_text(os.path.join(t.xhci, 'reg-runtime')))
            s['mfindex'] = rt.get('MFINDEX')
            s['iman'] = rt.get('IR0_IMAN')
            s['erdp'] = rt.get('IR0_ERDP_LOW')
            op = parse_kv_regs(read_text(os.path.join(t.xhci, 'reg-op')))
            s['usbcmd'] = op.get('USBCMD')
            s['usbsts'] = op.get('USBSTS')
            s['crcr'] = op.get('CRCR')

        if t.iso_in_dir:
            deq = read_int(os.path.join(t.iso_in_dir, 'dequeue'), 16)
            enq = read_int(os.path.join(t.iso_in_dir, 'enqueue'), 16)
            s['ep_deq'] = deq
            s['ep_enq'] = enq
            s['ep_cycle'] = read_int(os.path.join(t.iso_in_dir, 'cycle'))
            if deq is not None and enq is not None:
                # Ring segments are 4 KiB of 16-byte TRBs; a plain masked
                # difference is enough to show the backlog growing.
                s['ep_inflight'] = ((enq - deq) & 0xFFF) // 16

        if t.usb_path:
            s['urbnum'] = read_int(os.path.join(t.usb_path, 'urbnum'))

        s['xhci_irq'] = self.xhci_irq_total()

        if t.pcm_status:
            txt = read_text(t.pcm_status)
            if txt:
                for line in txt.splitlines():
                    f = line.split()
                    if len(f) >= 2:
                        if f[0] == 'state:':
                            s['pcm_state'] = f[1]
                        elif f[0] == 'hw_ptr':
                            s['hw_ptr'] = int(f[-1]) if f[-1].isdigit() else None
                        elif f[0] == 'appl_ptr':
                            s['appl_ptr'] = int(f[-1]) if f[-1].isdigit() else None
                        elif f[0] == 'avail':
                            s['avail'] = int(f[-1]) if f[-1].isdigit() else None
            else:
                s['pcm_state'] = 'CLOSED'

        if self.aer_brg:
            s['ce_brg'], s['ue_brg'], s['lnk_brg'], s['dev_brg'] = self.aer_brg.sample()
        if self.aer_dev:
            s['ce_dev'] = self.aer_dev.sample()[0]

        if self.pcm_p:
            txt = read_text(self.pcm_p).strip()
            s['pcm_p'] = 'closed' if not txt or txt == 'closed' else txt.split()[-1]
        if self.alt_paths:
            s['alts'] = ''.join(read_text(p).strip() or '?' for p in self.alt_paths)
        return s


# ---------------------------------------------------------------------------
# Log followers (journald + kernel ring buffer)
# ---------------------------------------------------------------------------

def dmesg_follow_flag():
    """--follow-new skips the existing ring buffer; -w would replay it as if live."""
    try:
        helptext = subprocess.run(['dmesg', '--help'], capture_output=True,
                                  text=True, timeout=5).stdout
        if '--follow-new' in helptext:
            return '--follow-new'
    except Exception:
        pass
    return '-w'


class LineFollower(threading.Thread):
    """Runs a follow-mode command and pushes matching lines onto a queue."""

    def __init__(self, argv, tag, sink):
        super().__init__(daemon=True)
        self.argv = argv
        self.tag = tag
        self.sink = sink
        self.proc = None
        self._stop = threading.Event()

    def run(self):
        try:
            self.proc = subprocess.Popen(
                self.argv, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                text=True, errors='replace', bufsize=1)
        except Exception as e:
            self.sink(self.tag, '[follower failed: %r]' % (e,))
            return
        for line in self.proc.stdout:
            if self._stop.is_set():
                break
            self.sink(self.tag, line.rstrip('\n'))

    def stop(self):
        self._stop.set()
        if self.proc and self.proc.poll() is None:
            try:
                self.proc.terminate()
            except Exception:
                pass


# Patterns that indicate the fault path, in the order the chain produces them.
APP_PATTERNS = [
    ('app_gap', re.compile(r'\[gap\] no LTC decoded')),
    ('app_eio', re.compile(r'read from audio interface failed')),
    ('app_unrecoverable', re.compile(r'audio device unrecoverable')),
    ('app_setparams', re.compile(r'cannot set parameters')),
]

KERN_PATTERNS = [
    ('kern_hc_died', re.compile(r'HC died|not responding, assume dead|Host halt failed')),
    ('kern_active_urbs', re.compile(r'timeout: still \d+ active urbs')),
    ('kern_stop_ep', re.compile(r'not responding to stop endpoint command')),
    ('kern_iface_fail', re.compile(r'usb_set_interface failed')),
    ('kern_setfreq_fail', re.compile(r'cannot set freq')),
    ('kern_disconnect', re.compile(r'USB disconnect')),
    ('kern_reset', re.compile(r'reset (full|high|low)-speed')),
    ('kern_xrun', re.compile(r'xrun|underrun', re.I)),
    # Must not match the benign boot line "AER: enabled with IRQ <n>".
    ('kern_aer', re.compile(r'PCIe Bus Error|AER:.*(error|recovery)', re.I)),
]

# Only these earn the 30-70 MB ftrace+usbmon dump. Everything else still gets the
# 10 Hz pre/post CSV and register snapshot, which is enough to classify it.
WEDGE_TRIGGERS = {
    'mfindex_frozen', 'ep_dequeue_frozen', 'hw_ptr_frozen', 'xhci_irq_frozen',
    'usbsts_HCE', 'usbsts_HSE', 'usbsts_HCHalted', 'mmio_all_ones',
    'pcie_ue', 'pcie_ue_cmplto', 'pcie_link_changed',
    'usb_altsetting_changed', 'playback_opened',
    'kern_hc_died', 'kern_active_urbs', 'kern_stop_ep',
}


# ---------------------------------------------------------------------------
# Forensic snapshot
# ---------------------------------------------------------------------------

def dump_snapshot(topo, path):
    t = topo
    parts = []

    def sec(title, body):
        parts.append('===== %s =====\n%s\n' % (title, body))

    sec('captured', time.strftime('%Y-%m-%d %H:%M:%S %z'))
    sec('uptime', read_text('/proc/uptime').strip())
    sec('loadavg', read_text('/proc/loadavg').strip())

    if t.xhci:
        for f in ('reg-cap', 'reg-op', 'reg-runtime', 'ports', 'command-ring',
                  'event-ring', 'port_bandwidth'):
            p = os.path.join(t.xhci, f)
            if os.path.exists(p):
                sec('xhci/%s' % f, read_text(p, 65536))
        devroot = os.path.join(t.xhci, 'devices')
        if os.path.isdir(devroot):
            for slot in sorted(os.listdir(devroot)):
                sd = os.path.join(devroot, slot)
                sec('xhci/devices/%s/name' % slot, read_text(os.path.join(sd, 'name')))
                for f in ('slot-context', 'ep-context'):
                    sec('xhci/devices/%s/%s' % (slot, f),
                        read_text(os.path.join(sd, f), 65536))
                for ep in sorted(x for x in os.listdir(sd) if x.startswith('ep')):
                    epd = os.path.join(sd, ep)
                    if not os.path.isdir(epd):
                        continue
                    for f in ('cycle', 'dequeue', 'enqueue'):
                        sec('xhci/devices/%s/%s/%s' % (slot, ep, f),
                            read_text(os.path.join(epd, f)).strip())
                    # The TRB ring is the record of what was queued and
                    # whether the controller ever consumed it.
                    sec('xhci/devices/%s/%s/trbs' % (slot, ep),
                        read_text(os.path.join(epd, 'trbs'), 262144))

    if t.usb_path:
        for f in ('urbnum', 'speed', 'bMaxPower', 'authorized', 'devnum',
                  'idVendor', 'idProduct', 'product',
                  'power/control', 'power/runtime_status',
                  'power/runtime_suspended_time', 'power/active_duration'):
            p = os.path.join(t.usb_path, f)
            if os.path.exists(p):
                sec('usb/%s' % f, read_text(p).strip())

    if t.pcm_status:
        sec('pcm status', read_text(t.pcm_status))
    sec('asound/cards', read_text('/proc/asound/cards'))
    sec('interrupts', read_text('/proc/interrupts'))
    sec('meminfo', read_text('/proc/meminfo', 2048))

    if t.pci_bridge:
        for f in ('aer_dev_correctable', 'aer_dev_fatal', 'aer_dev_nonfatal',
                  'current_link_speed', 'current_link_width'):
            p = os.path.join(t.pci_bridge, f)
            if os.path.exists(p):
                sec('pcie/%s' % f, read_text(p).strip())

    for argv, title in (
            (['lsusb'], 'lsusb'),
            (['lsusb', '-t'], 'lsusb -t'),
            (['vcgencmd', 'get_throttled'], 'throttled'),
            (['vcgencmd', 'measure_temp'], 'temp'),
            (['vcgencmd', 'pmic_read_adc'], 'pmic'),
            (['systemctl', 'show', 'alsa-ltc',
              '-p', 'NRestarts', '-p', 'ActiveState', '-p', 'SubState',
              '-p', 'ExecMainStartTimestamp', '-p', 'MainPID'], 'alsa-ltc unit'),
    ):
        if shutil.which(argv[0]):
            try:
                out = subprocess.run(argv, capture_output=True, text=True,
                                     timeout=10).stdout
            except Exception as e:
                out = '(failed: %r)' % (e,)
            sec(title, out.strip())

    with open(path, 'w') as f:
        f.write('\n'.join(parts))


def dump_logs(outdir):
    for argv, name in (
            (['dmesg', '-T'], 'dmesg.txt'),
            (['journalctl', '-u', 'alsa-ltc', '-n', '3000',
              '--no-pager', '-o', 'short-precise'], 'journal-alsa-ltc.txt'),
            (['journalctl', '-b', '-n', '3000',
              '--no-pager', '-o', 'short-precise'], 'journal-boot.txt'),
    ):
        if not shutil.which(argv[0]):
            continue
        try:
            out = subprocess.run(argv, capture_output=True, text=True,
                                 timeout=60).stdout
        except Exception as e:
            out = '(failed: %r)' % (e,)
        with open(os.path.join(outdir, name), 'w') as f:
            f.write(out)


def write_csv(path, rows, extra_cols=None):
    cols = list(FAST_FIELDS) + list(extra_cols or [])
    with open(path, 'w') as f:
        f.write(','.join(cols) + '\n')
        for r in rows:
            f.write(','.join(
                '' if r.get(c) is None else str(r.get(c)) for c in cols) + '\n')


# ---------------------------------------------------------------------------
# Kernel-level capture: usbmon URB stream + xHCI ftrace ring
# ---------------------------------------------------------------------------

TRACE_DIR = '/sys/kernel/debug/tracing'

# The five that form one isoc round trip, plus the error/recovery commands that
# only appear when the driver is trying to unstick an endpoint.
XHCI_EVENTS = [
    'xhci_handle_event', 'xhci_handle_transfer', 'xhci_urb_giveback',
    'xhci_urb_enqueue', 'xhci_ring_ep_doorbell', 'xhci_urb_dequeue',
    'xhci_handle_cmd_stop_ep', 'xhci_handle_cmd_reset_ep',
    'xhci_handle_cmd_set_deq_ep', 'xhci_dbg_cancel_urb', 'xhci_ring_expansion',
]


def _wr(path, val):
    with open(path, 'w') as f:
        f.write(str(val))


class UsbmonRing(threading.Thread):
    """Holds the last N usbmon lines for one bus in RAM, flushed on trigger."""

    def __init__(self, bus, maxlines, log):
        super().__init__(daemon=True)
        self.path = '/sys/kernel/debug/usb/usbmon/%su' % bus
        self.ring = collections.deque(maxlen=maxlines)
        self.log = log
        self.proc = None
        self._stop = threading.Event()
        self.lines_seen = 0

    def available(self):
        return os.path.exists(self.path)

    def run(self):
        try:
            # cat runs as our uid; the unit is root so the 0600 node is readable.
            self.proc = subprocess.Popen(
                ['cat', self.path], stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL, text=True, errors='replace', bufsize=1)
        except Exception as e:
            self.log('usbmon reader failed: %r' % (e,))
            return
        for line in self.proc.stdout:
            if self._stop.is_set():
                break
            self.ring.append(line)
            self.lines_seen += 1

    def dump(self, path):
        lines = list(self.ring)
        with gzip.open(path + '.gz', 'wt') as f:
            f.writelines(lines)
        return len(lines)

    def stop(self):
        self._stop.set()
        if self.proc and self.proc.poll() is None:
            try:
                self.proc.terminate()
            except Exception:
                pass


class Ftrace:
    """xHCI tracepoints into ftrace's own ring, frozen via the snapshot buffer.

    Snapshot swaps main and snapshot buffers atomically, so a capture costs one
    write and tracing continues uninterrupted into the emptied buffer.
    """

    def __init__(self, cpu0_kb, other_kb, log):
        self.cpu0_kb = cpu0_kb
        self.other_kb = other_kb
        self.log = log
        self.enabled = []
        self.active = False

    def available(self):
        return os.path.isdir(os.path.join(TRACE_DIR, 'events/xhci-hcd'))

    def setup(self):
        if not self.available():
            self.log('ftrace: no xhci-hcd tracepoints, skipping')
            return False
        try:
            cur = open(os.path.join(TRACE_DIR, 'current_tracer')).read().strip()
            if cur != 'nop':
                self.log('ftrace: current_tracer=%r not nop, refusing to clobber' % cur)
                return False
            _wr(os.path.join(TRACE_DIR, 'tracing_on'), 0)
            _wr(os.path.join(TRACE_DIR, 'trace'), '')
            # xhci IRQs are CPU-pinned, so only that CPU needs a deep buffer.
            percpu = os.path.join(TRACE_DIR, 'per_cpu')
            if os.path.isdir(percpu):
                for d in sorted(os.listdir(percpu)):
                    p = os.path.join(percpu, d, 'buffer_size_kb')
                    if os.path.exists(p):
                        try:
                            _wr(p, self.cpu0_kb if d == 'cpu0' else self.other_kb)
                        except Exception as e:
                            self.log('ftrace: %s size failed: %r' % (d, e))
            for ev in XHCI_EVENTS:
                p = os.path.join(TRACE_DIR, 'events/xhci-hcd', ev, 'enable')
                if os.path.exists(p):
                    _wr(p, 1)
                    self.enabled.append(ev)
            _wr(os.path.join(TRACE_DIR, 'snapshot'), 1)   # allocate snapshot buffer
            _wr(os.path.join(TRACE_DIR, 'trace'), '')
            _wr(os.path.join(TRACE_DIR, 'tracing_on'), 1)
            self.active = True
            self.log('ftrace: %d tracepoints, cpu0=%dKB others=%dKB'
                     % (len(self.enabled), self.cpu0_kb, self.other_kb))
            return True
        except Exception as e:
            self.log('ftrace setup failed: %r' % (e,))
            return False

    def snapshot(self, path):
        if not self.active:
            return 0
        try:
            _wr(os.path.join(TRACE_DIR, 'snapshot'), 1)
            n = 0
            with open(os.path.join(TRACE_DIR, 'snapshot')) as src, \
                    gzip.open(path + '.gz', 'wt') as dst:
                for line in src:
                    dst.write(line)
                    n += 1
            return n
        except Exception as e:
            self.log('ftrace snapshot failed: %r' % (e,))
            return 0

    def teardown(self):
        if not self.active:
            return
        try:
            _wr(os.path.join(TRACE_DIR, 'tracing_on'), 0)
            for ev in self.enabled:
                p = os.path.join(TRACE_DIR, 'events/xhci-hcd', ev, 'enable')
                if os.path.exists(p):
                    _wr(p, 0)
            _wr(os.path.join(TRACE_DIR, 'snapshot'), 0)   # free snapshot buffer
            _wr(os.path.join(TRACE_DIR, 'trace'), '')
        except Exception:
            pass
        self.active = False


# ---------------------------------------------------------------------------
# Recorder
# ---------------------------------------------------------------------------

class Recorder:
    def __init__(self, args):
        self.args = args
        self.topo = Topology().discover()
        self.sampler = Sampler(self.topo)
        self.interval = 1.0 / args.hz
        self.ring = collections.deque(maxlen=int(args.pre_seconds * args.hz))
        self.events = []
        self.event_count = 0
        self.log_lock = threading.Lock()
        self.pending = collections.deque()
        self.running = True
        self.trend_path = os.path.join(args.outdir, 'trend.csv')
        self.log_path = os.path.join(args.outdir, 'ltc-forensics.log')
        self._last_trend = 0.0
        self._trend_init = False
        # freeze detectors
        self._mf_same = 0
        self._deq_same = 0
        self._hw_same = 0
        self._irq_same = 0
        self._last = None
        self._cooldown_until = 0.0
        self.usbmon = None
        self.ftrace = None
        self._last_topo_check = 0.0
        self._topo_lost = False
        self._disk_warned = False

    def log(self, msg):
        line = '%s %s' % (time.strftime('%Y-%m-%d %H:%M:%S'), msg)
        print(line, flush=True)
        try:
            with open(self.log_path, 'a') as f:
                f.write(line + '\n')
        except Exception:
            pass

    def on_log_line(self, tag, line):
        pats = APP_PATTERNS if tag == 'app' else KERN_PATTERNS
        for name, rx in pats:
            if rx.search(line):
                with self.log_lock:
                    self.pending.append((name, line))
                break

    def start_followers(self):
        self.followers = []
        if shutil.which('journalctl'):
            self.followers.append(LineFollower(
                ['journalctl', '-u', 'alsa-ltc', '-f', '-n', '0', '-o', 'cat'],
                'app', self.on_log_line))
        if shutil.which('dmesg'):
            self.followers.append(LineFollower(
                ['dmesg', dmesg_follow_flag(), '-T'], 'kern', self.on_log_line))
        for f in self.followers:
            f.start()

    def refresh_topology(self):
        """The endpoint dir disappears while the device is wedged and does not
        exist yet if we start before enumeration, so recover it rather than
        running blind for the rest of the process lifetime."""
        t = self.topo
        if t.iso_in_dir and os.path.isdir(t.iso_in_dir):
            self._topo_lost = False
            return
        if not self._topo_lost:
            self._topo_lost = True
            self.log('topology: isoc endpoint gone, retrying discovery')
        new = Topology().discover()
        if new.iso_in_dir and os.path.isdir(new.iso_in_dir):
            self.topo = new
            self.sampler.t = new
            self._topo_lost = False
            self.log('topology: recovered %s slot %s ep%s'
                     % (new.usb_dev, new.slot, new.iso_in_index))

    def start_kernel_capture(self):
        if self.args.usbmon:
            bus = self.topo.usb_dev.split('-')[0] if self.topo.usb_dev else '1'
            r = UsbmonRing(bus, int(self.args.usbmon_seconds * 2200), self.log)
            if r.available():
                r.start()
                self.usbmon = r
                self.log('usbmon: bus %s, ring %d lines (~%ds)'
                         % (bus, r.ring.maxlen, self.args.usbmon_seconds))
            else:
                self.log('usbmon: %s missing (modprobe usbmon?)' % r.path)
        if self.args.ftrace:
            f = Ftrace(self.args.ftrace_cpu0_kb, self.args.ftrace_other_kb, self.log)
            if f.setup():
                self.ftrace = f

    def check_freezes(self, s):
        """Returns a list of trigger names based on register-level stalls."""
        trigs = []
        prev = self._last
        self._last = s
        if prev is None:
            return trigs

        streaming = s.get('pcm_state') == 'RUNNING'

        if s['mfindex'] is not None and s['mfindex'] == prev['mfindex']:
            self._mf_same += 1
        else:
            self._mf_same = 0
        if self._mf_same >= self.args.mfindex_frozen_samples:
            trigs.append('mfindex_frozen')

        if streaming and s['ep_deq'] is not None and s['ep_deq'] == prev['ep_deq']:
            self._deq_same += 1
        else:
            self._deq_same = 0
        if self._deq_same >= self.args.ep_frozen_samples:
            trigs.append('ep_dequeue_frozen')

        if streaming and s['hw_ptr'] is not None and s['hw_ptr'] == prev['hw_ptr']:
            self._hw_same += 1
        else:
            self._hw_same = 0
        if self._hw_same >= self.args.hw_frozen_samples:
            trigs.append('hw_ptr_frozen')

        if streaming and s['xhci_irq'] is not None and s['xhci_irq'] == prev['xhci_irq']:
            self._irq_same += 1
        else:
            self._irq_same = 0
        if self._irq_same >= self.args.irq_frozen_samples:
            trigs.append('xhci_irq_frozen')

        sts = s.get('usbsts')
        if sts is not None and sts != 0xffffffff:
            if sts & (1 << 12):
                trigs.append('usbsts_HCE')
            if sts & (1 << 2):
                trigs.append('usbsts_HSE')
            if sts & (1 << 0):
                trigs.append('usbsts_HCHalted')
        elif sts == 0xffffffff:
            # All-ones is not a status value, it is the absence of a response.
            trigs.append('mmio_all_ones')

        ue = s.get('ue_brg')
        if ue:
            trigs.append('pcie_ue_cmplto' if ue & (1 << 14) else 'pcie_ue')

        lnk = s.get('lnk_brg')
        if lnk is not None:
            prev_lnk = prev.get('lnk_brg')
            # Bits 0-9 are speed and width; a change there means the link
            # retrained or dropped, which the bridge can still report even
            # when the device below it has stopped answering.
            if prev_lnk is not None and (lnk & 0x3ff) != (prev_lnk & 0x3ff):
                trigs.append('pcie_link_changed')

        if s.get('alts') and prev.get('alts') and s['alts'] != prev['alts']:
            trigs.append('usb_altsetting_changed')
        if s.get('pcm_p') not in (None, 'closed') and prev.get('pcm_p') == 'closed':
            trigs.append('playback_opened')
        return trigs

    def fire(self, triggers, note=''):
        now = time.monotonic()
        if now < self._cooldown_until:
            return
        if self.event_count >= self.args.max_events:
            return
        # An unattended run must never be the reason the clock's rootfs fills.
        try:
            free_mb = shutil.disk_usage(self.args.outdir).free / 1048576
        except Exception:
            free_mb = None
        if free_mb is not None and free_mb < self.args.min_free_mb:
            if not self._disk_warned:
                self._disk_warned = True
                self.log('DISK GUARD: %.0f MB free < %.0f MB, capture suspended'
                         % (free_mb, self.args.min_free_mb))
            return
        self._disk_warned = False
        self._cooldown_until = now + self.args.cooldown
        self.event_count += 1
        name = 'event-%s-%s' % (time.strftime('%Y%m%d-%H%M%S'),
                                '+'.join(sorted(set(triggers)))[:60])
        d = os.path.join(self.args.outdir, name)
        os.makedirs(d, exist_ok=True)
        self.log('TRIGGER %s -> %s' % (','.join(sorted(set(triggers))), d))

        # Only the real wedge signature earns the expensive kernel-level dump
        # (30-70 MB/event). Cheap triggers like app_gap or a single kernel
        # error line still get the 10 Hz snapshot below, just not this part.
        is_wedge = bool(WEDGE_TRIGGERS & set(triggers))
        if is_wedge and self.ftrace:
            n = self.ftrace.snapshot(os.path.join(d, 'ftrace-at-trigger.txt'))
            self.log('  ftrace snapshot: %d lines' % n)
        if is_wedge and self.usbmon:
            n = self.usbmon.dump(os.path.join(d, 'usbmon.txt'))
            self.log('  usbmon: %d lines' % n)

        pre = list(self.ring)
        write_csv(os.path.join(d, 'pre.csv'), pre)
        try:
            dump_snapshot(self.topo, os.path.join(d, 'snapshot.txt'))
        except Exception as e:
            self.log('snapshot failed: %r' % (e,))
        threading.Thread(target=dump_logs, args=(d,), daemon=True).start()

        meta = {
            'triggers': sorted(set(triggers)),
            'note': note,
            'fired_at': time.strftime('%Y-%m-%dT%H:%M:%S%z'),
            'pre_samples': len(pre),
            'hz': self.args.hz,
            'topology': self.topo.summary(),
            'event_index': self.event_count,
        }
        with open(os.path.join(d, 'meta.json'), 'w') as f:
            json.dump(meta, f, indent=2)

        # Keep sampling at full rate into the post-trigger window.
        threading.Thread(target=self._post_capture, args=(d, is_wedge), daemon=True).start()

    def _post_capture(self, d, is_wedge):
        rows = []
        start = time.monotonic()
        end = start + self.args.post_seconds
        # Second snapshot lands after the stall has run its course, so the pair
        # brackets it: one holds the onset, one holds the recovery.
        recovery_at = start + self.args.ftrace_recovery_delay
        recovery_done = False
        while time.monotonic() < end and self.running:
            try:
                rows.append(self.sampler.sample())
            except Exception:
                pass
            if is_wedge and not recovery_done and self.ftrace and time.monotonic() >= recovery_at:
                recovery_done = True
                n = self.ftrace.snapshot(os.path.join(d, 'ftrace-recovery.txt'))
                self.log('  ftrace recovery snapshot: %d lines' % n)
            time.sleep(self.interval)
        write_csv(os.path.join(d, 'post.csv'), rows)
        try:
            dump_snapshot(self.topo, os.path.join(d, 'snapshot-after.txt'))
        except Exception:
            pass
        self.log('post-capture complete: %s (%d samples)' % (d, len(rows)))

    def append_trend(self, s):
        if not self._trend_init:
            new = not os.path.exists(self.trend_path)
            self._trend_init = True
            if new:
                with open(self.trend_path, 'w') as f:
                    f.write(','.join(FAST_FIELDS) + '\n')
        with open(self.trend_path, 'a') as f:
            f.write(','.join(
                '' if s.get(c) is None else str(s.get(c)) for c in FAST_FIELDS) + '\n')

    def run(self):
        os.makedirs(self.args.outdir, exist_ok=True)
        self.log('ltc-forensics starting: %s' % json.dumps(self.topo.summary()))
        if self.topo.errors:
            self.log('WARNING topology gaps: %s' % '; '.join(self.topo.errors))
        if not self.topo.iso_in_dir:
            self.log('WARNING no isoc IN endpoint: ring-level detection disabled')
        self.start_followers()
        self.start_kernel_capture()

        next_t = time.monotonic()
        while self.running:
            try:
                s = self.sampler.sample()
            except Exception as e:
                self.log('sample failed: %r' % (e,))
                time.sleep(self.interval)
                continue
            self.ring.append(s)

            trigs = self.check_freezes(s)

            with self.log_lock:
                while self.pending:
                    name, line = self.pending.popleft()
                    trigs.append(name)
                    self.log('LOG %s | %s' % (name, line[:200]))

            if trigs:
                self.fire(trigs)

            if s['mono'] - self._last_trend >= self.args.trend_interval:
                self._last_trend = s['mono']
                try:
                    self.append_trend(s)
                except Exception:
                    pass

            if s['mono'] - self._last_topo_check >= self.args.topology_recheck:
                self._last_topo_check = s['mono']
                try:
                    self.refresh_topology()
                except Exception as e:
                    self.log('topology refresh failed: %r' % (e,))

            next_t += self.interval
            delay = next_t - time.monotonic()
            if delay > 0:
                time.sleep(delay)
            else:
                next_t = time.monotonic()

    def stop(self, *_):
        self.running = False
        for f in getattr(self, 'followers', []):
            f.stop()
        if self.usbmon:
            self.usbmon.stop()
        if self.ftrace:
            self.ftrace.teardown()
        self.log('ltc-forensics stopping')


def selftest(args):
    topo = Topology().discover()
    print(json.dumps(topo.summary(), indent=2))
    if topo.errors:
        print('\nPROBLEMS:')
        for e in topo.errors:
            print('  - %s' % e)
    sampler = Sampler(topo)
    print('\nthree samples:')
    for _ in range(3):
        s = sampler.sample()
        print('  mfindex=%s usbsts=%s deq=%s enq=%s inflight=%s urbnum=%s '
              'irq=%s pcm=%s hw_ptr=%s' % (
                  s['mfindex'], s['usbsts'],
                  hex(s['ep_deq']) if s['ep_deq'] else None,
                  hex(s['ep_enq']) if s['ep_enq'] else None,
                  s['ep_inflight'], s['urbnum'], s['xhci_irq'],
                  s['pcm_state'], s['hw_ptr']))
        time.sleep(0.5)
    t0 = time.time()
    for _ in range(200):
        sampler.sample()
    print('\ncost: %.3f ms/sample' % ((time.time() - t0) / 200 * 1000))
    return 0 if not topo.errors else 1


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--outdir', default=DEFAULT_OUTDIR)
    p.add_argument('--hz', type=float, default=10.0,
                   help='fast sample rate (default 10)')
    p.add_argument('--pre-seconds', type=float, default=600.0,
                   help='seconds of pre-trigger history to retain (default 600)')
    p.add_argument('--post-seconds', type=float, default=120.0,
                   help='seconds to keep sampling after a trigger (default 120)')
    p.add_argument('--trend-interval', type=float, default=1.0,
                   help='seconds between rolling trend.csv rows (default 1)')
    p.add_argument('--cooldown', type=float, default=0.0,
                   help='minimum seconds between events (0 = capture every trigger)')
    p.add_argument('--max-events', type=int, default=2000)
    p.add_argument('--min-free-mb', type=float, default=3000.0,
                   help='suspend capture below this much free disk')
    p.add_argument('--topology-recheck', type=float, default=5.0,
                   help='seconds between endpoint re-discovery attempts')
    p.add_argument('--usbmon', action='store_true',
                   help='keep a rolling usbmon URB capture, dumped on trigger')
    p.add_argument('--usbmon-seconds', type=float, default=30.0,
                   help='seconds of usbmon history to retain (~200 KB/s)')
    p.add_argument('--ftrace', action='store_true',
                   help='record xHCI tracepoints, snapshotted on trigger')
    p.add_argument('--ftrace-cpu0-kb', type=int, default=12288,
                   help='ftrace ring on the IRQ cpu (measured ~840 KB/s)')
    p.add_argument('--ftrace-other-kb', type=int, default=512)
    p.add_argument('--ftrace-recovery-delay', type=float, default=4.0,
                   help='seconds after trigger for the second snapshot')
    p.add_argument('--mfindex-frozen-samples', type=int, default=3)
    p.add_argument('--ep-frozen-samples', type=int, default=5)
    p.add_argument('--hw-frozen-samples', type=int, default=15)
    p.add_argument('--irq-frozen-samples', type=int, default=20)
    p.add_argument('--selftest', action='store_true')
    args = p.parse_args()

    if args.selftest:
        return selftest(args)

    if os.geteuid() != 0:
        print('must run as root (debugfs)', file=sys.stderr)
        return 1

    rec = Recorder(args)
    signal.signal(signal.SIGTERM, rec.stop)
    signal.signal(signal.SIGINT, rec.stop)
    try:
        rec.run()
    except KeyboardInterrupt:
        rec.stop()
    return 0


if __name__ == '__main__':
    sys.exit(main())
