#!/usr/bin/env python3
"""cm5-hosted troubleshooting harness for piClock LTC/USB wedge testing.

Runs ON cm5, drives a target unit over SSH. Keeps all results on cm5 so they
survive card swaps, reflashes, or the unit hanging. Tracks every change it
makes to the unit in a manifest (written BEFORE each change is applied) so
`revert` can restore the unit to its pre-test state, and `verify` can prove
the revert actually worked rather than trusting exit codes.

Two test modes are implemented: `soak` and `cycle`. `accumulate` and `ab` are
stubbed (see NotImplementedError) - see /memories/session/plan.md for the full
spec and remaining phases (that file was session-scoped and no longer exists;
see tools/cm5-harness/salvage-reference/ for reference implementations).

Usage (run on cm5):
    ./harness.py run soak --host piclock.local --duration 3600 --label "my test"
    ./harness.py status --run <run_id>
    ./harness.py collect --run <run_id>
    ./harness.py revert --run <run_id>
    ./harness.py verify --run <run_id>
    ./harness.py list
"""
import argparse
import datetime
import json
import os
import subprocess
import sys
import time

RESULTS_ROOT = os.path.expanduser("~/piclock-tests")
SSH_OPTS = ["-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
            "-o", "ConnectTimeout=10"]

SOAK_SAMPLE_SH = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                              "salvage-reference", "ltc-soak-sample.sh")

SOAK_SERVICE = """[Unit]
Description=LTC soak sample (harness-managed)
After=alsa-ltc.service clock8002.service
Wants=alsa-ltc.service

[Service]
Type=oneshot
EnvironmentFile=-/etc/default/ltc-soak
ExecStart=/opt/clock8002/ltc-soak-sample.sh
"""

SOAK_TIMER = """[Unit]
Description=LTC soak sample timer (harness-managed)

[Timer]
OnBootSec=60
OnUnitActiveSec=60
Persistent=true

[Install]
WantedBy=timers.target
"""

CYCLE_SAMPLE_SH = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                                "salvage-reference", "ltc-cycle4.sh")

CYCLE_SERVICE = """[Unit]
Description=Multi-boot LTC stability cycle test (production conditions)
# Type=simple, and NOT After=clock8002.service. Both matter:
#   - oneshot would block multi-user.target, and clock8002 is After=multi-user.target,
#     so the display could never start during a cycle.
#   - After=clock8002.service creates an ordering cycle and the job gets deleted.
After=alsa-ltc.service
Wants=alsa-ltc.service

[Service]
Type=simple
ExecStart=/opt/clock8002/ltc-cycle.sh
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
"""


def ssh(host, remote_cmd, timeout=None):
    """Run a shell command on the target unit over SSH. Returns (rc, stdout, stderr)."""
    cmd = ["ssh"] + SSH_OPTS + [f"pi@{host}", remote_cmd]
    p = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    return p.returncode, p.stdout, p.stderr


def ssh_ok(host, remote_cmd, timeout=None):
    rc, out, err = ssh(host, remote_cmd, timeout=timeout)
    if rc != 0:
        raise RuntimeError(f"remote command failed (rc={rc}): {remote_cmd}\nstderr: {err}")
    return out


def scp_content_to(host, remote_path, content):
    """Write `content` to `remote_path` on the unit via ssh (no sftp dependency,
    matches the BusyBox-safe streaming approach used throughout this session)."""
    quoted = remote_path.replace("'", "'\\''")
    cmd = ["ssh"] + SSH_OPTS + [f"pi@{host}", f"sudo tee '{quoted}' >/dev/null"]
    p = subprocess.run(cmd, input=content, capture_output=True, text=True)
    if p.returncode != 0:
        raise RuntimeError(f"failed to write {remote_path}: {p.stderr}")


def read_remote_file(host, remote_path):
    """Return remote file content, or None if it doesn't exist."""
    rc, out, _ = ssh(host, f"sudo cat '{remote_path}' 2>/dev/null; echo RC=$?")
    if "RC=0" not in out.splitlines()[-1] if out else True:
        pass
    rc2, out2, _ = ssh(host, f"[ -e '{remote_path}' ] && echo EXISTS || echo MISSING")
    if "MISSING" in out2:
        return None
    _, content, _ = ssh(host, f"sudo cat '{remote_path}'")
    return content


