//! The pure half of the .wslconfig editor — no tauri, so the tests run on
//! any host. The key table mirrors Microsoft's documented `.wslconfig`
//! settings (learn.microsoft.com windows/wsl/wsl-config): every entry is a
//! (section, key, kind) triple. Anything the table does not own passes
//! through untouched — the file is the user's.
//!
//! Value kinds:
//! - `Size`  — "8GB", "512MB", "0", or "auto" where WSL allows it
//! - `Num`   — a whole number
//! - `Bool`  — "true" / "false"
//! - `Text`  — any single-line string
//! - `Enum`  — one of a fixed list of values

pub mod axfmt;
pub mod b64;
pub mod cdppolicy;
pub mod geometry;
pub mod layer_geometry;
pub mod keys;
pub mod origins;
pub mod permissions;

use std::collections::BTreeMap;

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum Kind {
    Size,
    Num,
    Bool,
    Text,
    Enum(&'static [&'static str]),
}

/// (section, key, kind) for every setting the window owns. A key the user
/// set in another section than we would write it is still found and edited
/// in place — only truly absent keys land in their documented section.
pub const OWNED: &[(&str, &str, Kind)] = &[
    // ---- [wsl2] — the virtual machine ------------------------------------
    ("wsl2", "memory", Kind::Size),
    ("wsl2", "processors", Kind::Num),
    ("wsl2", "swap", Kind::Size),
    ("wsl2", "swapFile", Kind::Text),
    ("wsl2", "kernel", Kind::Text),
    ("wsl2", "kernelModules", Kind::Text),
    ("wsl2", "kernelCommandLine", Kind::Text),
    ("wsl2", "localhostForwarding", Kind::Bool),
    ("wsl2", "guiApplications", Kind::Bool),
    ("wsl2", "debugConsole", Kind::Bool),
    ("wsl2", "nestedVirtualization", Kind::Bool),
    ("wsl2", "safeMode", Kind::Bool),
    ("wsl2", "firewall", Kind::Bool),
    ("wsl2", "dnsTunneling", Kind::Bool),
    ("wsl2", "dnsProxy", Kind::Bool),
    ("wsl2", "autoProxy", Kind::Bool),
    ("wsl2", "pageReporting", Kind::Bool),
    ("wsl2", "maxCrashDumpCount", Kind::Num),
    ("wsl2", "vmIdleTimeout", Kind::Num),
    (
        "wsl2",
        "networkingMode",
        Kind::Enum(&["none", "nat", "mirrored", "virtioproxy"]),
    ),
    ("wsl2", "defaultVhdSize", Kind::Size),
    // ---- [experimental] ---------------------------------------------------
    (
        "experimental",
        "autoMemoryReclaim",
        Kind::Enum(&["disabled", "gradual", "dropCache"]),
    ),
    ("experimental", "sparseVhd", Kind::Bool),
    ("experimental", "hostAddressLoopback", Kind::Bool),
    ("experimental", "bestEffortDnsParsing", Kind::Bool),
    ("experimental", "dnsTunnelingIpAddress", Kind::Text),
    ("experimental", "initialAutoProxyTimeout", Kind::Num),
    ("experimental", "ignoredPorts", Kind::Text),
];

pub fn lookup(key: &str) -> Option<Kind> {
    // Keys are unique across the table, and a key the user parked under a
    // different section than the docs say is still theirs to edit in place —
    // so ownership matches on the key name alone.
    OWNED
        .iter()
        .find(|(_, k, _)| k.eq_ignore_ascii_case(key))
        .map(|(_, _, kind)| *kind)
}

/// Every owned key with the value the file currently gives it, if any. A key
/// found under an unusual section still reports with that section, so the
/// editor edits what is there instead of piling a second copy.
pub fn parse_owned(raw: &str) -> BTreeMap<(String, String), String> {
    let mut out = BTreeMap::new();
    let mut section = String::new();
    for line in raw.lines() {
        let t = line.trim();
        if t.starts_with('[') && t.ends_with(']') {
            section = t[1..t.len() - 1].trim().to_string();
            continue;
        }
        if section.is_empty() {
            continue;
        }
        if let Some((k, v)) = t.split_once('=') {
            let k = k.trim();
            if lookup(k).is_some() {
                out.insert((section.clone(), k.to_string()), v.trim().to_string());
            }
        }
    }
    out
}

fn split_kv(line: &str) -> Option<(&str, &str)> {
    let (k, v) = line.split_once('=')?;
    Some((k.trim(), v.trim()))
}

