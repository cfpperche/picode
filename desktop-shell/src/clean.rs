//! The distro's prunable caches, as `picode-desktop clean` reports them, and
//! the same streaming contract as the compact: one progress object per line,
//! the final outcome as the last line.

use crate::disk::{hide_console, run_cli, tool_exe};
use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader};
use std::process::{Command, Stdio};
use tauri::Emitter;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CleanCache {
    pub id: String,
    pub title: String,
    pub kind: String,
    pub bytes: i64,
    pub how: String,
    pub note: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CleanList {
    pub at: String,
    pub consumers: Vec<CleanCache>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CleanResult {
    pub id: String,
    pub before: i64,
    pub after: i64,
    pub freed: i64,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CleanOutcome {
    pub results: Vec<CleanResult>,
    #[serde(rename = "totalFreed")]
    pub total_freed: i64,
}

/// The prunable caches with fresh sizes. Read-only.
#[tauri::command(async)]
pub fn clean_list() -> Result<CleanList, String> {
    let text = run_cli(&["clean", "--list", "--json"])?;
    let start = text.find('{').ok_or("the tool printed no JSON")?;
    serde_json::from_str(text[start..].trim_end()).map_err(|e| e.to_string())
}

/// Prunes the named caches through the in-distro tool. The tool refuses
/// anything that is not a cache — including Pi sessions and the database,
/// even when asked for by exact name. Steps stream to the window as
/// `mgmt-progress` events; the outcome is the last line or a failure.
#[tauri::command(async)]
pub fn clean_apply(app: tauri::AppHandle, ids: Vec<String>) -> Result<CleanOutcome, String> {
    if ids.is_empty() {
        return Err("no caches selected".into());
    }
    let exe = tool_exe().ok_or_else(|| {
        "picode-desktop.exe was not found next to the shell or in %LOCALAPPDATA%\\PiCode — reinstall PiCode Desktop".to_string()
    })?;
    let joined = ids.join(",");
    let mut cmd = Command::new(&exe);
    cmd.args(["clean", "--apply", &joined, "--yes", "--json"]);
    hide_console(&mut cmd);
    cmd.stdout(Stdio::piped());
    let mut child = cmd.spawn().map_err(|e| e.to_string())?;

    let stdout = child.stdout.take().ok_or("the tool opened no stdout")?;
    let reader = BufReader::new(stdout);
    let mut outcome: Option<CleanOutcome> = None;
    for line in reader.lines() {
        let line = line.map_err(|e| e.to_string())?;
        if line.trim().is_empty() {
            continue;
        }
        if let Some(start) = line.find('{') {
            if let Ok(o) = serde_json::from_str::<CleanOutcome>(line[start..].trim_end()) {
                outcome = Some(o);
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
            "the clean ended without an outcome (exit {})",
            status
        )),
    }
}
