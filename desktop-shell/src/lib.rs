//! The pure half of the .wslconfig editor — no tauri, so the tests run on
//! any host. The commands in the bin's wslconfig.rs wrap these.

/// Pulls the owned keys out of the `[wsl2]` section. Case-insensitive on the
/// section and key names, which is what Windows' INI reader accepts.
#[derive(Default)]
pub struct Parsed {
    pub memory: Option<String>,
    pub processors: Option<String>,
    pub swap: Option<String>,
    pub sparse_vhd: Option<String>,
}

pub fn parse_wsl2(raw: &str) -> Parsed {
    let mut out = Parsed::default();
    let names = ["memory", "processors", "swap", "sparsevhd"];
    let mut in_wsl2 = false;
    for line in raw.lines() {
        let t = line.trim();
        if t.starts_with('[') && t.ends_with(']') {
            in_wsl2 = t.eq_ignore_ascii_case("[wsl2]");
            continue;
        }
        if !in_wsl2 {
            continue;
        }
        if let Some((k, v)) = t.split_once('=') {
            let kl = k.trim().to_ascii_lowercase();
            match kl.as_str() {
                "memory" => out.memory = Some(v.trim().to_string()),
                "processors" => out.processors = Some(v.trim().to_string()),
                "swap" => out.swap = Some(v.trim().to_string()),
                "sparsevhd" => out.sparse_vhd = Some(v.trim().to_string()),
                _ => {}
            }
        }
    }
    out
}

fn split_kv(line: &str) -> Option<(&str, &str)> {
    let (k, v) = line.split_once('=')?;
    Some((k.trim(), v.trim()))
}

/// Rewrites the file with the four owned keys set: replaces them in place,
/// drops the ones set to `None`, inserts the missing ones at the end of
/// `[wsl2]` (creating the section if the file has none), and leaves every
/// other line of the user's file alone.
pub fn edit_wsl2(raw: &str, edits: &[(&str, Option<&str>)]) -> String {
    let owned = |key: &str| edits.iter().position(|(k, _)| k.eq_ignore_ascii_case(key));
    let wanted: Vec<Option<&str>> = edits.iter().map(|(_, v)| *v).collect();

    let mut out: Vec<String> = Vec::new();
    let mut in_wsl2 = false;
    let mut seen_wsl2 = false;
    let mut pending: Vec<usize> = (0..edits.len()).filter(|&i| wanted[i].is_some()).collect();

    for line in raw.lines() {
        let t = line.trim();
        let is_section = t.starts_with('[') && t.ends_with(']');
        if is_section {
            if in_wsl2 {
                // Leaving [wsl2]: insert anything the section still lacks
                // here, so the keys land in the right section, never in the
                // next one.
                for i in &pending {
                    out.push(format!("{}={}", edits[*i].0, wanted[*i].unwrap()));
                }
                pending.clear();
            }
            in_wsl2 = t.eq_ignore_ascii_case("[wsl2]");
            if in_wsl2 {
                seen_wsl2 = true;
            }
        }

        if in_wsl2 {
            if let Some((k, _)) = split_kv(t) {
                if let Some(slot) = owned(k) {
                    match wanted[slot] {
                        Some(v) => {
                            out.push(format!("{}={}", edits[slot].0, v));
                            pending.retain(|&i| i != slot);
                        }
                        None => continue, // unset: drop the line
                    }
                    continue;
                }
            }
        }
        out.push(line.to_string());
    }
    for i in &pending {
        if !seen_wsl2 {
            if !out.is_empty() {
                out.push(String::new());
            }
            out.push("[wsl2]".into());
            seen_wsl2 = true;
        }
        out.push(format!("{}={}", edits[*i].0, wanted[*i].unwrap()));
    }

    let mut result = out.join("\n");
    if raw.ends_with('\n') || (raw.is_empty() && !result.is_empty()) {
        result.push('\n');
    }
    result
}

