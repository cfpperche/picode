// Extra keys for the phone terminal accessory (ADR-0044). One scrolling
// row, Fable-first order so esc/tab/ctrl/alt/arrows/^C fit a 390px
// screen without scrolling; home/end/pages and the symbols a software
// keyboard buries follow. Hide is a separate pinned button, not a key.

export const KEYS = [
  { id: "esc", label: "esc", seq: "\x1b" },
  { id: "tab", label: "tab", seq: "\t" },
  { id: "ctrl", label: "ctrl", mod: "ctrl" },
  { id: "alt", label: "alt", mod: "alt" },
  { id: "left", label: "◀", seq: "\x1b[D" },
  { id: "up", label: "▲", seq: "\x1b[A" },
  { id: "down", label: "▼", seq: "\x1b[B" },
  { id: "right", label: "▶", seq: "\x1b[C" },
  { id: "intr", label: "^C", seq: "\x03", title: "Interrupt (Ctrl+C)" },
  { id: "home", label: "home", seq: "\x1b[H" },
  { id: "end", label: "end", seq: "\x1b[F" },
  { id: "pgup", label: "pgup", seq: "\x1b[5~" },
  { id: "pgdn", label: "pgdn", seq: "\x1b[6~" },
  { id: "pipe", label: "|", seq: "|" },
  { id: "tilde", label: "~", seq: "~" },
  { id: "slash", label: "/", seq: "/" },
  { id: "dash", label: "-", seq: "-" },
];