def harness_ram():
    """Return (rss_kb, swap_kb) for this harness.py control process itself, on
    cm5 - not the target. This is the only place the harness's own footprint
    can be captured, so per-poll samples get tagged with it at pull time (see
    pull_csv/pull_cycle_results) and start/end snapshots go in meta.json."""
    try:
        with open("/proc/self/status") as f:
            status = f.read()
    except OSError:
        return (None, None)
    rss = swap = None
    for line in status.splitlines():
        if line.startswith("VmRSS:"):
            rss = int(line.split()[1])
        elif line.startswith("VmSwap:"):
            swap = int(line.split()[1])
    return (rss, swap)


def check_no_active_run(host, force=False):
    """Refuse to deploy against a host that already has a run marked 'running'
    against it - two concurrent deploys of the same manifest-tracked unit names
    corrupt each other's backup/revert chain (observed 2026-08-01: a non-screen
    launch was left running when the same test was relaunched in screen against
    the same host, and the second run's manifest backed up the first run's
    already-deployed files as if they were the pre-existing originals)."""
    if not os.path.isdir(RESULTS_ROOT):
        return
    for run_id in sorted(os.listdir(RESULTS_ROOT)):
        meta_path = os.path.join(RESULTS_ROOT, run_id, "meta.json")
        if not os.path.exists(meta_path):
            continue
        with open(meta_path) as f:
            meta = json.load(f)
        if meta.get("host") == host and meta.get("status") == "running":
            if force:
                print(f"WARNING: --force override: ignoring active run {run_id} "
                      f"against {host} (status=running). Only do this if you've "
                      f"confirmed that run's process is dead and the target is clean.")
                continue
            sys.exit(
                f"ERROR: run {run_id} is already marked 'running' against {host}. "
                f"Deploying now would corrupt both runs' revert manifests. Either "
                f"wait for it to finish, run 'revert --run {run_id}' first, or pass "
                f"--force if you've confirmed its process is dead and the target "
                f"is already clean.")


class Manifest:
    """Append-only TSV log of every change made to the unit for this run.
    Written BEFORE each change is applied, so a crash mid-deploy still leaves
    a usable (if partial) record for revert."""

    def __init__(self, run_dir):
        self.path = os.path.join(run_dir, "manifest.tsv")
        if not os.path.exists(self.path):
            with open(self.path, "w") as f:
                f.write("timestamp\taction\ttarget\tbackup_file\tdetail\n")

    def _append(self, action, target, backup_file="", detail=""):
        ts = datetime.datetime.now().isoformat()
        with open(self.path, "a") as f:
            f.write(f"{ts}\t{action}\t{target}\t{backup_file}\t{detail}\n")

    def record_write_file(self, host, remote_path, run_dir):
        """Call BEFORE writing remote_path. Backs up existing content (if any)
        into run_dir, then records the action."""
        existing = read_remote_file(host, remote_path)
        backup_file = ""
        if existing is not None:
            backup_name = remote_path.strip("/").replace("/", "_") + ".orig"
            backup_file = os.path.join(run_dir, "backups", backup_name)
            os.makedirs(os.path.dirname(backup_file), exist_ok=True)
            with open(backup_file, "w") as f:
                f.write(existing)
            self._append("write_file_existed", remote_path, backup_file)
        else:
            self._append("write_file_new", remote_path, "")

    def record_enable_service(self, host, unit_name, was_enabled, was_active):
        self._append("enable_service", unit_name, "",
                      f"was_enabled={was_enabled} was_active={was_active}")

    def record_always_delete(self, remote_path):
        """For generated files (e.g. results logs) that should be removed on
        revert regardless of whether they pre-existed - no backup is kept, since
        the point is a clean target rather than restoring old accumulated data.
        The harness's own local copy (pulled during polling) is the archive."""
        self._append("always_delete", remote_path, "")

    def read_entries(self):
        entries = []
        with open(self.path) as f:
            next(f)  # header
            for line in f:
                parts = line.rstrip("\n").split("\t")
                if len(parts) < 5:
                    continue
                entries.append({"timestamp": parts[0], "action": parts[1],
                                 "target": parts[2], "backup_file": parts[3],
                                 "detail": parts[4]})
        return entries


