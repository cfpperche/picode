// The WSL Disk management window (Phase 2, docs/plans/desktop-v2.md): the
// shell drives the tested Go CLIs as subprocesses and renders their outcome.
// The shell never reimplements disk logic — `picode-desktop.exe` next to this
// executable is the tool, exactly the way `provision` uses the binary inside
// the distro.

use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader};
use std::process::{Command, Stdio};
use tauri::Emitter;

/// Spawns a Go CLI with no console window. Lookup order: next to the shell
/// (self-contained install), then the canonical PiCode folder that
/// `make desktop-restart` keeps fresh.
pub(crate) fn run_cli(args: &[&str]) -> Result<String, String> {
    let exe = tool_exe().ok_or_else(|| {
        "picode-desktop.exe was not found next to the shell or in %LOCALAPPDATA%\\PiCode — reinstall PiCode Desktop".to_string()
    })?;
    let mut cmd = Command::new(&exe);
    cmd.args(args);
    hide_console(&mut cmd);
    let out = cmd.output().map_err(|e| e.to_string())?;
    let mut text = String::from_utf8_lossy(&out.stdout).into_owned();
    if text.trim().is_empty() {
        text = String::from_utf8_lossy(&out.stderr).into_owned();
    }
    if text.trim().is_empty() {
        text = format!("exit status {}", out.status.code().unwrap_or(-1));
    }
    Ok(text)
}

// A subprocess tool must not flash a console window over the app.
#[cfg(windows)]
pub(crate) fn hide_console(cmd: &mut Command) {
    use std::os::windows::process::CommandExt;
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    cmd.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(windows))]
pub(crate) fn hide_console(_cmd: &mut Command) {}

// tool_exe finds the Go CLI: next to the shell first (a self-contained
// folder), then the canonical PiCode install folder.
pub(crate) fn tool_exe() -> Option<std::path::PathBuf> {
    if let Ok(exe) = std::env::current_exe() {
        if let Some(dir) = exe.parent() {
            let candidate = dir.join("picode-desktop.exe");
            if candidate.exists() {
                return Some(candidate);
            }
        }
    }
    if let Ok(local) = std::env::var("LOCALAPPDATA") {
        let candidate = std::path::Path::new(&local)
            .join("PiCode")
            .join("picode-desktop.exe");
        if candidate.exists() {
            return Some(candidate);
        }
    }
    None
}

/// One compact outcome, exactly the object `disk-compact --json` prints.
#[derive(Debug, Deserialize, Serialize, Default)]
pub struct CompactOutcome {
    pub distro: String,
    #[serde(default)]
    pub method: String,
    #[serde(default)]
    pub plan: String,
    #[serde(default)]
    pub held: i64,
    #[serde(default)]
    pub before: i64,
    #[serde(default)]
    pub free: i64,
    #[serde(default)]
    pub refused: String,
    #[serde(default)]
    pub note: String,
    #[serde(default)]
    pub error: String,
    #[serde(default)]
    pub after: Option<i64>,
    #[serde(default)]
    pub returned: Option<i64>,
    #[serde(default)]
    pub sparse: Option<bool>,
    /// The flow reached the stop: the distro's sessions ended even when the
    /// run failed afterwards.
    #[serde(default)]
    pub stopped: bool,
}

fn parse_outcome(text: &str) -> Result<CompactOutcome, String> {
    let start = text
        .find('{')
        .ok_or_else(|| format!("no JSON in the tool output: {text}"))?;
    serde_json::from_str(text[start..].trim_end())
        .map_err(|e| format!("compact outcome JSON: {e} — raw: {text}"))
}

/// The event every streamed Management operation reports on. The payload is
/// the tool's own progress object plus `op`, so the window knows which of its
/// views a step belongs to.
pub(crate) const PROGRESS_EVENT: &str = "mgmt-progress";

/// Runs the Go tool with piped stdout and no console, forwarding every line
/// that carries `progress` as a `mgmt-progress` event tagged with `op`, and
/// returns the first line that does not — the outcome, by contract the last
/// line. One reader for scan, compact and clean: the three copies this
/// replaced had already drifted (the page listened on a name none emitted).
/// One Management job at a time across every window: `busy` lives in the
/// page, and a page closed and reopened mid-run would otherwise start a
/// second compact over the first.
static JOB: std::sync::Mutex<()> = std::sync::Mutex::new(());

pub(crate) fn stream_cli(
    app: &tauri::AppHandle,
    op: &str,
    run: &str,
    args: &[&str],
) -> Result<serde_json::Value, String> {
    let _job = match JOB.try_lock() {
        Ok(g) => g,
        Err(std::sync::TryLockError::Poisoned(p)) => p.into_inner(),
        Err(std::sync::TryLockError::WouldBlock) => {
            return Err("another Management job is still running — wait for it to finish".into())
        }
    };
    let exe = tool_exe().ok_or_else(|| {
        "picode-desktop.exe was not found next to the shell or in %LOCALAPPDATA%\\PiCode — reinstall PiCode Desktop".to_string()
    })?;
    let mut cmd = Command::new(&exe);
    cmd.args(args);
    hide_console(&mut cmd);
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::piped());
    let mut child = cmd.spawn().map_err(|e| e.to_string())?;

    // A stream with no stdout is a tool that cannot answer; a stream that
    // ends without an outcome is a run we must not pretend succeeded.
    let stdout = child.stdout.take().ok_or("the tool opened no stdout")?;
    // Drain stderr on its own thread so a chatty tool cannot fill the pipe
    // and stall stdout; it is the diagnosis when no outcome arrives.
    let stderr = child.stderr.take();
    let err_reader = std::thread::spawn(move || {
        let mut text = String::new();
        if let Some(mut e) = stderr {
            let _ = std::io::Read::read_to_string(&mut e, &mut text);
        }
        text
    });
    let outcome = read_stream(BufReader::new(stdout), |mut step| {
        step.insert("op".into(), serde_json::Value::String(op.into()));
        // The page's own token for this call: steps from an earlier run (or
        // another window) arriving late are told apart by it.
        step.insert("run".into(), serde_json::Value::String(run.into()));
        let _ = app.emit(PROGRESS_EVENT, serde_json::Value::Object(step));
    });
    // Reading stops at the outcome; whatever follows is not read, so the
    // child is waited for rather than left holding a full pipe.
    let status = child.wait().map_err(|e| e.to_string())?;
    let stderr = err_reader.join().unwrap_or_default();
    outcome.map_err(|e| {
        let why = stderr.trim();
        if why.is_empty() {
            format!("{e} (exit {status})")
        } else {
            format!("{e} (exit {status}): {why}")
        }
    })
}

