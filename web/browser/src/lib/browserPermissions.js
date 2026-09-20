// The page-side half of the work browser's site permissions (slice 3,
// Browser permissions): what the Settings dialog hands back to the shell on
// every load, and the Ask prompt's copy. Pure, so the rows below are tested
// without the desktop shell.

// The every-site origin: a kind's default policy (the dialog's per-kind
// choice). A site's own standing is keyed by its URI.
export const ALL_SITES = "*";

// What this standing tells the shell, or null when it is only a reported
// decision and not saved policy. The every-site rows are always policy; a
// site row only when it is a standing (the Ask prompt's "Always allow").
// Without that filter a one-off Allow/Block — which the shell reports like
// any other decision — would quietly become permanent on the next load.
export function permissionPush(st) {
  if (!st || !st.kind || !st.decision) return null;
  if (st.origin === ALL_SITES) return { kind: st.kind, state: st.decision, origin: null };
  if (!st.standing) return null;
  return { kind: st.kind, state: st.decision, origin: st.origin };
}

// The kinds as the prompt reads them: "Meet wants to use your camera".
const ASK_ACTIONS = {
  camera: "use your camera",
  microphone: "use your microphone",
  location: "know your location",
  notifications: "send notifications",
  clipboard: "read your clipboard",
  autoplay: "play media",
  sensors: "use your sensors",
  midi: "use MIDI devices",
  fonts: "use your local fonts",
  filesystem: "read or write files",
  unknown: "use a permission",
};

export function askHost(origin) {
  try {
    return new URL(origin).host || origin;
  } catch {
    return origin || "This site";
  }
}

export function askTitle(ask) {
  const kind = ask?.kind || "unknown";
  return `${askHost(ask?.origin)} wants to ${ASK_ACTIONS[kind] || ASK_ACTIONS.unknown}`;
}