def fingerprint(host):
    """Capture a snapshot of board identity + service state, used to verify
    `revert` actually restored the unit (fingerprint before == fingerprint after)."""
    out = ssh_ok(host, "sh -c '"
        "echo SERIAL=$(awk \"/^Serial/{print \\$3}\" /proc/cpuinfo); "
        "echo HOSTNAME=$(hostname); "
        "echo KERNEL=$(uname -r); "
        "echo ALSA_LTC_ENABLED=$(systemctl is-enabled alsa-ltc 2>&1); "
        "echo ALSA_LTC_ACTIVE=$(systemctl is-active alsa-ltc 2>&1); "
        "echo CLOCK8002_ACTIVE=$(systemctl is-active clock8002 2>&1); "
        "echo SOAK_TIMER=$(systemctl is-active ltc-soak.timer 2>&1); "
        "echo SOAK_TIMER_ENABLED=$(systemctl is-enabled ltc-soak.timer 2>&1); "
        "'")
    fp = {}
    for line in out.splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            fp[k] = v.strip()
    return fp


def new_run_dir(mode, host, label):
    date = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    safe_host = host.replace(".", "_")
    run_id = f"{date}-{mode}-{safe_host}"
    run_dir = os.path.join(RESULTS_ROOT, run_id)
    os.makedirs(run_dir, exist_ok=True)
    meta = {"run_id": run_id, "mode": mode, "host": host, "label": label,
            "started": datetime.datetime.now().isoformat(), "status": "running"}
    with open(os.path.join(run_dir, "meta.json"), "w") as f:
        json.dump(meta, f, indent=2)
    return run_id, run_dir


def update_meta(run_dir, **kwargs):
    meta_path = os.path.join(run_dir, "meta.json")
    with open(meta_path) as f:
        meta = json.load(f)
    meta.update(kwargs)
    with open(meta_path, "w") as f:
        json.dump(meta, f, indent=2)


def deploy_soak(host, run_dir, manifest, label):
    """Install the soak-sample script + timer on the unit, manifest-tracked."""
    with open(SOAK_SAMPLE_SH) as f:
        script = f.read()

    # Manifest-track the results log itself too, not just the deploy control
    # files - so revert can fully restore pre-test state. Always deleted on
    # revert (not restored) so the target never accumulates test history across
    # runs; the harness's own local copy on cm5 (pulled during polling) is the
    # archive - see cm5_harness_notes.md for how to view results after a run.
    manifest.record_always_delete("/opt/clock8002/logs/ltc-soak.csv")

    targets = [
        ("/opt/clock8002/ltc-soak-sample.sh", script),
        ("/etc/systemd/system/ltc-soak.service", SOAK_SERVICE),
        ("/etc/systemd/system/ltc-soak.timer", SOAK_TIMER),
        ("/etc/default/ltc-soak",
         f"SOAK_LABEL={label}\nSOAK_CSV=/opt/clock8002/logs/ltc-soak.csv\n"),
    ]
    for path, content in targets:
        manifest.record_write_file(host, path, run_dir)
        scp_content_to(host, path, content)
    ssh_ok(host, "sudo chmod +x /opt/clock8002/ltc-soak-sample.sh")

    was_enabled = ssh(host, "systemctl is-enabled ltc-soak.timer 2>&1")[1].strip()
    was_active = ssh(host, "systemctl is-active ltc-soak.timer 2>&1")[1].strip()
    manifest.record_enable_service(host, "ltc-soak.timer", was_enabled, was_active)

    # Persistent=true timers keep a last-fired stamp in /var/lib/systemd/timers/
    # keyed by unit NAME, independent of the unit files. A stamp left over from a
    # prior deploy of this same unit name corrupts OnUnitActiveSec's schedule
    # calculation on the next deploy (observed 2026-08-01: timer enabled but
    # `systemctl list-timers` showed NEXT=- and the service never fired again).
    # Not manifest-tracked as a file since it's systemd-internal bookkeeping, not
    # unit config - reverting the enable_service state doesn't touch it either.
    ssh(host, "sudo rm -f /var/lib/systemd/timers/stamp-ltc-soak.timer")

    ssh_ok(host, "sudo systemctl daemon-reload && sudo systemctl enable --now ltc-soak.timer")