/// Rewrites the file with the given edits applied: replaces owned keys in
/// place, drops the ones set to `None`, inserts the missing ones at the end
/// of their section (creating the section if the file has none), and leaves
/// every other line of the user's file alone.
pub fn edit(raw: &str, edits: &[(&str, &str, Option<&str>)]) -> String {
    let mut out: Vec<String> = Vec::new();
    let mut section = String::new();
    let mut seen: BTreeMap<(String, String), ()> = BTreeMap::new();
    let mut ends: Vec<(String, String, String)> = Vec::new(); // section, key, value to append

    for line in raw.lines() {
        let t = line.trim();
        let is_section = t.starts_with('[') && t.ends_with(']');
        if is_section {
            section = t[1..t.len() - 1].trim().to_string();
        }
        if in_owned_section(&section) {
            if let Some((k, _)) = split_kv(t) {
                if let Some(edit) = edits.iter().find(|(_, ek, _)| ek.eq_ignore_ascii_case(k)) {
                    seen.insert((edit.0.to_string(), edit.1.to_string()), ());
                    match edit.2 {
                        Some(v) => {
                            out.push(format!("{}={}", edit.1, v));
                        }
                        None => continue, // unset: drop the line
                    }
                    continue;
                }
            }
        }
        out.push(line.to_string());
    }
    for (s, k, v) in edits {
        if seen.contains_key(&(s.to_string(), k.to_string())) || k.is_empty() {
            continue;
        }
        if v.is_some() {
            ends.push((s.to_string(), k.to_string(), v.unwrap().to_string()));
        }
    }
    // Append the keys their sections never had, right after each section's
    // last line; sections that do not exist are created at the end.
    let mut result_lines: Vec<String> = out;
    let mut done_sections: Vec<String> = Vec::new();
    for (s, _, _) in &ends {
        if done_sections.iter().any(|x| x == s) {
            continue;
        }
        done_sections.push(s.clone());
        let keys: Vec<&(String, String, String)> =
            ends.iter().filter(|(es, _, _)| es == s).collect();
        match result_lines.iter().position(|l| {
            l.trim().starts_with('[') && l.trim()[1..l.trim().len() - 1].trim() == s.as_str()
        }) {
            Some(start) => {
                let mut at = start + 1;
                for (i, l) in result_lines.iter().enumerate().skip(start + 1) {
                    let t = l.trim();
                    if t.starts_with('[') && t.ends_with(']') {
                        break;
                    }
                    if !t.is_empty() {
                        at = i + 1;
                    }
                }
                for (i, (_, ek, ev)) in keys.iter().enumerate() {
                    result_lines.insert(at + i, format!("{}={}", ek, ev));
                }
            }
            None => {
                if !result_lines.is_empty()
                    && result_lines.last().map(|l| !l.trim().is_empty()) == Some(true)
                {
                    result_lines.push(String::new());
                }
                result_lines.push(format!("[{}]", s));
                for (_, ek, ev) in &keys {
                    result_lines.push(format!("{}={}", ek, ev));
                }
            }
        }
    }

    let mut result = result_lines.join("\n");
    if raw.ends_with('\n') || (raw.is_empty() && !result.is_empty()) {
        result.push('\n');
    }
    result
}

fn in_owned_section(section: &str) -> bool {
    OWNED
        .iter()
        .any(|(s, _, _)| s.eq_ignore_ascii_case(section))
}

/// WSL sizes are a number plus an optional unit; "0" and bare numbers are
/// legal, and swap also allows "auto".
fn valid_size(v: &str) -> bool {
    let u = v.trim().to_ascii_uppercase();
    if u == "AUTO" {
        return true;
    }
    for suffix in ["KB", "MB", "GB", "TB", "B"] {
        if let Some(num) = u.strip_suffix(suffix) {
            return !num.is_empty()
                && num.chars().all(|c| c.is_ascii_digit() || c == '.')
                && num.chars().any(|c| c.is_ascii_digit());
        }
    }
    !u.is_empty() && u.chars().all(|c| c.is_ascii_digit())
}

