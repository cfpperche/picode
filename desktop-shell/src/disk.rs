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
}

fn parse_outcome(text: &str) -> Result<CompactOutcome, String> {
    let start = text
        .find('{')
        .ok_or_else(|| format!("no JSON in the tool output: {text}"))?;
    serde_json::from_str(text[start..].trim_end())
        .map_err(|e| format!("compact outcome JSON: {e} — raw: {text}"))
}

/// The full two-sided report: `disk --json` already measures Windows and the
/// distro in one shot.
#[tauri::command(async)]
pub fn disk_report() -> Result<String, String> {
    run_cli(&["disk", "--json"])
}

/// The compact flow, gates included: readiness interlock, stop, convert,
/// restart, measured outcome. Refusals (someone working) come back in the
/// outcome's `refused` field — the window renders them, nothing was stopped.
/// The compact flow as a stream. The Go tool prints one progress object per
/// line and the final outcome as the last line; this command forwards each
/// step to the window as a `disk-progress` event and returns the outcome.
/// Refusals (someone working) come back in `refused` — nothing was stopped.
#[tauri::command(async)]
pub fn disk_compact(app: tauri::AppHandle) -> Result<CompactOutcome, String> {
    let exe = tool_exe().ok_or_else(|| {
        "picode-desktop.exe was not found next to the shell or in %LOCALAPPDATA%\\PiCode — reinstall PiCode Desktop".to_string()
    })?;
    let mut cmd = Command::new(&exe);
    cmd.args(["disk-compact", "--yes", "--json"]);
    hide_console(&mut cmd);
    cmd.stdout(Stdio::piped());
    let mut child = cmd.spawn().map_err(|e| e.to_string())?;

    // A stream with no stdout is a tool that cannot answer; a stream that
    // ends without an outcome is a run we must not pretend succeeded.
    let stdout = child.stdout.take().ok_or("the tool opened no stdout")?;
    let reader = BufReader::new(stdout);
    let mut outcome: Option<CompactOutcome> = None;
    for line in reader.lines() {
        let line = line.map_err(|e| e.to_string())?;
        if line.trim().is_empty() {
            continue;
        }
        if let Some(start) = line.find('{') {
            if let Ok(v) = serde_json::from_str::<CompactOutcome>(line[start..].trim_end()) {
                outcome = Some(v);
                break; // the outcome is the last line by contract
            }
        }
        if let Ok(v) = serde_json::from_str::<std::collections::HashMap<String, String>>(&line) {
            if let Some(step) = v.get("progress") {
                let _ = app.emit("mgmt-progress", step.clone());
            }
        }
    }

    let status = child.wait().map_err(|e| e.to_string())?;
    match outcome {
        Some(o) => Ok(o),
        None => Err(format!(
            "the compact ended without an outcome (exit {}) — the distro was restarted by the flow itself",
            status
        )),
    }
}

/// The plan without stopping anything — what the window shows before asking.
#[tauri::command(async)]
pub fn disk_compact_dry_run() -> Result<CompactOutcome, String> {
    let text = run_cli(&["disk-compact", "--json", "--dry-run"])?;
    parse_outcome(&text)
}

/// The tray's disk verdict: one `disk --json` folded to the line facts. A
/// failed half reads as unread, never as a zero.
pub fn line_facts() -> Result<crate::diskline::Facts, String> {
    let text = run_cli(&["disk", "--json"])?;
    crate::diskline::parse_report(&text)
}