def deploy_cycle(host, run_dir, manifest, label, max_cycles, monitor_s):
    """Install the multi-boot cycle script + service on the unit, manifest-tracked.
    Also clears any leftover cycle.count/.stop from a prior run (manifest-tracked,
    so revert restores whatever was there before) so this run starts at cycle 1
    instead of silently resuming a previous run's count."""
    with open(CYCLE_SAMPLE_SH) as f:
        script = f.read()

    # Same rationale as deploy_soak(): the results log is always deleted on
    # revert, not restored - the harness's own local copy is the archive.
    manifest.record_always_delete("/opt/clock8002/logs/ltc-cycle-results.csv")

    targets = [
        ("/opt/clock8002/ltc-cycle.sh", script),
        ("/etc/systemd/system/ltc-cycle.service", CYCLE_SERVICE),
        ("/etc/default/ltc-cycle",
         f"MAX_CYCLES={max_cycles}\nMONITOR_S={monitor_s}\nLABEL={label}\n"),
    ]
    for path, content in targets:
        manifest.record_write_file(host, path, run_dir)
        scp_content_to(host, path, content)
    ssh_ok(host, "sudo chmod +x /opt/clock8002/ltc-cycle.sh")

    for stale in ("/opt/clock8002/ltc-cycle.count", "/opt/clock8002/ltc-cycle.stop"):
        manifest.record_write_file(host, stale, run_dir)
        ssh(host, f"sudo rm -f '{stale}'")

    was_enabled = ssh(host, "systemctl is-enabled ltc-cycle.service 2>&1")[1].strip()
    was_active = ssh(host, "systemctl is-active ltc-cycle.service 2>&1")[1].strip()
    manifest.record_enable_service(host, "ltc-cycle.service", was_enabled, was_active)

    ssh_ok(host, "sudo systemctl daemon-reload && sudo systemctl enable --now ltc-cycle.service")


def pull_cycle_results(host, run_dir):
    """Copy new rows from the unit's cycle results CSV into the run dir on cm5."""
    rc, out, _ = ssh(host, "cat /opt/clock8002/logs/ltc-cycle-results.csv 2>/dev/null", timeout=20)
    if rc == 0 and out:
        _merge_csv_with_harness_ram(os.path.join(run_dir, "ltc-cycle-results.csv"), out)
    return out


def pull_cycle_count(host):
    """Return the current on-unit cycle count, or -1 if unreachable (expected
    while the unit is mid-reboot - not itself a failure)."""
    rc, out, _ = ssh(host, "cat /opt/clock8002/ltc-cycle.count 2>/dev/null", timeout=15)
    if rc == 0 and out.strip().isdigit():
        return int(out.strip())
    return -1


def check_cycle_failure(csv_text):
    """Look for any non-clean outcome in the cycle results CSV. Returns a
    description string if found, else None."""
    if not csv_text:
        return None
    lines = csv_text.strip().splitlines()
    if len(lines) < 2:
        return None
    header = lines[0].split(",")
    try:
        outcome_idx = header.index("outcome")
    except ValueError:
        return None
    for line in lines[1:]:
        fields = line.split(",")
        if len(fields) > outcome_idx and fields[outcome_idx] not in ("clean", ""):
            return f"cycle failure: {line}"
    return None


def _merge_csv_with_harness_ram(local_path, remote_text):
    """Append only the NEW rows of remote_text (vs. what's already in
    local_path) to local_path, tagging each with the harness control process's
    own current RSS/Swap - same treatment as the clock8002/alsa-ltc RSS/Swap
    columns already sampled per-row on the target, but the harness itself runs
    on cm5, so this is the only point its footprint can be attached to a row.
    The raw remote_text is returned unchanged for failure-detection callers,
    which index columns by the target's own header (unaffected by the two
    extra trailing columns added only to the local merged copy)."""
    remote_lines = remote_text.strip("\n").splitlines()
    if not remote_lines:
        return
    local_lines = []
    if os.path.exists(local_path):
        with open(local_path) as f:
            local_lines = f.read().strip("\n").splitlines()
    already = max(0, len(local_lines) - 1) if local_lines else 0
    rss, swap = harness_ram()
    new_rows = remote_lines[1 + already:]
    if not local_lines:
        with open(local_path, "w") as f:
            f.write(remote_lines[0] + ",harness_rss_kb,harness_swap_kb\n")
    if new_rows:
        with open(local_path, "a") as f:
            for row in new_rows:
                f.write(f"{row},{rss if rss is not None else ''},"
                        f"{swap if swap is not None else ''}\n")


