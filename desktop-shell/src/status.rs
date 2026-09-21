// The tray's at-a-glance contract (ADR-0142): the status line and the tooltip
// the retired Go tray composed in tray_windows.go. One writer owns both, so
// the health timer can never erase what another timer wrote.

/// The disabled status line at the top of the tray menu.
pub fn title(up: bool, detail: &str) -> String {
    if up {
        format!("Running · {detail}")
    } else {
        format!("Stopped · {detail}")
    }
}

/// The tooltip: the status detail, nothing else. The disk line lived here
/// until the tray's disk half moved into the Management window.
pub fn tooltip(detail: &str) -> String {
    format!("PiCode — {detail}")
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
    fn tooltip_is_status_only() {
        assert_eq!(
            tooltip("https://localhost:8445"),
            "PiCode — https://localhost:8445"
        );
    }
}