/// Validates one value against its key's kind. Empty means "back to WSL's
/// default" — the key is removed, not set empty.
pub fn validate_value(section: &str, key: &str, value: &str) -> Result<(), String> {
    let _ = section; // the section names where new keys are written; lookup is by key
    let kind = lookup(key).ok_or_else(|| format!("unknown setting {}\\{}", section, key))?;
    let v = value.trim();
    let bad = || format!("{}: {:?} is not a valid value", key, value);
    match kind {
        Kind::Text => Ok(()),
        Kind::Num => {
            if !v.is_empty() && v.chars().all(|c| c.is_ascii_digit()) {
                Ok(())
            } else {
                Err(bad())
            }
        }
        Kind::Size => {
            if v.is_empty() || valid_size(v) {
                Ok(())
            } else {
                Err(bad())
            }
        }
        Kind::Bool => {
            if v.is_empty() || v == "true" || v == "false" {
                Ok(())
            } else {
                Err(bad())
            }
        }
        Kind::Enum(values) => {
            if v.is_empty() || values.contains(&v) {
                Ok(())
            } else {
                Err(format!("{}: use one of {}", key, values.join(", ")))
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_reads_owned_keys_across_sections() {
        let raw = "[wsl2]\nmemory=8GB\nautoMemoryReclaim=gradual\n[experimental]\nsparseVhd=true\n";
        let m = parse_owned(raw);
        assert_eq!(
            m.get(&("wsl2".into(), "memory".into())).map(String::as_str),
            Some("8GB")
        );
        // sparseVhd is owned in [experimental]; a copy under [wsl2] is still found.
        let m2 = parse_owned("[wsl2]\nsparseVhd=true\n");
        assert_eq!(
            m2.get(&("wsl2".into(), "sparseVhd".into()))
                .map(String::as_str),
            Some("true")
        );
        // A key the docs put under [experimental] but the user parked under
        // [wsl2] is still owned — reported where it actually lives, so the
        // editor edits it in place instead of writing a second copy.
        assert_eq!(
            m.get(&("wsl2".into(), "autoMemoryReclaim".into()))
                .map(String::as_str),
            Some("gradual")
        );
    }

    #[test]
    fn edit_replaces_in_place_and_keeps_unknown_lines() {
        let raw = "[wsl2]\nmemory=8GB\nautoMemoryReclaim=gradual\nswap=0\n";
        let out = edit(
            raw,
            &[
                ("wsl2", "memory", Some("16GB")),
                ("wsl2", "processors", Some("4")),
                ("wsl2", "swap", None),
            ],
        );
        assert!(out.contains("memory=16GB"));
        assert!(out.contains("processors=4"));
        assert!(out.contains("autoMemoryReclaim=gradual"));
        assert!(!out.contains("swap="));
    }

    #[test]
    fn edit_creates_missing_sections() {
        let out = edit(
            "",
            &[
                ("wsl2", "memory", Some("8GB")),
                ("experimental", "sparseVhd", Some("true")),
            ],
        );
        assert!(out.contains("[wsl2]\nmemory=8GB"));
        assert!(out.contains("[experimental]\nsparseVhd=true"));
    }

    #[test]
    fn edit_writes_keys_into_their_own_sections() {
        let raw = "[wsl2]\nmemory=4GB\n[experimental]\nautoMemoryReclaim=gradual\n";
        let out = edit(
            raw,
            &[
                ("wsl2", "processors", Some("2")),
                ("experimental", "sparseVhd", Some("true")),
            ],
        );
        let proc_at = out.find("processors=2").unwrap();
        let exp_at = out.find("[experimental]").unwrap();
        assert!(proc_at < exp_at, "processors belongs to [wsl2]");
        let sparse_at = out.find("sparseVhd=true").unwrap();
        assert!(sparse_at > exp_at, "sparseVhd lands in [experimental]");
    }

    #[test]
    fn unset_keys_are_removed_not_emptied() {
        let raw = "[wsl2]\nmemory=8GB\nswap=0\n";
        let out = edit(
            raw,
            &[("wsl2", "memory", None), ("wsl2", "swap", Some("2GB"))],
        );
        assert!(!out.contains("memory"));
        assert!(out.contains("swap=2GB"));
    }

    #[test]
    fn validate_by_kind() {
        assert!(validate_value("wsl2", "memory", "8GB").is_ok());
        assert!(validate_value("wsl2", "memory", "").is_ok());
        assert!(validate_value("wsl2", "memory", "eight").is_err());
        assert!(validate_value("wsl2", "swap", "auto").is_ok());
        assert!(validate_value("wsl2", "processors", "0").is_ok());
        assert!(validate_value("wsl2", "processors", "x").is_err());
        assert!(validate_value("wsl2", "localhostForwarding", "true").is_ok());
        assert!(validate_value("wsl2", "localhostForwarding", "maybe").is_err());
        assert!(validate_value("wsl2", "networkingMode", "mirrored").is_ok());
        assert!(validate_value("wsl2", "networkingMode", "bridged").is_err());
        assert!(validate_value("experimental", "autoMemoryReclaim", "dropCache").is_ok());
        assert!(validate_value("wsl2", "not-a-key", "1").is_err());
    }
}