def pull_csv(host, run_dir):
    """Copy new rows from the unit's soak CSV into the run dir on cm5."""
    rc, out, _ = ssh(host, "cat /opt/clock8002/logs/ltc-soak.csv 2>/dev/null")
    if rc == 0 and out:
        _merge_csv_with_harness_ram(os.path.join(run_dir, "ltc-soak.csv"), out)
    return out


def check_failure(csv_text):
    """Look for a stall (advancing=NO) or a service failure in the CSV. Returns
    a description string if found, else None."""
    if not csv_text:
        return None
    lines = csv_text.strip().splitlines()
    if len(lines) < 2:
        return None
    header = lines[0].split(",")
    try:
        adv_idx = header.index("advancing")
    except ValueError:
        return None
    for line in lines[1:]:
        fields = line.split(",")
        if len(fields) > adv_idx and fields[adv_idx] == "NO":
            return f"stall detected: {line}"
    return None


def cmd_run_soak(args):
    host = args.host
    check_no_active_run(host, args.force)
    run_id, run_dir = new_run_dir("soak", host, args.label)
    manifest = Manifest(run_dir)

    rss0, swap0 = harness_ram()
    update_meta(run_dir, harness_rss_kb_start=rss0, harness_swap_kb_start=swap0)

    print(f"[{run_id}] capturing pre-test fingerprint...")
    fp_before = fingerprint(host)
    with open(os.path.join(run_dir, "fingerprint-before.json"), "w") as f:
        json.dump(fp_before, f, indent=2)

    print(f"[{run_id}] deploying soak sampler to {host}...")
    deploy_soak(host, run_dir, manifest, args.label)

    print(f"[{run_id}] running for {args.duration}s (poll every {args.poll}s)...")
    deadline = time.time() + args.duration
    failure = None
    while time.time() < deadline:
        time.sleep(min(args.poll, max(0, deadline - time.time())))
        csv_text = pull_csv(host, run_dir)
        failure = check_failure(csv_text)
        if failure:
            print(f"[{run_id}] FAILURE DETECTED: {failure}")
            if args.on_failure == "halt":
                break

    rss1, swap1 = harness_ram()
    update_meta(run_dir, harness_rss_kb_end=rss1, harness_swap_kb_end=swap1)

    outcome = "FAILED" if failure else "PASSED"
    update_meta(run_dir, status="complete", outcome=outcome,
                finished=datetime.datetime.now().isoformat(),
                failure_detail=failure or "")

    if failure:
        print(f"[{run_id}] collecting forensic dumps (auto-collect on failure)...")
        cmd_collect(argparse.Namespace(run=run_id))

    print(f"[{run_id}] reverting all changes made to {host} (auto-revert on completion)...")
    cmd_revert(argparse.Namespace(run=run_id))
    try:
        cmd_verify(argparse.Namespace(run=run_id))
    except SystemExit:
        print(f"[{run_id}] WARNING: unit does not match pre-test fingerprint after "
              f"revert - see fingerprint-before/after.json in {run_dir}")

    print(f"[{run_id}] {outcome}. Results: {run_dir}")


