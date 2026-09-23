//! The distro hold, read side (the Go tool writes it:
//! internal/desktop/hold.go). While a move, a backup or a WSL update needs
//! the distro stopped, the tool keeps `%LOCALAPPDATA%\PiCode\distro-hold.json`
//! fresh; while it is, the shell neither re-arms its keepalive nor runs its
//! server discovery — both start the distro with `wsl.exe`. A file rather
//! than a flag here, so a flow started from a terminal holds too; a
//! heartbeat rather than a pid, so a killed flow's hold expires by itself.

use std::time::{Duration, SystemTime};

/// Must match HoldFresh in internal/desktop/hold.go.
const FRESH: Duration = Duration::from_secs(120);

fn path() -> Option<std::path::PathBuf> {
    let base = std::env::var_os("LOCALAPPDATA")?;
    Some(std::path::Path::new(&base).join("PiCode").join("distro-hold.json"))
}

/// True while a flow holds the distro down.
pub fn distro_held() -> bool {
    let Some(p) = path() else { return false };
    let Ok(meta) = std::fs::metadata(&p) else { return false };
    let Ok(modified) = meta.modified() else { return false };
    SystemTime::now()
        .duration_since(modified)
        .map(|age| age < FRESH)
        .unwrap_or(true) // a clock skewed into the future still holds
}
