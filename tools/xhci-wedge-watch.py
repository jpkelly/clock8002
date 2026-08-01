#!/usr/bin/env python3
"""High-rate xHCI state sampler that dumps a ring buffer when USB audio capture
stalls, so the wedge *transition* is captured rather than only its aftermath.

All prior captures were post-mortem: by the time a human SSHes in, the moment of
failure is gone. This keeps a rolling window of controller registers and ring
pointers and freezes it on detection.

Run as root on the target:
    sudo ./xhci-wedge-watch.py
"""

import argparse
import ctypes
import fcntl
import os
import struct
import subprocess
import sys
import time
from collections import deque

DEBUGFS = "/sys/kernel/debug/usb/xhci"
LOGDIR = "/opt/clock8002/logs"
USBDEVFS_CONTROL = 0xC0185500
MFINDEX_MASK = 0x3FFF


def read_text(path):
    try:
        with open(path, "r") as fh:
            return fh.read()
    except OSError:
        return ""


def reg_fields(text):
    out = {}
    for line in text.splitlines():
        if "=" in line:
            k, _, v = line.partition("=")
            out[k.strip()] = v.strip()
    return out


def find_card_index():
    """Resolve by name -- assuming card0 has produced false 'wedged' readings."""
    for line in read_text("/proc/asound/cards").splitlines():
        if "USB-Audio" in line:
            return line.split()[0]
    return None


def find_controller():
    try:
        for name in os.listdir(DEBUGFS):
            if ":" in name and os.path.isdir(os.path.join(DEBUGFS, name)):
                return os.path.join(DEBUGFS, name)
    except OSError:
        pass
    return None


def find_slot(ctrl, usb_path):
    devdir = os.path.join(ctrl, "devices")
    try:
        for slot in sorted(os.listdir(devdir)):
            if read_text(os.path.join(devdir, slot, "name")).strip() == usb_path:
                return os.path.join(devdir, slot)
    except OSError:
        pass
    return None


def find_isoc_ep(slotdir):
    try:
        for ep in sorted(os.listdir(slotdir)):
            if not ep.startswith("ep") or not os.path.isdir(os.path.join(slotdir, ep)):
                continue
            if "Isoch" in read_text(os.path.join(slotdir, ep, "trbs")):
                return os.path.join(slotdir, ep)
    except OSError:
        pass
    return None


def irq_count(pci):
    for line in read_text("/proc/interrupts").splitlines():
        if pci in line:
            parts = line.split()[1:]
            total = 0
            for p in parts:
                if p.isdigit():
                    total += int(p)
                else:
                    break
            return total
    return -1


def hw_ptr(card):
    for line in read_text(f"/proc/asound/card{card}/pcm0c/sub0/status").splitlines():
        if line.startswith("hw_ptr"):
            return int(line.split(":")[1])
    return -1


def control_probe(dev, timeout_ms=1500):
    buf = ctypes.create_string_buffer(18)
    pkt = struct.pack("BBHHHIQ", 0x80, 6, 0x0100, 0, 18, timeout_ms,
                      ctypes.addressof(buf))
    t0 = time.monotonic()
    try:
        fd = os.open(dev, os.O_RDWR)
    except OSError as exc:
        return f"OPEN FAIL errno={exc.errno}"
    try:
        n = fcntl.ioctl(fd, USBDEVFS_CONTROL, pkt)
        return f"OK {n} bytes in {(time.monotonic()-t0)*1000:.0f}ms"
    except OSError as exc:
        return (f"FAIL errno={exc.errno} ({os.strerror(exc.errno)}) "
                f"after {(time.monotonic()-t0)*1000:.0f}ms")
    finally:
        os.close(fd)


