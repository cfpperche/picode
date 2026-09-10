"""One private native checkpoint and ordered fence per terminal; no locked network."""
import fcntl
import json
import os
import re
import tempfile


def process_start(pid):
    with open("/proc/%d/stat" % pid) as f:
        return f.read().rsplit(")", 1)[1].split()[19]


def save_observation(root, term, report):
    if not re.fullmatch(r"[A-Za-z0-9_-]{1,160}", term):
        return False
    directory = os.path.join(root, "native-observations")
    path = os.path.join(directory, term + ".json")
    fence_path = os.path.join(directory, term + ".lock")
    fence_saved = False
    try:
        os.makedirs(directory, mode=0o700, exist_ok=True)
        starting = report.get("action") == "start"
        blocked = os.path.exists(fence_path) and os.stat(fence_path).st_mode & 0o777 != 0o600
        # Only a wrapper start (before its child runs) may reset a blocked fence.
        if starting and os.path.exists(fence_path):
            os.chmod(fence_path, 0o600)
        fd = os.open(fence_path, os.O_RDWR | os.O_CREAT, 0o600)
        with os.fdopen(fd, "r+") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            if os.fstat(lock.fileno()).st_mode & 0o777 != 0o600:
                return False  # Another writer may have blocked our already-open fd.
            previous_raw = lock.read()
            try:
                previous = json.loads(previous_raw) if previous_raw else {}
                if previous and (previous.get("version") != 1 or previous.get("termId") != term or not previous.get("runId") or not previous.get("procStart") or not previous.get("bootId") or not isinstance(previous.get("pid"), int) or not isinstance(previous.get("sessionSeq"), int)):
                    raise ValueError("invalid fence")
            except (ValueError, AttributeError):
                if not starting:
                    return False
                previous = {}
            pid = report.get("pid", 0)
            run = report.get("runId", "")
            if blocked and previous.get("runId") == run:
                os.fchmod(lock.fileno(), 0o000)
                return False
            if not pid or not run:
                return False
            start = process_start(pid)
            with open("/proc/sys/kernel/random/boot_id") as f:
                boot = f.read().strip()
            same = previous.get("runId") == run and previous.get("pid") == pid and previous.get("procStart") == start and previous.get("bootId") == boot
            if same:
                if report["sessionSeq"] <= previous.get("sessionSeq", 0):
                    return False
                if previous.get("action") == "end":
                    return False
                if previous.get("codexHooks") and report.get("source") == "codex-notify":
                    return False
            elif previous and not starting:
                try:
                    if previous.get("bootId") == boot and process_start(previous["pid"]) == previous.get("procStart"):
                        return False
                except (OSError, ValueError, KeyError):
                    pass
            record = {k: report[k] for k in ("cli", "runId", "pid", "state", "sessionId", "sessionPath", "sessionSeq", "source", "action") if k in report}
            record.update(version=1, termId=term, procStart=start, bootId=boot,
                          codexHooks=(same and previous.get("codexHooks", False)) or report.get("source") == "codex-hook")
            # Keep the sequence/source/end fence even if checkpoint publication fails.
            lock.seek(0)
            lock.truncate()
            json.dump(record, lock)
            lock.flush()
            os.fsync(lock.fileno())
            fence_saved = True
            fd, temporary = tempfile.mkstemp(prefix=".observation-", dir=directory)
            try:
                with os.fdopen(fd, "w") as f:
                    json.dump(record, f)
                    f.flush()
                    os.fsync(f.fileno())
                os.replace(temporary, path)
            finally:
                if os.path.exists(temporary):
                    os.unlink(temporary)
            report["observationVersion"] = 1
            return True
    except (OSError, ValueError, KeyError, IndexError):
        if not fence_saved:
            # Lost ordering requires a fresh wrapper, never a delayed Idle event.
            try:
                os.chmod(fence_path, 0o000)
            except OSError:
                try:
                    with open(fence_path, "w") as f:
                        f.write("blocked\n")
                except OSError:
                    pass
        try:
            os.unlink(path)
        except OSError:
            pass
        return False


if __name__ == "__main__":
    import sys
    import time
    term, action, cli, run, pid = sys.argv[1:]
    save_observation(os.path.dirname(__file__), term, {
        "action": action, "cli": cli, "runId": run, "pid": int(pid),
        "sessionSeq": time.time_ns(),
    })
