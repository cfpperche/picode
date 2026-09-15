// The tray's disk line (ADR-0142 slice 2): the pure half of what the retired
// Go tray composed in disk.go — byte formatting, the one-line summary and
// the Give-back item's label. Wording is tested here instead of discovered
// in a tooltip nobody can screenshot from CI.

use serde::Deserialize;

/// A volume this close to full starts refusing writes.
const LOW_FREE: i64 = 20 << 30; // 20 GiB
/// A file holding this much unused space is worth one conversion. Smaller
/// than a Go build cache on purpose: the point is to name the day the
/// distro started costing real space, not to nag about the first gigabyte.
const HELD_WORTH_TELLING: i64 = 8 << 30; // 8 GiB

/// Formats a byte count the way the Go side does (`hostfs.Bytes`): whole
/// units, one decimal below ten.
pub fn bytes(n: i64) -> String {
    const UNIT: f64 = 1024.0;
    if n < 1024 {
        return format!("{n} B");
    }
    let units = ["KB", "MB", "GB", "TB", "PB"];
    let mut v = n as f64 / UNIT;
    let mut i = 0;
    while v >= UNIT && i < units.len() - 1 {
        v /= UNIT;
        i += 1;
    }
    if v < 10.0 {
        format!("{v:.1} {}", units[i])
    } else {
        format!("{v:.0} {}", units[i])
    }
}

/// The Windows half of `picode-desktop disk --json`, plus the precomputed
/// held number. The distro half is not needed for the line: `held` already
/// folds it in, and a missing distro half arrives as zero, which reads as
/// "no held part" rather than as an error.
#[derive(Debug, Default, Clone, Deserialize)]
pub struct Facts {
    #[serde(rename = "allocatedBytes", default)]
    pub allocated: i64,
    #[serde(rename = "freeBytes", default)]
    pub free: i64,
    #[serde(default)]
    pub held: i64,
    #[serde(default)]
    pub wsl: String,
    #[serde(rename = "canSparse", default)]
    pub can_sparse: bool,
}

#[derive(Deserialize)]
struct ReportJson {
    #[serde(default)]
    windows: Option<Facts>,
    // Held is top-level: it folds the distro half in, and a missing distro
    // half arrives as zero — "no held part", not an error.
    #[serde(default)]
    held: i64,
}

/// Parses one `disk --json` report. A missing Windows half is the "not
/// read" case, never a zero: the tray is the last place that should tell
/// someone their disk is empty.
pub fn parse_report(text: &str) -> Result<Facts, String> {
    let start = text
        .find('{')
        .ok_or_else(|| "disk report is not JSON".to_string())?;
    let report: ReportJson = serde_json::from_str(text[start..].trim_end())
        .map_err(|e| format!("disk report JSON: {e}"))?;
    let mut facts = report.windows.ok_or_else(|| "disk was not read".to_string())?;
    facts.held = report.held;
    Ok(facts)
}

/// The tray's one line about the disk, and whether the wording warns.
pub fn line(f: &Facts) -> (String, bool) {
    let mut parts = vec![format!("WSL {}", bytes(f.allocated))];
    if f.held >= HELD_WORTH_TELLING {
        parts.push(format!("≈{} held by Windows", bytes(f.held)));
    }
    let mut free = format!("C: {} free", bytes(f.free));
    let warn = f.free < LOW_FREE;
    if warn {
        free.push_str(" — low");
    }
    parts.push(free);
    (parts.join(" · "), warn)
}

/// The Give-back item's label and whether it acts. Disabled on purpose when
/// a compact is running, when the disk was not read, when this WSL build
/// cannot convert the file, or when nothing is held — a grey line that says
/// why beats an error after the click.
pub fn compact_label(facts: Option<&Facts>, compacting: bool) -> (String, bool) {
    if compacting {
        return ("Compacting…".to_string(), false);
    }
    let Some(f) = facts else {
        return ("Disk not read — nothing to offer".to_string(), false);
    };
    if !f.can_sparse {
        return (
            format!("Compact from an admin terminal — WSL {} cannot convert", f.wsl),
            false,
        );
    }
    if f.held < HELD_WORTH_TELLING {
        return ("No held space to give back".to_string(), false);
    }
    (format!("Give back ≈{}…", bytes(f.held)), true)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn byte_steps_match_the_go_side() {
        assert_eq!(bytes(0), "0 B");
        assert_eq!(bytes(1023), "1023 B");
        assert_eq!(bytes(1024), "1.0 KB");
        assert_eq!(bytes(5 << 30), "5.0 GB");
        assert_eq!(bytes(27 << 30), "27 GB");
        assert_eq!(bytes(218 << 30), "218 GB");
        assert_eq!(bytes((1 << 40) * 3 / 2), "1.5 TB");
    }

    fn facts() -> Facts {
        Facts {
            allocated: 218 << 30,
            free: 27 << 30,
            held: 92 << 30,
            wsl: "2.7.0.0".to_string(),
            can_sparse: true,
        }
    }

    #[test]
    fn full_line_names_allocated_held_and_free() {
        let (title, warn) = line(&facts());
        assert_eq!(
            title,
            "WSL 218 GB · ≈92 GB held by Windows · C: 27 GB free"
        );
        assert!(!warn);
    }

    #[test]
    fn low_free_warns_in_words() {
        let mut f = facts();
        f.free = 5 << 30;
        let (title, warn) = line(&f);
        assert!(title.ends_with("C: 5.0 GB free — low"), "{title}");
        assert!(warn);
    }

    #[test]
    fn small_held_has_no_part() {
        let mut f = facts();
        f.held = 1 << 30;
        let (title, _) = line(&f);
        assert_eq!(title, "WSL 218 GB · C: 27 GB free");
    }

    #[test]
    fn report_parses_and_missing_windows_is_not_read() {
        let f = parse_report(
            r#"{"windows":{"allocatedBytes":1,"freeBytes":2,"wsl":"2.7.0.0","canSparse":true},"held":3}"#,
        )
        .expect("report");
        assert_eq!((f.allocated, f.free, f.held), (1, 2, 3));
        assert!(f.can_sparse);
        assert!(parse_report(r#"{"windowsError":"boom"}"#).is_err());
        assert!(parse_report("not json").is_err());
    }

    #[test]
    fn compact_item_states_its_reason() {
        let f = facts();
        assert_eq!(
            compact_label(Some(&f), false),
            ("Give back ≈92 GB…".to_string(), true)
        );
        assert_eq!(
            compact_label(Some(&f), true),
            ("Compacting…".to_string(), false)
        );
        assert_eq!(
            compact_label(None, false).0,
            "Disk not read — nothing to offer"
        );
        let mut old = facts();
        old.can_sparse = false;
        old.wsl = "1.2.5.0".to_string();
        let (label, on) = compact_label(Some(&old), false);
        assert!(!on && label.contains("1.2.5.0"), "{label}");
        let mut none = facts();
        none.held = 0;
        assert_eq!(
            compact_label(Some(&none), false),
            ("No held space to give back".to_string(), false)
        );
    }
}
