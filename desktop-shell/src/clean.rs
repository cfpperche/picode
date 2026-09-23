//! The distro's prunable caches, as `picode-desktop clean` reports them, and
//! the same streaming contract as the compact: one progress object per line,
//! the final outcome as the last line.

use crate::disk::{run_cli, stream_cli};
use serde::{Deserialize, Serialize};

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
/// `mgmt-progress` events (op "clean"); the outcome is the last line or a
/// failure.
#[tauri::command(async)]
pub fn clean_apply(
    app: tauri::AppHandle,
    ids: Vec<String>,
    run: Option<String>,
) -> Result<CleanOutcome, String> {
    if ids.is_empty() {
        return Err("no caches selected".into());
    }
    let joined = ids.join(",");
    let v = stream_cli(&app, "clean", &run.unwrap_or_default(), &["clean", "--apply", &joined, "--yes", "--json"])?;
    // A refusal is an object with `refused`, not an outcome — say it plainly
    // instead of reporting a missing `results` field.
    if let Some(why) = v.get("refused").and_then(|r| r.as_str()) {
        return Err(format!("not run — {why}"));
    }
    serde_json::from_value(v).map_err(|e| format!("clean outcome JSON: {e}"))
}