def cmd_run_cycle(args):
    host = args.host
    check_no_active_run(host, args.force)
    run_id, run_dir = new_run_dir("cycle", host, args.label)
    manifest = Manifest(run_dir)

    rss0, swap0 = harness_ram()
    update_meta(run_dir, harness_rss_kb_start=rss0, harness_swap_kb_start=swap0)

    print(f"[{run_id}] capturing pre-test fingerprint...")
    fp_before = fingerprint(host)
    with open(os.path.join(run_dir, "fingerprint-before.json"), "w") as f:
        json.dump(fp_before, f, indent=2)

    print(f"[{run_id}] deploying cycle harness to {host} "
          f"(max_cycles={args.max_cycles}, monitor_s={args.monitor_s})...")
    deploy_cycle(host, run_dir, manifest, args.label, args.max_cycles, args.monitor_s)

    print(f"[{run_id}] running up to {args.max_cycles} reboot cycles "
          f"(poll every {args.poll}s). The unit WILL reboot repeatedly - SSH "
          f"drops during this are expected, not harness errors.")
    failure = None
    unreachable_s = 0
    while True:
        time.sleep(args.poll)
        count = pull_cycle_count(host)
        if count < 0:
            unreachable_s += args.poll
            print(f"[{run_id}] unit unreachable (likely mid-reboot), "
                  f"{unreachable_s}s so far...")
            if unreachable_s > args.stall_timeout:
                failure = (f"unit unreachable for over {args.stall_timeout}s - "
                           f"treating as wedged, not a normal reboot window")
                break
            continue
        unreachable_s = 0

        csv_text = pull_cycle_results(host, run_dir)
        failure = check_cycle_failure(csv_text)
        if failure:
            print(f"[{run_id}] FAILURE DETECTED: {failure}")
            if args.on_failure == "halt":
                break
        # ltc-cycle.count is bumped to n at the START of cycle n (before it
        # runs), so it hits max_cycles the instant the FINAL cycle begins, not
        # when it finishes. Use completed rows in the results CSV instead.
        completed = max(0, len(csv_text.strip().splitlines()) - 1) if csv_text else 0
        if completed >= args.max_cycles:
            print(f"[{run_id}] all {args.max_cycles} cycles complete")
            break

    rss1, swap1 = harness_ram()
    update_meta(run_dir, harness_rss_kb_end=rss1, harness_swap_kb_end=swap1)

    outcome = "FAILED" if failure else "PASSED"
    update_meta(run_dir, status="complete", outcome=outcome,
                finished=datetime.datetime.now().isoformat(),
                failure_detail=failure or "")

    if failure:
        print(f"[{run_id}] collecting forensic dumps (auto-collect on failure)...")
        cmd_collect(argparse.Namespace(run=run_id))

    print(f"[{run_id}] reverting all changes made to {host} (auto-revert on completion)...")
    cmd_revert(argparse.Namespace(run=run_id))
    try:
        cmd_verify(argparse.Namespace(run=run_id))
    except SystemExit:
        print(f"[{run_id}] WARNING: unit does not match pre-test fingerprint after "
              f"revert - see fingerprint-before/after.json in {run_dir}")

    print(f"[{run_id}] {outcome}. Results: {run_dir}")


def cmd_status(args):
    run_dir = os.path.join(RESULTS_ROOT, args.run)
    with open(os.path.join(run_dir, "meta.json")) as f:
        meta = json.load(f)
    print(json.dumps(meta, indent=2))
    csv_path = os.path.join(run_dir, "ltc-soak.csv")
    if os.path.exists(csv_path):
        with open(csv_path) as f:
            lines = f.readlines()
        print(f"samples so far: {max(0, len(lines) - 1)}")
        if len(lines) > 1:
            print("last sample:", lines[-1].strip())
    cycle_csv_path = os.path.join(run_dir, "ltc-cycle-results.csv")
    if os.path.exists(cycle_csv_path):
        with open(cycle_csv_path) as f:
            lines = f.readlines()
        print(f"cycles so far: {max(0, len(lines) - 1)}")
        if len(lines) > 1:
            print("last cycle:", lines[-1].strip())


def cmd_collect(args):
    run_dir = os.path.join(RESULTS_ROOT, args.run)
    with open(os.path.join(run_dir, "meta.json")) as f:
        meta = json.load(f)
    host = meta["host"]
    pull_csv(host, run_dir)
    pull_cycle_results(host, run_dir)
    # Pull any wedge forensic dumps present on the unit.
    rc, out, _ = ssh(host, "ls /opt/clock8002/logs/xhci-wedge-*.log 2>/dev/null")
    dumps = [l for l in out.splitlines() if l.strip()]
    if dumps:
        dump_dir = os.path.join(run_dir, "wedge-dumps")
        os.makedirs(dump_dir, exist_ok=True)
        for remote_path in dumps:
            rc2, content, _ = ssh(host, f"cat '{remote_path}'")
            local_name = os.path.basename(remote_path)
            with open(os.path.join(dump_dir, local_name), "w") as f:
                f.write(content)
        print(f"collected {len(dumps)} wedge dump(s) into {dump_dir}")
    else:
        print("no wedge dumps found on unit")


