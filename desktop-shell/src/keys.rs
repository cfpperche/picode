//! Key names for the computer tool's `key` and `hold_key` actions. The
//! vocabulary is xdotool's, the one Anthropic's contract and its reference
//! implementation speak ("ctrl+s", "Return", "Page_Down", "super"), mapped to
//! Windows virtual-key codes. Pure: `rustc --edition 2021 --test src/keys.rs`.

/// One key of a chord: a virtual-key code the table knows, or a character
/// the caller resolves at runtime with the keyboard layout (VkKeyScanW).
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Key {
    Vk(u16),
    Char(char),
}

/// A chord: modifiers held, then the key pressed. "ctrl+shift+t".
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Chord {
    pub mods: Vec<u16>,
    pub key: Key,
}

pub const VK_SHIFT: u16 = 0x10;
pub const VK_CONTROL: u16 = 0x11;
pub const VK_MENU: u16 = 0x12; // alt
pub const VK_LWIN: u16 = 0x5B;
pub const VK_RMENU: u16 = 0xA5;
pub const VK_RETURN: u16 = 0x0D;

/// The modifier a chord part names, if it is one.
pub fn modifier(part: &str) -> Option<u16> {
    Some(match part {
        "ctrl" | "control" | "ctl" | "lctrl" | "rctrl" => VK_CONTROL,
        "shift" | "lshift" | "rshift" => VK_SHIFT,
        "alt" | "option" | "lalt" | "meta_alt" => VK_MENU,
        "altgr" | "ralt" => VK_RMENU,
        "super" | "win" | "windows" | "cmd" | "command" | "meta" | "lsuper" | "rsuper" => VK_LWIN,
        _ => return None,
    })
}

/// A named key (lowercase) → virtual-key code, or a character for the
/// layout to resolve. None when the name means nothing.
pub fn key(name: &str) -> Option<Key> {
    let vk = |v: u16| Some(Key::Vk(v));
    let ch = |c: char| Some(Key::Char(c));
    match name {
        "return" | "enter" => vk(VK_RETURN),
        "kp_enter" => vk(VK_RETURN),
        "tab" => vk(0x09),
        "escape" | "esc" => vk(0x1B),
        "backspace" => vk(0x08),
        "delete" | "del" => vk(0x2E),
        "insert" | "ins" => vk(0x2D),
        "home" => vk(0x24),
        "end" => vk(0x23),
        "page_up" | "pageup" | "prior" | "pgup" => vk(0x21),
        "page_down" | "pagedown" | "next" | "pgdn" => vk(0x22),
        "up" => vk(0x26),
        "down" => vk(0x28),
        "left" => vk(0x25),
        "right" => vk(0x27),
        "space" => vk(0x20),
        "print" | "printscreen" | "prtsc" => vk(0x2C),
        "scroll_lock" => vk(0x91),
        "pause" => vk(0x13),
        "caps_lock" | "capslock" => vk(0x14),
        "num_lock" | "numlock" => vk(0x90),
        "menu" | "apps" => vk(0x5D),
        "kp_add" => vk(0x6B),
        "kp_subtract" => vk(0x6D),
        "kp_multiply" => vk(0x6A),
        "kp_divide" => vk(0x6F),
        "kp_decimal" => vk(0x6E),
        "minus" => ch('-'),
        "plus" => ch('+'),
        "equal" => ch('='),
        "comma" => ch(','),
        "period" => ch('.'),
        "slash" => ch('/'),
        "backslash" => ch('\\'),
        "semicolon" => ch(';'),
        "apostrophe" | "quote" => ch('\''),
        "quotedbl" => ch('"'),
        "grave" => ch('`'),
        "asciitilde" | "tilde" => ch('~'),
        "asciicircum" => ch('^'),
        "bracketleft" => ch('['),
        "bracketright" => ch(']'),
        "braceleft" => ch('{'),
        "braceright" => ch('}'),
        "underscore" => ch('_'),
        "asterisk" => ch('*'),
        "question" => ch('?'),
        "exclam" => ch('!'),
        "at" => ch('@'),
        "numbersign" => ch('#'),
        "dollar" => ch('$'),
        "percent" => ch('%'),
        "ampersand" => ch('&'),
        "parenleft" => ch('('),
        "parenright" => ch(')'),
        "less" => ch('<'),
        "greater" => ch('>'),
        "colon" => ch(':'),
        "bar" => ch('|'),
        _ => {
            if let Some(n) = name.strip_prefix('f') {
                if let Ok(n) = n.parse::<u16>() {
                    if (1..=24).contains(&n) {
                        return vk(0x70 + n - 1);
                    }
                }
            }
            if let Some(n) = name.strip_prefix("kp_") {
                if let Ok(n) = n.parse::<u16>() {
                    if n <= 9 {
                        return vk(0x60 + n);
                    }
                }
            }
            let mut chars = name.chars();
            match (chars.next(), chars.next()) {
                (Some(c), None) if c.is_ascii_alphabetic() => vk(c.to_ascii_uppercase() as u16),
                (Some(c), None) if c.is_ascii_digit() => vk(c as u16),
                (Some(c), None) => ch(c),
                _ => None,
            }
        }
    }
}

