// The tray's at-a-glance contract (ADR-0142): the status line and the tooltip
// the retired Go tray composed in tray_windows.go. One writer owns both, so
// the health timer can never erase what the disk timer wrote — and, until
// slice 2 wires the disk half, the disk line is simply absent.

/// The disabled status line at the top of the tray menu.
pub fn title(up: bool, detail: &str) -> String {
    if up {
        format!("Running · {detail}")
    } else {
        format!("Stopped · {detail}")
    }
}

/// The tooltip: status plus the disk line when slice 2 provides it, plus the
/// two-sided-report pointer while the disk warns. The tray has no balloon
/// left to raise, so the tooltip is the alert surface.
pub fn tooltip(detail: &str, disk: Option<&str>, warn: bool) -> String {
    let mut s = format!("PiCode — {detail}");
    if let Some(line) = disk {
        if !line.is_empty() {
            s.push_str(" · ");
            s.push_str(line);
        }
    }
    if warn {
        s.push_str(" — run picode-desktop disk for the two-sided report");
    }
    s
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn titles_mirror_the_tray_contract() {
        assert_eq!(title(true, "https://localhost:8445"), "Running · https://localhost:8445");
        assert_eq!(title(false, "not answering"), "Stopped · not answering");
    }

    #[test]
    fn tooltip_without_disk_is_status_only() {
        assert_eq!(
            tooltip("https://localhost:8445", None, false),
            "PiCode — https://localhost:8445"
        );
    }

    #[test]
    fn tooltip_carries_the_disk_line_when_present() {
        assert_eq!(
            tooltip(
                "https://localhost:8445",
                Some("WSL 218 GB · C: 27 GB free"),
                false
            ),
            "PiCode — https://localhost:8445 · WSL 218 GB · C: 27 GB free"
        );
    }

    #[test]
    fn warning_appends_the_report_pointer() {
        let t = tooltip("https://localhost:8445", None, true);
        assert!(t.ends_with(" — run picode-desktop disk for the two-sided report"), "{t}");
    }
}