def cmd_revert(args):
    run_dir = os.path.join(RESULTS_ROOT, args.run)
    with open(os.path.join(run_dir, "meta.json")) as f:
        meta = json.load(f)
    host = meta["host"]
    manifest = Manifest(run_dir)
    entries = manifest.read_entries()

    print(f"reverting {len(entries)} recorded change(s) on {host}...")
    for e in reversed(entries):
        if e["action"] == "write_file_new":
            print(f"  removing {e['target']} (was new)")
            ssh(host, f"sudo rm -f '{e['target']}'")
        elif e["action"] == "always_delete":
            print(f"  removing {e['target']} (generated file, not restored)")
            ssh(host, f"sudo rm -f '{e['target']}'")
        elif e["action"] == "write_file_existed":
            print(f"  restoring {e['target']} from backup")
            with open(e["backup_file"]) as f:
                content = f.read()
            scp_content_to(host, e["target"], content)
        elif e["action"] == "enable_service":
            # detail is "was_enabled=<enabled|disabled> was_active=<active|inactive>".
            # Parse key=value properly - a naive "enabled" in e["detail"] substring
            # check is ALWAYS true because the key name "was_enabled" itself
            # contains "enabled", regardless of the actual value. That bug meant
            # revert never disabled a previously-disabled service, leaving a
            # dangling timers.target.wants/ symlink after the unit file was
            # removed (observed 2026-08-01, corrupted the next deploy's timer
            # scheduling).
            detail = dict(kv.split("=", 1) for kv in e["detail"].split() if "=" in kv)
            was_enabled = detail.get("was_enabled") == "enabled"
            was_active = detail.get("was_active") == "active"
            unit = e["target"]
            if not was_enabled:
                ssh(host, f"sudo systemctl disable {unit} 2>/dev/null")
            if not was_active:
                ssh(host, f"sudo systemctl stop {unit} 2>/dev/null")
            # Persistent=true timers leave a last-fired stamp in
            # /var/lib/systemd/timers/ keyed by unit name - this is systemd's own
            # bookkeeping, not unit config, so it was never manifest-tracked as a
            # file, but it's still an artifact of this run and should not survive
            # cleanup (see cm5_harness_notes.md for why a stale stamp corrupts the
            # next deploy's OnUnitActiveSec scheduling).
            if unit.endswith(".timer"):
                ssh(host, f"sudo rm -f '/var/lib/systemd/timers/stamp-{unit}'")

    ssh(host, "sudo systemctl daemon-reload")
    fields = {"reverted": datetime.datetime.now().isoformat()}
    # A run killed before cmd_run_* could set status=complete stays "running"
    # forever, and check_no_active_run then blocks every future run against that
    # host (observed 2026-08-02: a cancelled soak blocked the next deploy even
    # after a clean revert). Reverting a run is the point at which it is
    # provably no longer active, so settle the status here. Only touch it while
    # it still reads "running" - never downgrade a completed run.
    if meta.get("status") == "running":
        fields["status"] = "cancelled"
        fields["outcome"] = "cancelled"
    update_meta(run_dir, **fields)
    print("revert complete. Run 'verify' to confirm.")


def cmd_verify(args):
    run_dir = os.path.join(RESULTS_ROOT, args.run)
    with open(os.path.join(run_dir, "meta.json")) as f:
        meta = json.load(f)
    host = meta["host"]

    before_path = os.path.join(run_dir, "fingerprint-before.json")
    if not os.path.exists(before_path):
        print("no pre-test fingerprint recorded; cannot verify")
        sys.exit(1)
    with open(before_path) as f:
        fp_before = json.load(f)

    fp_after = fingerprint(host)
    with open(os.path.join(run_dir, "fingerprint-after.json"), "w") as f:
        json.dump(fp_after, f, indent=2)

    diffs = {k: (fp_before.get(k), fp_after.get(k)) for k in fp_before
             if fp_before.get(k) != fp_after.get(k)}
    if diffs:
        print("VERIFY FAILED - unit does not match pre-test state:")
        for k, (b, a) in diffs.items():
            print(f"  {k}: before={b!r} after={a!r}")
        sys.exit(1)
    else:
        print("VERIFY PASSED - unit matches pre-test fingerprint")