def sample(ctrl, eppath, card, pci, urbnum_path, want_irq):
    # Each read source is timed separately: a stall in the debugfs reads (MMIO to
    # the controller) implicates the controller/PCIe link, whereas a stall in
    # /proc/asound implicates the ALSA driver holding a lock.
    t_a = time.perf_counter()
    op = reg_fields(read_text(os.path.join(ctrl, "reg-op")))
    rt = reg_fields(read_text(os.path.join(ctrl, "reg-runtime")))
    enq = read_text(os.path.join(eppath, "enqueue")).strip()[-8:]
    deq = read_text(os.path.join(eppath, "dequeue")).strip()[-8:]
    t_dbg = time.perf_counter()
    hwp = hw_ptr(card)
    t_snd = time.perf_counter()
    urb = read_text(urbnum_path).strip()
    t_sys = time.perf_counter()
    return {
        "t": time.monotonic(),
        "mfindex": rt.get("MFINDEX", "?"),
        "usbsts": op.get("USBSTS", "?"),
        "usbcmd": op.get("USBCMD", "?"),
        "crcr": op.get("CRCR", "?"),
        "iman": rt.get("IR0_IMAN", "?"),
        "erdp": rt.get("IR0_ERDP_LOW", "?"),
        "enq": enq,
        "deq": deq,
        "hw_ptr": hwp,
        "urbnum": urb,
        "irq": irq_count(pci) if want_irq else None,
        "us_dbg": int((t_dbg - t_a) * 1e6),
        "us_snd": int((t_snd - t_dbg) * 1e6),
        "us_sys": int((t_sys - t_snd) * 1e6),
    }


def mf_delta(a, b):
    try:
        return (int(b, 16) - int(a, 16)) & MFINDEX_MASK
    except (ValueError, TypeError):
        return -1