/// Parses "ctrl+shift+t", "Return", "super+e", "alt+F4". The last part is
/// the key; everything before it must be a modifier. A lone modifier is a
/// key press of that modifier ("shift" taps Shift). The plus key is "plus",
/// or a trailing "++" ("ctrl++" is Ctrl and '+').
pub fn parse_chord(text: &str) -> Result<Chord, String> {
    let raw = text.trim();
    if raw.is_empty() {
        return Err("key: nothing to press".into());
    }
    // "+" alone and "ctrl++": the key is the plus sign itself.
    let (mods_text, plus_key) = if raw == "+" {
        ("", true)
    } else if let Some(head) = raw.strip_suffix("++") {
        (head, true)
    } else {
        (raw, false)
    };
    if !plus_key && raw.ends_with('+') {
        return Err(format!("key: '{raw}' ends with '+' and names no key (the plus key is \"plus\")"));
    }
    let mut mods = Vec::new();
    let mut parts: Vec<&str> = if mods_text.is_empty() { Vec::new() } else { mods_text.split('+').collect() };
    let key = if plus_key {
        Key::Char('+')
    } else {
        let last = parts.pop().unwrap_or("");
        let p = last.trim().to_ascii_lowercase();
        if let Some(m) = modifier(&p) {
            Key::Vk(m)
        } else {
            match key(&p) {
                Some(k) => k,
                None => return Err(format!("key: '{last}' is not a key name")),
            }
        }
    };
    for part in parts {
        let p = part.trim().to_ascii_lowercase();
        match modifier(&p) {
            Some(m) => mods.push(m),
            None => return Err(format!("key: '{part}' is not a modifier (use ctrl, shift, alt, super)")),
        }
    }
    Ok(Chord { mods, key })
}

/// Keys whose scan code carries the extended flag (the keypad-adjacent
/// cluster and the right-hand modifiers); SendInput needs the flag for them.
pub fn is_extended(vk: u16) -> bool {
    matches!(
        vk,
        0x21..=0x28 | 0x2C..=0x2E | 0x5B | 0x5C | 0x5D | 0x6F | 0x90 | 0xA3 | 0xA5
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn chords_parse_modifiers_then_the_key() {
        assert_eq!(
            parse_chord("ctrl+shift+t").unwrap(),
            Chord { mods: vec![VK_CONTROL, VK_SHIFT], key: Key::Vk(b'T' as u16) }
        );
        assert_eq!(parse_chord("Return").unwrap(), Chord { mods: vec![], key: Key::Vk(VK_RETURN) });
        assert_eq!(parse_chord("alt+F4").unwrap(), Chord { mods: vec![VK_MENU], key: Key::Vk(0x73) });
        assert_eq!(parse_chord("super+e").unwrap(), Chord { mods: vec![VK_LWIN], key: Key::Vk(b'E' as u16) });
        assert_eq!(parse_chord("Page_Down").unwrap().key, Key::Vk(0x22));
        assert_eq!(parse_chord("kp_5").unwrap().key, Key::Vk(0x65));
        assert_eq!(parse_chord("ctrl+plus").unwrap().key, Key::Char('+'));
        assert_eq!(parse_chord("ctrl++").unwrap(), Chord { mods: vec![VK_CONTROL], key: Key::Char('+') });
        assert_eq!(parse_chord("+").unwrap().key, Key::Char('+'));
        assert_eq!(parse_chord("shift").unwrap().key, Key::Vk(VK_SHIFT), "a lone modifier taps it");
        assert_eq!(parse_chord("ç").unwrap().key, Key::Char('ç'), "layout keys resolve at runtime");
    }

    #[test]
    fn nonsense_is_refused_with_the_part_named() {
        assert!(parse_chord("").is_err());
        assert!(parse_chord("ctrl+").is_err());
        assert!(parse_chord("foo+a").unwrap_err().contains("foo"));
        assert!(parse_chord("banana").unwrap_err().contains("banana"));
    }

    #[test]
    fn extended_keys_are_the_navigation_cluster() {
        assert!(is_extended(0x2E)); // Delete
        assert!(is_extended(0x26)); // Up
        assert!(is_extended(0x5B)); // LWin
        assert!(!is_extended(b'A' as u16));
        assert!(!is_extended(VK_RETURN));
    }
}
