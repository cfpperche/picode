//! The snapshot's text: UI Automation nodes rendered as lines the model can
//! act on. One node per line, a ref it can name (`e12`), the role, the name,
//! the centre in the coordinates of the last image (or physical when no
//! image exists yet), and the flags that matter (disabled, offscreen,
//! password). Pure: `rustc --edition 2021 --test src/axfmt.rs`.

use crate::geometry::Frame;

/// One accessibility node as the walker collected it. `rect` is physical
/// (left, top, right, bottom).
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Node {
    pub depth: u16,
    pub control_type: i32,
    pub name: String,
    pub value: String,
    pub automation_id: String,
    pub rect: (i32, i32, i32, i32),
    pub enabled: bool,
    pub offscreen: bool,
    pub password: bool,
}

/// The rendered snapshot: lines, and how many nodes the caps dropped.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Rendered {
    pub lines: Vec<String>,
    pub dropped: usize,
}

/// The nodes kept in one snapshot. Enough for a busy window; a model that
/// needs more drills down with `ref`.
pub const MAX_NODES: usize = 400;

/// UIA control type ids (UIA_*ControlTypeId) → the short role the line shows.
pub fn role_name(control_type: i32) -> &'static str {
    match control_type {
        50000 => "Button",
        50001 => "Calendar",
        50002 => "CheckBox",
        50003 => "ComboBox",
        50004 => "Edit",
        50005 => "Hyperlink",
        50006 => "Image",
        50007 => "ListItem",
        50008 => "List",
        50009 => "Menu",
        50010 => "MenuBar",
        50011 => "MenuItem",
        50012 => "ProgressBar",
        50013 => "RadioButton",
        50014 => "ScrollBar",
        50015 => "Slider",
        50016 => "Spinner",
        50017 => "StatusBar",
        50018 => "Tab",
        50019 => "TabItem",
        50020 => "Text",
        50021 => "ToolBar",
        50022 => "ToolTip",
        50023 => "Tree",
        50024 => "TreeItem",
        50025 => "Custom",
        50026 => "Group",
        50027 => "Thumb",
        50028 => "DataGrid",
        50029 => "DataItem",
        50030 => "Document",
        50031 => "SplitButton",
        50032 => "Window",
        50033 => "Pane",
        50034 => "Header",
        50035 => "HeaderItem",
        50036 => "Table",
        50037 => "TitleBar",
        50038 => "Separator",
        50039 => "SemanticZoom",
        50040 => "AppBar",
        _ => "Element",
    }
}

fn quote(s: &str, max: usize) -> String {
    let mut out = String::with_capacity(s.len().min(max) + 2);
    out.push('"');
    let mut n = 0;
    for c in s.chars() {
        if n >= max {
            out.push('…');
            break;
        }
        match c {
            '"' => out.push_str("\\\""),
            '\n' | '\r' => out.push(' '),
            c => out.push(c),
        }
        n += 1;
    }
    out.push('"');
    out
}

/// Renders the nodes, `e1`… in walk order, mapping each centre into the
/// frame's image space when a frame is given. Nodes past `max` are counted
/// as dropped; a node with no name, no value and no id is skipped (it is
/// structure, not something to act on) but never counted as dropped.
pub fn render(nodes: &[Node], frame: Option<&Frame>, max: usize) -> Rendered {
    let mut lines = Vec::new();
    let mut dropped = 0;
    for (i, n) in nodes.iter().enumerate() {
        if n.name.is_empty() && n.value.is_empty() && n.automation_id.is_empty() {
            continue;
        }
        if lines.len() >= max {
            dropped += 1;
            continue;
        }
        let (l, t, r, b) = n.rect;
        let cx = l + (r - l) / 2;
        let cy = t + (b - t) / 2;
        let centre = match frame {
            Some(f) => match f.to_image(cx, cy) {
                Some((x, y)) => format!("@center({x},{y})"),
                None => "@offimage".to_string(),
            },
            None => format!("@screen({cx},{cy})"),
        };
        let mut line = format!("e{} {} {} {} depth={}", i + 1, role_name(n.control_type), quote(&n.name, 80), centre, n.depth);
        if !n.value.is_empty() {
            line.push_str(" value=");
            line.push_str(&quote(&n.value, 120));
        }
        if !n.automation_id.is_empty() {
            line.push_str(" id=");
            line.push_str(&n.automation_id);
        }
        if !n.enabled {
            line.push_str(" [disabled]");
        }
        if n.offscreen {
            line.push_str(" [offscreen]");
        }
        if n.password {
            line.push_str(" [password]");
        }
        lines.push(line);
    }
    Rendered { lines, dropped }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::geometry::frame_for;

    fn node(depth: u16, ct: i32, name: &str, rect: (i32, i32, i32, i32)) -> Node {
        Node { depth, control_type: ct, name: name.into(), rect, enabled: true, ..Default::default() }
    }

    #[test]
    fn lines_carry_ref_role_name_centre_and_flags() {
        let f = frame_for(0, 0, 1920, 1080, 1280, 1280); // scale 1.5
        let mut pw = node(2, 50004, "Password", (300, 300, 600, 330));
        pw.password = true;
        pw.enabled = false;
        let nodes = vec![
            node(0, 50032, "Untitled - Notepad", (0, 0, 1920, 1080)),
            node(1, 50033, "", (0, 0, 10, 10)), // nameless structure: skipped, not dropped
            node(2, 50000, "Save", (600, 200, 660, 230)),
            pw,
        ];
        let r = render(&nodes, Some(&f), MAX_NODES);
        assert_eq!(r.dropped, 0);
        assert_eq!(r.lines.len(), 3);
        assert_eq!(r.lines[0], "e1 Window \"Untitled - Notepad\" @center(640,360) depth=0");
        assert_eq!(r.lines[1], "e3 Button \"Save\" @center(420,143) depth=2");
        assert_eq!(r.lines[2], "e4 Edit \"Password\" @center(300,210) depth=2 [disabled] [password]");
    }

    #[test]
    fn the_cap_counts_what_it_dropped_and_no_frame_means_screen_coordinates() {
        let nodes: Vec<Node> = (0..5).map(|i| node(1, 50020, &format!("t{i}"), (i * 10, 0, i * 10 + 10, 10))).collect();
        let r = render(&nodes, None, 3);
        assert_eq!(r.lines.len(), 3);
        assert_eq!(r.dropped, 2);
        assert!(r.lines[0].contains("@screen(5,5)"));
        let mut v = node(1, 50004, "Text Editor", (0, 0, 100, 100));
        v.value = "hello \"world\"\nline".into();
        v.automation_id = "RichEditD2DPT".into();
        let r = render(&[v], None, 10);
        assert_eq!(r.lines[0], "e1 Edit \"Text Editor\" @screen(50,50) depth=1 value=\"hello \\\"world\\\" line\" id=RichEditD2DPT");
    }

    #[test]
    fn roles_are_named_and_unknown_ids_stay_generic() {
        assert_eq!(role_name(50004), "Edit");
        assert_eq!(role_name(1), "Element");
    }
}
