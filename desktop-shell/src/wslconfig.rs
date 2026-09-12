//! `%USERPROFILE%\.wslconfig` — read, and write with a backup.
//!
//! The file is the user's: unknown keys, other sections and comments are
//! theirs and stay untouched. A write only touches the four keys this
//! window owns inside `[wsl2]`, after copying the original to
//! `.wslconfig.bak`. Changes apply at the next full WSL restart — never
//! from this window; stopping WSL ends sessions, and that action lives in
//! the Give back flow where it belongs.

use picode_shell::{edit_wsl2, parse_wsl2, validate};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WslConfig {
    pub path: String,
    pub exists: bool,
    pub memory: Option<String>,
    pub processors: Option<String>,
    pub swap: Option<String>,
    #[serde(rename = "sparseVhd")]
    pub sparse_vhd: Option<String>,
    /// The whole file, so the window can show what a save will touch.
    pub raw: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WslConfigSaved {
    pub path: String,
    pub backup: String,
    pub note: String,
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
    let parsed = parse_wsl2(&raw);
    Ok(WslConfig {
        path: path.to_string_lossy().into_owned(),
        exists,
        memory: parsed.memory,
        processors: parsed.processors,
        swap: parsed.swap,
        sparse_vhd: parsed.sparse_vhd,
        raw,
    })
}

/// Saves the four owned keys. The backup is written before the file, so a
/// crash mid-write leaves the original recoverable either way.
#[tauri::command(async)]
pub fn wslconfig_write(
    memory: String,
    processors: String,
    swap: String,
    sparse_vhd: bool,
) -> Result<WslConfigSaved, String> {
    let memory = memory.trim().to_string();
    let processors = processors.trim().to_string();
    let swap = swap.trim().to_string();

    if let Err(msg) = validate(&memory, &processors, &swap) {
        return Err(msg);
    }

    let path = config_path();
    let raw = std::fs::read_to_string(&path).unwrap_or_default();
    let backup = path.with_extension("wslconfig.bak");
    if path.exists() {
        std::fs::copy(&path, &backup).map_err(|e| format!("backup failed: {}", e))?;
    }

    // An empty field means "back to WSL's default" — the key goes away.
    let edits = [
        (
            "memory",
            if memory.is_empty() {
                None
            } else {
                Some(memory.as_str())
            },
        ),
        (
            "processors",
            if processors.is_empty() {
                None
            } else {
                Some(processors.as_str())
            },
        ),
        (
            "swap",
            if swap.is_empty() {
                None
            } else {
                Some(swap.as_str())
            },
        ),
        ("sparseVhd", Some(if sparse_vhd { "true" } else { "false" })),
    ];
    let updated = edit_wsl2(&raw, &edits);
    std::fs::write(&path, &updated).map_err(|e| format!("write failed: {}", e))?;

    Ok(WslConfigSaved {
        path: path.to_string_lossy().into_owned(),
        backup: backup.to_string_lossy().into_owned(),
        note: "Saved. It applies at the next full WSL restart — stopping the distro from the Give back flow is one.".into(),
    })
}
