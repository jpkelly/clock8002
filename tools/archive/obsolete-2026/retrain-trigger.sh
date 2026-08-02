#!/bin/sh
# Identify what triggers PCIe link retraining on the VL805's root port.
#
# A software-commanded retrain must write Link Control (PCIe cap + 0x10) with
# bit 5 set. These kprobes catch every config-space word write with a stack
# trace, so the responsible driver is named. If a retrain is observed with no
# matching write, nothing in software asked for it and the trigger is the
# hardware's own LTSSM dropping into Recovery - i.e. physical layer.
#
# Uses its own ftrace instance. The main buffer carries ~5000 xHCI events/sec,
# which overwrites everything within ~45s - far shorter than the gap between a
# retrain and someone looking at it.
set -u
T=/sys/kernel/debug/tracing
I=$T/instances/retrain
OUT=${OUT:-/var/log/ltc-forensics}
LOG=$OUT/retrain-trigger.log
PROBES="cfgw_cap cfgw_pci cfgw_bus cfgw_user"

# grep -c prints a count and exits non-zero when it is 0, so a naive
# `|| echo 0` fallback emits the number twice.
count() { n=$(grep -c cfgw_ "$1" 2>/dev/null); echo "${n:-0}"; }

case "${1:-start}" in
start)
    mkdir -p $I 2>/dev/null
    [ -d $I ] || { echo "ftrace instances unsupported" >&2; exit 1; }
    echo 8192 > $I/buffer_size_kb 2>/dev/null

    {
        echo 'p:cfgw_cap pcie_capability_write_word pos=%x1 val=%x2'
        echo 'p:cfgw_pci pci_write_config_word pos=%x1 val=%x2'
        echo 'p:cfgw_bus pci_bus_write_config_word devfn=%x1 pos=%x2 val=%x3'
        # Userspace/sysfs writes reach dev->bus->ops->write directly and never
        # pass through the three above; this is the one that catches setpci.
        echo 'p:cfgw_user pci_user_write_config_word pos=%x1 val=%x2'
    } >> $T/kprobe_events 2>/dev/null

    for e in $PROBES; do
        [ -d $I/events/kprobes/$e ] || continue
        echo stacktrace > $I/events/kprobes/$e/trigger 2>/dev/null
        echo 1 > $I/events/kprobes/$e/enable
    done
    echo 1 > $I/tracing_on

    {
        echo "=== $(date '+%H:%M:%S') retrain-trigger monitor armed ==="
        echo "instance=$I  Link Control is PCIe cap+0x10, retrain is bit 5 (0x20)"
    } >> "$LOG"
    echo "armed on dedicated instance. log: $LOG"
    ;;

stop)
    for e in $PROBES; do
        [ -d $I/events/kprobes/$e ] || continue
        echo 0 > $I/events/kprobes/$e/enable 2>/dev/null
        echo '!stacktrace' > $I/events/kprobes/$e/trigger 2>/dev/null
    done
    echo > $T/kprobe_events 2>/dev/null
    rmdir $I 2>/dev/null
    echo "stopped"
    ;;

selftest)
    # A silent zero is otherwise indistinguishable from broken instrumentation.
    before=$(count $I/trace)
    setpci -s 0001:00:00.0 CAP_EXP+0x10.W=0x0000 >/dev/null 2>&1
    sleep 1
    after=$(count $I/trace)
    echo "events before=$before after=$after"
    if [ "$after" -gt "$before" ]; then echo "SELFTEST PASS: kprobes fire"; else echo "SELFTEST FAIL: no events captured"; fi
    ;;

dump)
    echo "total captured: $(count $I/trace)"
    echo "=== config-space word writes (with call stacks) ==="
    grep -A14 'cfgw_' $I/trace 2>/dev/null | tail -80
    ;;

*)
    echo "usage: $0 {start|stop|selftest|dump}" >&2
    exit 2
    ;;
esac