def write_dump(path, ring, ctrl, slotdir, eppath, pci, args, detect_info):
    with open(path, "w") as out:
        w = out.write
        w("=== xhci wedge capture ===\n")
        w(f"wallclock       : {time.strftime('%Y-%m-%d %H:%M:%S')}\n")
        w(f"uptime          : {read_text('/proc/uptime').split()[0]}s\n")
        w(f"kernel          : {os.uname().release}\n")
        w(f"chassis_serial  : {read_text('/proc/device-tree/serial-number').strip(chr(0))}\n")
        w(f"controller      : {ctrl}\n")
        w(f"isoc endpoint   : {eppath}\n")
        w(f"detect          : {detect_info}\n")
        w(f"sample interval : {args.interval*1000:.0f}ms   window {len(ring)} samples\n")
        w("=" * 78 + "\n\n")

        w("--- rolling sample ring (oldest first) ---\n")
        w(f"{'rel_s':>8} {'MFINDEX':>9} {'dMF':>5} {'USBSTS':>10} {'IMAN':>6} "
          f"{'ERDP':>10} {'enq':>9} {'deq':>9} {'hw_ptr':>11} {'dHW':>7} "
          f"{'urbnum':>8} {'IRQ':>9} {'usDBG':>7} {'usSND':>7} {'usSYS':>7}\n")
        t_end = ring[-1]["t"] if ring else 0
        prev = None
        for s in ring:
            dmf = mf_delta(prev["mfindex"], s["mfindex"]) if prev else 0
            dhw = (s["hw_ptr"] - prev["hw_ptr"]) if prev and s["hw_ptr"] >= 0 and prev["hw_ptr"] >= 0 else 0
            w(f"{s['t']-t_end:8.3f} {s['mfindex']:>9} {dmf:5d} {s['usbsts']:>10} "
              f"{s['iman']:>6} {s['erdp']:>10} {s['enq']:>9} {s['deq']:>9} "
              f"{s['hw_ptr']:>11} {dhw:7d} {s['urbnum']:>8} "
              f"{s['irq'] if s['irq'] is not None else '-':>9} "
              f"{s.get('us_dbg', -1):>7} {s.get('us_snd', -1):>7} "
              f"{s.get('us_sys', -1):>7}\n")
            prev = s

        w("\n--- MFINDEX still advancing after wedge? ---\n")
        for i in range(6):
            rt = reg_fields(read_text(os.path.join(ctrl, "reg-runtime")))
            w(f"  {i}: MFINDEX={rt.get('MFINDEX')}  "
              f"USBSTS={reg_fields(read_text(os.path.join(ctrl,'reg-op'))).get('USBSTS')}\n")
            time.sleep(0.12)

        w("\n--- pcm status / hw_params / stream0 ---\n")
        for p in (f"/proc/asound/card{args._card}/pcm0c/sub0/status",
                  f"/proc/asound/card{args._card}/pcm0c/sub0/hw_params",
                  f"/proc/asound/card{args._card}/stream0"):
            w(f"[{p}]\n{read_text(p) or '  (empty -- device closed)'}\n")

        w("\n--- reg-cap / reg-op / reg-runtime ---\n")
        for f in ("reg-cap", "reg-op", "reg-runtime"):
            w(f"[{f}]\n{read_text(os.path.join(ctrl, f))}\n")

        w("--- ports ---\n")
        pdir = os.path.join(ctrl, "ports")
        for p in sorted(os.listdir(pdir)) if os.path.isdir(pdir) else []:
            w(f"  {p}: {read_text(os.path.join(pdir, p, 'portsc')).strip()}\n")

        w("\n--- command / event ring ---\n")
        for r in ("command-ring", "event-ring"):
            for f in ("enqueue", "dequeue", "cycle"):
                w(f"  {r}/{f}: {read_text(os.path.join(ctrl, r, f)).strip()}\n")

        w("\n--- slot + endpoint contexts ---\n")
        w(read_text(os.path.join(slotdir, "slot-context")))
        for line in read_text(os.path.join(slotdir, "ep-context")).splitlines():
            if "INVALID" not in line:
                w(line + "\n")

        w("\n--- isoc endpoint TRBs ---\n")
        w(read_text(os.path.join(eppath, "trbs")))

        w("\n--- control transfer probe (hub first, then dongle) ---\n")
        for dev in args.probe:
            w(f"  {dev}: {control_probe(dev)}\n")

        w("\n--- /proc/interrupts (controller lines) ---\n")
        for line in read_text("/proc/interrupts").splitlines():
            if pci in line or "xhci" in line:
                w("  " + line.strip() + "\n")

        w("\n--- dmesg tail ---\n")
        try:
            w(subprocess.run(["dmesg"], capture_output=True, text=True,
                             timeout=10).stdout[-4000:])
        except Exception as exc:
            w(f"  dmesg failed: {exc}\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--interval", type=float, default=0.025)
    ap.add_argument("--window", type=int, default=600)
    ap.add_argument("--stall-ms", type=int, default=300)
    ap.add_argument("--usb-path", default="1-1.1")
    ap.add_argument("--pci", default="0001:01:00.0")
    ap.add_argument("--probe", nargs="*",
                    default=["/dev/bus/usb/001/002", "/dev/bus/usb/001/003"])
    ap.add_argument("--logdir", default=LOGDIR)
    ap.add_argument("--once", action="store_true",
                    help="exit after the first wedge is captured")
    args = ap.parse_args()

    if os.geteuid() != 0:
        sys.exit("must run as root (debugfs + usbfs)")

    ctrl = find_controller()
    card = find_card_index()
    if not ctrl or card is None:
        sys.exit(f"controller={ctrl} card={card} -- cannot start")
    slotdir = find_slot(ctrl, args.usb_path)
    if not slotdir:
        sys.exit(f"no slot for {args.usb_path}")
    eppath = find_isoc_ep(slotdir)
    if not eppath:
        sys.exit(f"no isoc endpoint under {slotdir}")

    urbnum_path = f"/sys/bus/usb/devices/{args.usb_path}/urbnum"
    args._card = card
    os.makedirs(args.logdir, exist_ok=True)
    print(f"controller {ctrl}\ncard {card}  slot {slotdir}\nisoc ep {eppath}")
    print(f"sampling every {args.interval*1000:.0f}ms, "
          f"stall threshold {args.stall_ms}ms", flush=True)

    ring = deque(maxlen=args.window)
    last_change = time.monotonic()
    last_hw = -1
    dumped_hw = None
    n = 0
    while True:
        s = sample(ctrl, eppath, card, args.pci, urbnum_path, n % 4 == 0)
        ring.append(s)
        n += 1

        if s["hw_ptr"] < 0:
            # PCM not open (status file has no hw_ptr) -- not a stall
            last_hw, last_change = -1, s["t"]
        elif s["hw_ptr"] != last_hw:
            last_hw, last_change = s["hw_ptr"], s["t"]
            dumped_hw = None
        elif ((s["t"] - last_change) * 1000 >= args.stall_ms
              and dumped_hw != s["hw_ptr"]):
            stamp = time.strftime("%Y%m%d-%H%M%S")
            path = os.path.join(args.logdir, f"xhci-wedge-{stamp}.log")
            info = (f"hw_ptr frozen at {s['hw_ptr']} for "
                    f"{(s['t']-last_change)*1000:.0f}ms")
            print(f"WEDGE: {info}\nwriting {path}", flush=True)
            write_dump(path, list(ring), ctrl, slotdir, eppath, args.pci, args, info)
            print("dump complete", flush=True)
            dumped_hw = s["hw_ptr"]
            if args.once:
                return

        time.sleep(args.interval)


if __name__ == "__main__":
    main()
