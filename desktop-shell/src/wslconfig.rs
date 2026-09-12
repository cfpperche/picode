//! `%USERPROFILE%\.wslconfig` — read, and write with a backup.
//!
//! The file is the user's: unknown keys, other sections and comments are
//! theirs and stay untouched. A write only touches the keys the window
//! owns (the table in lib.rs, straight from Microsoft's documented
//! `.wslconfig` settings), after copying the original to `.wslconfig.bak`.
//! Changes apply at the next full WSL restart — never from this window;
//! stopping WSL ends sessions, and that action lives in the Give back flow
//! where the cost is stated.

use picode_shell::{edit, parse_owned, validate_value, OWNED};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WslValue {
    /// The section the key was found under — where a write will edit it.
    pub section: String,
    pub key: String,
    /// Absent when the file does not set the key (WSL's default applies).
    #[serde(rename = "value")]
    pub value: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WslConfig {
    pub path: String,
    pub exists: bool,
    /// Every owned key with its current value; absent keys are the ones the
    /// file does not set.
    pub values: Vec<WslValue>,
    /// The whole file, so the window can show what a save will touch.
    pub raw: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WslConfigSaved {
    pub path: String,
    pub backup: String,
    pub note: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct WslEdit {
    pub section: String,
    pub key: String,
    /// None (or an empty string) removes the key: back to WSL's default.
    #[serde(default)]
    pub value: Option<String>,
}

fn config_path() -> PathBuf {
    let profile = std::env::var("USERPROFILE").unwrap_or_default();
    PathBuf::from(profile).join(".wslconfig")
}

#[tauri::command(async)]
pub fn wslconfig_read() -> Result<WslConfig, String> {
    let path = config_path();
    let raw = std::fs::read_to_string(&path).unwrap_or_default();
    let exists = path.exists();
    let current = parse_owned(&raw);
    let values = OWNED
        .iter()
        .map(|(sec, key, _)| {
            let found = current
                .iter()
                .find(|((fs, fk), _)| fk.eq_ignore_ascii_case(key))
                .map(|((fs, _), v)| (fs.clone(), Some(v.clone())));
            let (section, value) = found.unwrap_or_else(|| ((*sec).to_string(), None));
            WslValue {
                section,
                key: key.to_string(),
                value,
            }
        })
        .collect();
    Ok(WslConfig {
        path: path.to_string_lossy().into_owned(),
        exists,
        values,
        raw,
    })
}

/// Saves the named keys. The backup is written before the file, so a crash
/// mid-write leaves the original recoverable either way. Keys unknown to
/// the table are refused — the window only ever sends what it rendered.
#[tauri::command(async)]
pub fn wslconfig_write(edits: Vec<WslEdit>) -> Result<WslConfigSaved, String> {
    if edits.is_empty() {
        return Err("nothing to save".into());
    }
    for e in &edits {
        let v = e.value.as_deref().unwrap_or("");
        validate_value(&e.section, &e.key, v).map_err(|m| format!("{}: {}", e.key, m))?;
    }

    let path = config_path();
    let raw = std::fs::read_to_string(&path).unwrap_or_default();
    let backup = path.with_extension("wslconfig.bak");
    if path.exists() {
        std::fs::copy(&path, &backup).map_err(|e| format!("backup failed: {}", e))?;
    }

    let tuple_edits: Vec<(&str, &str, Option<&str>)> = edits
        .iter()
        .map(|e| (e.section.as_str(), e.key.as_str(), e.value.as_deref()))
        .collect();
    let updated = edit(&raw, &tuple_edits);
    std::fs::write(&path, &updated).map_err(|e| format!("write failed: {}", e))?;

    Ok(WslConfigSaved {
        path: path.to_string_lossy().into_owned(),
        backup: backup.to_string_lossy().into_owned(),
        note: "Saved. It applies at the next full WSL restart — stopping the distro from the Give back flow is one.".into(),
    })
}
