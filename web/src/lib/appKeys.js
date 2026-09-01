import { fromEvent, effectiveKeys } from "./piKey.js";
import { readAppKeyOverrides } from "./appKeyPrefs.js";

// Same-group entries must stay contiguous — AppKeys.jsx groups via a linear
// walk (copied from PiKeys.jsx), not a sort/groupBy.
export const CATALOG = [
  { id: "app.palette.toggle", group: "Global", label: "Command palette", defaults: ["ctrl+k", "super+k"] },
  { id: "app.terminal.new", group: "Global", label: "New terminal", defaults: ["ctrl+`", "super+`"] },
  { id: "composer.voice.toggle", group: "Composer", label: "Toggle voice", defaults: ["ctrl+shift+o", "super+shift+o"] },
  { id: "composer.dictate", group: "Composer", label: "Dictate", defaults: ["ctrl+d", "super+d"] },
];

export function matchAction(id, ev, overrides = readAppKeyOverrides()) {
  const action = CATALOG.find((a) => a.id === id);
  if (!action) return false;
  const chord = fromEvent(ev);
  return !!chord && effectiveKeys(action, overrides).includes(chord);
}

export function primaryChord(id, overrides) {
  const action = CATALOG.find((a) => a.id === id);
  if (!action) return "";
  return effectiveKeys(action, overrides)[0] || "";
}

const PART_LABELS = { ctrl: "Ctrl", shift: "Shift", alt: "Alt", super: "Cmd" };

export function formatChord(chord) {
  if (!chord) return "";
  return chord
    .split("+")
    .map((part) => PART_LABELS[part] || (part.length === 1 ? part.toUpperCase() : part[0].toUpperCase() + part.slice(1)))
    .join("+");
}