/// The pure half of stream_cli: sorts lines into progress steps and the
/// outcome. Non-JSON noise (a login banner) is skipped, not fatal.
pub(crate) fn read_stream<R: BufRead>(
    reader: R,
    mut on_step: impl FnMut(serde_json::Map<String, serde_json::Value>),
) -> Result<serde_json::Value, String> {
    let mut reader = reader;
    let mut buf = Vec::new();
    loop {
        buf.clear();
        // Bytes, then lossy text: one non-UTF-8 line (a translated wsl.exe
        // message) must not abort a run — for clean, dropping the pipe
        // mid-prune would kill the prune in the distro.
        if reader.read_until(b'\n', &mut buf).map_err(|e| e.to_string())? == 0 {
            break;
        }
        let line = String::from_utf8_lossy(&buf);
        let Some(start) = line.find('{') else { continue };
        let Ok(serde_json::Value::Object(obj)) =
            serde_json::from_str::<serde_json::Value>(line[start..].trim_end())
        else {
            continue;
        };
        if obj.contains_key("progress") {
            on_step(obj);
            continue;
        }
        return Ok(serde_json::Value::Object(obj));
    }
    Err("the tool ended without an outcome".into())
}

/// The full two-sided report as a scan: `disk --json --stream` reads the
/// Windows half, then the distro half, and each one reaches the window as a
/// `mgmt-progress` event (op "scan") the moment it lands. The report itself
/// is the return value.
#[tauri::command(async)]
pub fn disk_report(app: tauri::AppHandle, run: Option<String>) -> Result<serde_json::Value, String> {
    let run = run.unwrap_or_default();
    match stream_cli(&app, "scan", &run, &["disk", "--json", "--stream"]) {
        // An older picode-desktop.exe (the %LOCALAPPDATA% fallback) predates
        // --stream: read the report whole rather than failing the window.
        Err(e) if e.contains("-stream") => {
            let text = run_cli(&["disk", "--json"])?;
            serde_json::from_str(text[text.find('{').ok_or(e)?..].trim_end())
                .map_err(|e| format!("disk report JSON: {e}"))
        }
        other => other,
    }
}

/// The compact flow, gates included: readiness interlock, stop, convert,
/// restart, measured outcome. Steps stream as `mgmt-progress` (op
/// "compact"). Refusals (someone working) come back in the outcome's
/// `refused` field — nothing was stopped.
#[tauri::command(async)]
pub fn disk_compact(app: tauri::AppHandle, run: Option<String>) -> Result<CompactOutcome, String> {
    // No claim about the distro here: a missing tool or an early failure
    // stopped nothing, and a failure after the stop arrives as an outcome
    // with `error` and `stopped` set, which the page states.
    let v = stream_cli(&app, "compact", &run.unwrap_or_default(), &["disk-compact", "--yes", "--json"])?;
    serde_json::from_value(v).map_err(|e| format!("compact outcome JSON: {e}"))
}

/// The plan without stopping anything — what the window shows before asking.
#[tauri::command(async)]
pub fn disk_compact_dry_run() -> Result<CompactOutcome, String> {
    let text = run_cli(&["disk-compact", "--json", "--dry-run"])?;
    parse_outcome(&text)
}

#[cfg(test)]
mod tests {
    use super::read_stream;

    #[test]
    fn stream_sorts_steps_from_the_outcome() {
        let text = "Welcome to Ubuntu\n\
{\"progress\":\"Reading\",\"stage\":\"windows\",\"state\":\"running\"}\n\
not json {\n\
{\"progress\":\"Read\",\"stage\":\"windows\",\"state\":\"done\"}\n\
{\"at\":\"2026-09-22T00:00:00Z\",\"held\":5}\n\
{\"progress\":\"after the outcome is never read\"}\n";
        let mut bytes = text.as_bytes().to_vec();
        bytes.splice(0..0, b"\xff\xfe bad bytes\n".iter().copied());
        let mut steps = Vec::new();
        let out = read_stream(&bytes[..], |s| steps.push(s)).expect("an outcome");
        assert_eq!(steps.len(), 2);
        assert_eq!(steps[1]["state"], "done");
        assert_eq!(out["held"], 5);
    }

    #[test]
    fn stream_without_outcome_is_an_error() {
        let text = "{\"progress\":\"Cleaning\"}\n";
        assert!(read_stream(text.as_bytes(), |_| {}).is_err());
    }
}
