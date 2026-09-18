// The Servers panel's decisions, kept out of the JSX so they can be tested:
// what the panel shows first, what waits behind a disclosure, what a row is
// called, and which verbs a row may offer. The server reports every listener
// it can see — including the ones the panel keeps behind "Show N more" — so
// revealing the rest costs no round trip.

// A page is what PiCode's browser tab can show; an API answered but is not a
// page (JSON, plain text, a 404); opaque answered nothing at all.
export const PAGE = "page";
export const API = "api";
export const OPAQUE = "opaque";

// isOpenable answers whether the row's URL leads somewhere worth opening. The
// panel's first version offered Open for anything that answered below 500,
// which put an agent CLI's HTTPS control channel in the list as a page nobody
// could open.
export function isOpenable(s) {
  return !!(s && s.kind === PAGE);
}

// canStop and canHide need the identity of the process holding the socket:
// that is what the server re-derives before it signals anything (Stop), and
// what makes a Hide mean "this listener" instead of "this port forever". A
// port PiCode cannot attribute has neither verb — the row says so instead.
export function canStop(s) {
  return !!(s && Number(s.pid) > 0 && s.startKey);
}

export function canHide(s) {
  return canStop(s);
}

export function serverName(s) {
  // A page with no <title> is genuinely unnamed; a port that is not a page has
  // no name to give, and the port number is its identity — "unnamed page"
  // next to a "not a page" badge would be the panel arguing with itself.
  if (s.title) return s.title;
  if (s.tool) return s.tool;
  return s.kind === PAGE ? "unnamed page" : "";
}

// serverNote is the quiet truth about a row, in the words the panel uses:
// what a port is, when it is not a page; whether the answer came over TLS,
// which is the case the browser will warn about.
export function serverNote(s) {
  const bits = [];
  if (s.scheme === "https") bits.push("HTTPS");
  if (s.kind === API) bits.push("not a page");
  if (s.kind === OPAQUE) bits.push(s.state === "starting" ? "starting" : "not answering");
  return bits.join(" · ");
}

// serverMeta is who holds the port and, when the process is known, since when.
export function serverMeta(s, now = Date.now()) {
  const who = s.ownerName
    ? (s.ownerKind === "agent" ? `agent "${s.ownerName}"` : `terminal "${s.ownerName}"`)
    : "started outside PiCode";
  const bits = [who];
  if (s.workspace) bits.push(s.workspace);
  const age = ageLabel(s.startedAt, now);
  if (age) bits.push(age);
  return bits.join(" · ");
}

// ageLabel is "just now", "8 min", "2 h", "3 d" — how long the process has
// been up. The panel never invents a time: no start token, no age.
export function ageLabel(startedAt, now = Date.now()) {
  const started = Date.parse(String(startedAt || ""));
  if (!Number.isFinite(started)) return "";
  const seconds = Math.max(0, Math.round((now - started) / 1000));
  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} h`;
  return `${Math.floor(seconds / 86400)} d`;
}

// defaultVisible is the disclosure rule: a page is what the panel is for, and
// a listener PiCode started is always the human's business. Everything else —
// a port that only answers an API, outside PiCode, on a port the list merely
// guesses at — waits behind "Show N more".
export function defaultVisible(s) {
  return !!s && !s.hidden && (s.kind === PAGE || !!s.ownerKind);
}

// sections splits the list the way the rail reads it: the listeners PiCode
// started (grouped by owner, pages first, then by port), then the rest. Hidden
// rows belong to neither — they are behind their own disclosure.
export function sections(rows) {
  const visible = (rows || []).filter(defaultVisible);
  return {
    mine: rank(visible.filter((s) => !!s.ownerKind)),
    other: rank(visible.filter((s) => !s.ownerKind)),
  };
}

export function otherRows(rows) {
  return rank((rows || []).filter((s) => !s.hidden && !defaultVisible(s)));
}

export function hiddenRows(rows) {
  return rank((rows || []).filter((s) => !!s.hidden));
}

const ORDER = { [PAGE]: 0, [API]: 1, [OPAQUE]: 2 };

function rank(rows) {
  return [...rows].sort((a, b) => (ORDER[a.kind] ?? 3) - (ORDER[b.kind] ?? 3) || a.port - b.port);
}

// stopPrompt is the confirmation copy for the one destructive verb in the
// panel: it names the port, says whose process it is, and admits the cost.
export function stopPrompt(s) {
  const who = s.ownerName
    ? ` It is running in ${s.ownerKind === "agent" ? "agent" : "terminal"} "${s.ownerName}".`
    : "";
  return {
    title: `Stop ${s.tool || "the server"} on port ${s.port}?`,
    message: `PiCode sends a stop signal to the process holding port ${s.port}.${who} Anything it was running stops too.`,
  };
}