def cmd_list(args):
    if not os.path.isdir(RESULTS_ROOT):
        print("no runs yet")
        return
    for run_id in sorted(os.listdir(RESULTS_ROOT)):
        meta_path = os.path.join(RESULTS_ROOT, run_id, "meta.json")
        if os.path.exists(meta_path):
            with open(meta_path) as f:
                meta = json.load(f)
            print(f"{run_id}: {meta.get('status')} {meta.get('outcome', '')} "
                  f"host={meta.get('host')} label={meta.get('label')!r}")


def main():
    p = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="command", required=True)

    run_p = sub.add_parser("run", help="start a test run")
    run_sub = run_p.add_subparsers(dest="mode", required=True)

    soak_p = run_sub.add_parser("soak", help="continuous passive soak test")
    soak_p.add_argument("--host", required=True)
    soak_p.add_argument("--duration", type=int, default=3600,
                         help="seconds to run (default 3600)")
    soak_p.add_argument("--poll", type=int, default=60,
                         help="seconds between checks (default 60)")
    soak_p.add_argument("--label", default="soak")
    soak_p.add_argument("--on-failure", choices=["halt", "continue"], default="halt")
    soak_p.add_argument("--force", action="store_true",
                         help="deploy even if another run is marked 'running' "
                              "against this host (see check_no_active_run)")
    soak_p.set_defaults(func=cmd_run_soak)

    cycle_p = run_sub.add_parser("cycle", help="multi-boot reboot-cycle stability test")
    cycle_p.add_argument("--host", required=True)
    cycle_p.add_argument("--max-cycles", type=int, default=20,
                          help="number of reboot cycles to run (default 20)")
    cycle_p.add_argument("--monitor-s", type=int, default=240,
                          help="seconds to monitor for stalls after each boot "
                               "settles, per cycle (default 240)")
    cycle_p.add_argument("--poll", type=int, default=30,
                          help="seconds between progress checks (default 30)")
    cycle_p.add_argument("--stall-timeout", type=int, default=900,
                          help="seconds the unit may stay unreachable before "
                               "it's treated as wedged rather than mid-reboot "
                               "(default 900)")
    cycle_p.add_argument("--label", default="cycle")
    cycle_p.add_argument("--on-failure", choices=["halt", "continue"], default="halt")
    cycle_p.add_argument("--force", action="store_true",
                          help="deploy even if another run is marked 'running' "
                               "against this host (see check_no_active_run)")
    cycle_p.set_defaults(func=cmd_run_cycle)

    for stub_mode in ("accumulate", "ab"):
        stub_p = run_sub.add_parser(stub_mode, help=f"[NOT YET IMPLEMENTED] {stub_mode} mode")
        stub_p.set_defaults(func=lambda a, m=stub_mode: (_ for _ in ()).throw(
            NotImplementedError(
                f"'{m}' mode is not implemented yet - see "
                f"/memories/session/plan.md phase 5. Reference implementations "
                f"exist in tools/cm5-harness/salvage-reference/.")))

    status_p = sub.add_parser("status")
    status_p.add_argument("--run", required=True)
    status_p.set_defaults(func=cmd_status)

    collect_p = sub.add_parser("collect")
    collect_p.add_argument("--run", required=True)
    collect_p.set_defaults(func=cmd_collect)

    revert_p = sub.add_parser("revert")
    revert_p.add_argument("--run", required=True)
    revert_p.set_defaults(func=cmd_revert)

    verify_p = sub.add_parser("verify")
    verify_p.add_argument("--run", required=True)
    verify_p.set_defaults(func=cmd_verify)

    list_p = sub.add_parser("list")
    list_p.set_defaults(func=cmd_list)

    args = p.parse_args()
    os.makedirs(RESULTS_ROOT, exist_ok=True)
    args.func(args)


if __name__ == "__main__":
    main()