/// WSL accepts `8GB`, `512MB`, `auto`. Anything else is refused here so a
/// typo never lands in the user's file behind a save button.
fn valid_size(v: &str) -> bool {
    let u = v.trim().to_ascii_uppercase();
    if u == "AUTO" {
        return true;
    }
    for suffix in ["KB", "MB", "GB", "TB"] {
        if let Some(num) = u.strip_suffix(suffix) {
            return !num.is_empty()
                && num.chars().all(|c| c.is_ascii_digit() || c == '.')
                && num.chars().any(|c| c.is_ascii_digit());
        }
    }
    false
}

pub fn validate(memory: &str, processors: &str, swap: &str) -> Result<(), String> {
    if !memory.is_empty() && !valid_size(memory) {
        return Err(format!("memory: use a size like 8GB (got {:?})", memory));
    }
    if !swap.is_empty() && !valid_size(swap) {
        return Err(format!(
            "swap: use a size like 2GB, or auto (got {:?})",
            swap
        ));
    }
    if !processors.is_empty() && processors.parse::<u32>().map(|p| p == 0).unwrap_or(true) {
        return Err(format!(
            "processors: a whole number above 0 (got {:?})",
            processors
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_reads_owned_keys_and_ignores_the_rest() {
        let raw = "# my machine\n[wsl2]\nmemory=8GB\nswap=0\nautoMemoryReclaim=gradual\n[experimental]\nmemory=1KB\n";
        let p = parse_wsl2(raw);
        let (memory, processors, swap, sparse) = (p.memory, p.processors, p.swap, p.sparse_vhd);
        assert_eq!(memory.as_deref(), Some("8GB"));
        assert_eq!(swap.as_deref(), Some("0"));
        assert_eq!(processors, None);
        assert_eq!(sparse, None);
    }

    #[test]
    fn edit_replaces_in_place_and_keeps_unknown_lines() {
        let raw = "[wsl2]\nmemory=8GB\nautoMemoryReclaim=gradual\nsparseVhd=false\n";
        let out = edit_wsl2(
            raw,
            &[
                ("memory", Some("16GB")),
                ("processors", Some("4")),
                ("swap", None),
                ("sparseVhd", Some("true")),
            ],
        );
        assert!(out.contains("memory=16GB"));
        assert!(out.contains("processors=4"));
        assert!(out.contains("sparseVhd=true"));
        assert!(out.contains("autoMemoryReclaim=gradual"));
        assert!(!out.contains("swap="));
    }

    #[test]
    fn edit_creates_the_section_when_missing() {
        let out = edit_wsl2("", &[("memory", Some("8GB")), ("sparseVhd", Some("true"))]);
        assert_eq!(out, "[wsl2]\nmemory=8GB\nsparseVhd=true\n");
    }

    #[test]
    fn edit_appends_inside_an_existing_section() {
        let raw = "[wsl2]\nswap=2GB\n\n[experimental]\nx=1\n";
        let out = edit_wsl2(raw, &[("memory", Some("4GB"))]);
        let mem_at = out.find("memory=4GB").unwrap();
        let exp_at = out.find("[experimental]").unwrap();
        assert!(
            mem_at < exp_at,
            "the key belongs to [wsl2], not to a later section"
        );
    }

    #[test]
    fn unset_keys_are_removed_not_emptied() {
        let raw = "[wsl2]\nmemory=8GB\nswap=0\n";
        let out = edit_wsl2(raw, &[("memory", None), ("swap", Some("2GB"))]);
        assert!(!out.contains("memory"));
        assert!(out.contains("swap=2GB"));
    }

    #[test]
    fn validate_accepts_sizes_and_refuses_junk() {
        assert!(validate("8GB", "4", "2GB").is_ok());
        assert!(validate("", "", "").is_ok());
        assert!(validate("auto", "4", "auto").is_ok());
        assert!(validate("eight gb", "4", "").is_err());
        assert!(validate("GB", "4", "").is_err());
        assert!(validate("8GB", "0", "").is_err());
        assert!(validate("8GB", "x", "").is_err());
    }
}
