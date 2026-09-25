// canvasGrants.js — what a Canvas edge grants *right now* (ADR-0116 §7).
//
// An edge is the owner's recorded intent that two sessions may exchange
// messages; it grants exactly ADR-0104's mailbox contact and nothing else —
// no transcript, no scrollback, no history. The store derives that grant on
// every contact read from the *live* edge joined with `peer_connections`,
// so the drawn line and the real grant can disagree for exactly the reasons
// ADR-0104 already invalidates a connection: the session moved, the owner
// revoked it, the target is gone, or it was never enrolled.
//
// ADR-0116 §7 refuses to draw that difference as a decorative dotted line:
// **a broken end must read broken**, with a one-line reason. This module is
// the one place that decides which, joining a canvas's `edges` + `panels`
// with `GET /api/communication` (`owners` + `connections`, whose `active`
// flag *is* the store's `peerCurrent`). Pure: both the canvas and the
// Messages audit list read the same answer, because a grant shown in two
// places that disagree is worse than one place.
//
// It never asks the server "does this grant?" — there is no such route, and
// inventing one would be a second source of truth for a security control.

const str = (v) => (typeof v === "string" ? v : "");

// The key a panel and a peer row agree on: a panel's (kind, ref) is exactly
// the connection's (kind, ownerId) — `agent` matches an agent id, `terminal`
// a terminal id (docs/architecture/canvas.md, *The contact union*).
export const peerKey = (kind, ref) => str(kind) + ":" + str(ref);

// peerIndex({owners, connections}) -> Map key -> { owner, connection, state,
// label, workspaceId }. One pass over the payload `GET /api/communication`
// answers. A session may hold several connection rows (re-enrolling makes a
// new one); the live one wins, else the newest that is still recorded, so a
// revoked row is not mistaken for "never enrolled".
export function peerIndex(payload) {
  const out = new Map();
  const owners = Array.isArray(payload?.owners) ? payload.owners : [];
  for (const o of owners) {
    if (!o || !nonEmpty(o.ownerId)) continue;
    // `ready` is what the store's EnablePeer demands before it will enrol
    // anything: a recorded session, a workspace and a CLI. A session that
    // has never run has none, so offering to connect it would be offering an
    // action that answers 409.
    const ready = nonEmpty(o.sessionKey) && nonEmpty(o.workspaceId) && nonEmpty(o.cli);
    out.set(peerKey(o.kind, o.ownerId), { owner: o, connection: null, state: "off", ready, label: str(o.label), cli: str(o.cli), workspaceId: str(o.workspaceId) });
  }
  const rows = Array.isArray(payload?.connections) ? payload.connections : [];
  for (const c of rows) {
    if (!c || !nonEmpty(c.ownerId)) continue;
    const key = peerKey(c.kind, c.ownerId);
    const at = out.get(key);
    // A connection whose owner is not in `owners` is a row for a session the
    // fleet no longer has: `gone`, and it cannot be repaired by enrolling.
    if (!at) {
      out.set(key, { owner: null, connection: c, state: "gone", ready: false, label: str(c.label), cli: str(c.cli), workspaceId: str(c.workspaceId) });
      continue;
    }
    if (better(c, at.connection)) at.connection = c;
  }
  for (const at of out.values()) {
    if (at.state === "gone") continue;
    at.state = connState(at.connection);
    if (!at.label) at.label = str(at.connection && at.connection.label);
  }
  return out;
}

const nonEmpty = (v) => typeof v === "string" && v.trim() !== "";
// The row that decides: active beats everything, then the most recently
// created, so "revoked" reflects the last thing the owner actually did.
function better(next, held) {
  if (!held) return true;
  if (!!next.active !== !!held.active) return !!next.active;
  return str(next.createdAt) > str(held.createdAt);
}
function connState(c) {
  if (!c) return "off";
  if (c.active) return "on";
  if (c.revokedAt) return "revoked";
  return "stale";
}

// endOf(panel, index) -> { key, kind, ref, state, label, workspaceId } for
// one end of an edge. A panel of a kind with no mailbox (a note, a file, a
// diff) can hold no contact at all; the store refuses such an edge, so this
// only ever answers for data that got in some other way.
export function endOf(panel, index) {
  const kind = str(panel && panel.kind);
  const ref = str(panel && panel.ref);
  const key = peerKey(kind, ref);
  const at = index instanceof Map ? index.get(key) : null;
  if (!at) return { key, kind, ref, state: "gone", ready: false, label: "", cli: "", workspaceId: "" };
  return { key, kind, ref, state: at.state, ready: !!at.ready, label: at.label, cli: at.cli, workspaceId: at.workspaceId };
}

const REASON = {
  gone: (n) => `${n} is gone; the link grants nothing.`,
  off: (n) => `${n} is not connected; the link grants nothing.`,
  revoked: (n) => `${n}’s connection was revoked; the link grants nothing.`,
  stale: (n) => `${n}’s session changed; the link grants nothing.`,
};

// edgeGrant(edge, panels, index, names) -> null when an end is not on the
// board (the canvas must not draw that one), else:
//
//   { grants, ends: [a, b], reason, cross }
//
// `grants` is true only when **both** ends hold a live connection — the
// same conjunction `peerEdgeContacts` applies in the store. `reason` is the
// one line a broken edge shows; it is empty when the edge grants. `cross`
// says the two ends sit in different workspaces, which is the one new power
// ADR-0116 §3 adds and §5 makes the owner confirm separately.
//
// `names` is what the caller already calls each panel (a model's name, a
// peer label); an end falls back to the peer's own label, then to its ref,
// so a reason always names something the owner can recognise.
export function edgeGrant(edge, panels, index, names) {
  const list = Array.isArray(panels) ? panels : [];
  const a = list.find((p) => p && p.id === (edge && edge.aPanel));
  const b = list.find((p) => p && p.id === (edge && edge.bPanel));
  if (!a || !b) return null;
  const ends = [a, b].map((panel) => {
    const end = endOf(panel, index);
    return { ...end, panel, name: nameOf(panel, end, names) };
  });
  const broken = ends.filter((e) => e.state !== "on");
  const cross = ends[0].workspaceId !== ends[1].workspaceId;
  return {
    grants: broken.length === 0,
    ends,
    cross,
    reason: broken.length ? REASON[broken[0].state](broken[0].name) : "",
  };
}

function nameOf(panel, end, names) {
  const given = names && (names instanceof Map ? names.get(panel.id) : names[panel.id]);
  return str(given) || end.label || panel.ref;
}

// edgeLinks(edges, panelId) -> the edges that touch one panel. The count is
// what a panel's header shows beside the line the plane draws, so a reader
// sees how many links a panel carries without tracing them
// (docs/architecture/canvas.md, *Edges*).
export function edgeLinks(edges, panelId) {
  if (!Array.isArray(edges) || !nonEmpty(panelId)) return [];
  return edges.filter((e) => e && (e.aPanel === panelId || e.bPanel === panelId));
}

// linkCounts(edges, panels, index, names) -> { [panelId]: { count, broken } }.
// One walk for the whole board, so a 200-panel canvas costs one pass and not
// one per header.
export function linkCounts(edges, panels, index, names) {
  const out = {};
  for (const e of Array.isArray(edges) ? edges : []) {
    const g = edgeGrant(e, panels, index, names);
    if (!g) continue;
    for (const id of [e.aPanel, e.bPanel]) {
      const at = out[id] || (out[id] = { count: 0, broken: 0 });
      at.count += 1;
      if (!g.grants) at.broken += 1;
    }
  }
  return out;
}

// linkChipTitle({count, broken}) -> the header chip's one line. It says what
// the count means, because "2" beside a panel means nothing on its own.
export function linkChipTitle(at) {
  const count = (at && at.count) || 0;
  const broken = (at && at.broken) || 0;
  if (!count) return "";
  const word = count === 1 ? "link" : "links";
  const live = count - broken;
  if (!broken) return `${count} ${word}: these sessions may message each other. Read them in Messages.`;
  if (!live) return `${count} ${word}, ${count === 1 ? "broken" : "all broken"}: ${count === 1 ? "it grants" : "they grant"} nothing. Read them in Messages.`;
  return `${count} ${word}, ${broken} broken: ${broken === 1 ? "it grants" : "they grant"} nothing. Read them in Messages.`;
}

// ---- what the owner is asked before a line is drawn (ADR-0116 §5) --------
//
// "An edge never enrols silently." Two questions, never merged into one:
// the enrolment is the existing owner action this ADR reuses, and the
// cross-folder pairing is the one new power it adds. Each is one line and
// one action, and **nothing is written until it is confirmed** — the copy
// lives here so it is tested once and says the same thing wherever it is
// asked from.
//
// Both lines name the limit, because the limit is the decision: a link is a
// mailbox contact, and neither session can read the other's history.

// enrolOffer(ends) -> null when both ends already hold a live connection
// (nothing to enrol, so nothing to ask), else the askConfirm argument.
// `ends` are edgeGrant's, whose `state` says which side is missing.
export function enrolOffer(ends) {
  const pair = Array.isArray(ends) ? ends : [];
  if (pair.length !== 2) return null;
  const missing = pair.filter((e) => e.state !== "on");
  if (!missing.length) return null;
  // A session that is gone cannot be enrolled: there is nothing left to
  // enrol. Nor can one that has never held a conversation — the store needs
  // a recorded session before it will mint a credential. Both are said
  // plainly instead of being offered an action that would fail.
  const gone = missing.filter((e) => e.state === "gone");
  if (gone.length) {
    return {
      title: "Can’t link these two",
      message: `${list(gone.map((e) => e.name))} ${gone.length > 1 ? "are" : "is"} gone, so a link between them would grant nothing.`,
      confirmLabel: "",
      blocked: true,
    };
  }
  const unready = missing.filter((e) => !e.ready);
  if (unready.length) {
    return {
      title: "Can’t connect these yet",
      message: `${list(unready.map((e) => e.name))} ${unready.length > 1 ? "have" : "has"} no conversation yet; open ${unready.length > 1 ? "them" : "it"} once, then draw the link.`,
      confirmLabel: "",
      blocked: true,
    };
  }
  const who = list(missing.map((e) => e.name));
  const both = pair.map((e) => e.name);
  return {
    title: `Connect ${who}?`,
    message: `${who} ${missing.length > 1 ? "are" : "is"} not connected yet — connecting ${missing.length > 1 ? "them" : "it"} and drawing this link lets ${both[0]} and ${both[1]} message each other. Neither can read the other’s history.`,
    confirmLabel: missing.length > 1 ? "Connect both and link" : "Connect and link",
  };
}

// crossFolderConfirm(ends, folders) -> the second question, asked only when
// the two ends live in different workspaces. It names **both folders**,
// separately from the enrolment sentence, because pairing across folders is
// the only thing an edge can do that the workspace rule could not.
export function crossFolderConfirm(ends, folders) {
  const pair = Array.isArray(ends) ? ends : [];
  if (pair.length !== 2) return null;
  const at = (e) => str(folders && (folders instanceof Map ? folders.get(e.panel && e.panel.id) : folders[e.panel && e.panel.id])) || "an unnamed folder";
  return {
    title: "Link across two folders?",
    message: `${pair[0].name} works in ${at(pair[0])} and ${pair[1].name} in ${at(pair[1])}. Linking them lets these two sessions message each other across folders — nothing else is shared, and neither can read the other’s history.`,
    confirmLabel: "Link across folders",
  };
}

// removeConfirm(grant) -> null when the edge grants nothing (removing a line
// that grants nothing takes nothing away, so it does not ask), else the one
// line that says what is lost.
export function removeConfirm(grant) {
  if (!grant || !grant.grants) return null;
  const [a, b] = grant.ends;
  return {
    title: "Remove this link?",
    message: `${a.name} and ${b.name} will no longer be able to message each other.`,
    confirmLabel: "Remove link",
    danger: true,
  };
}

function list(names) {
  const v = names.filter(Boolean);
  if (v.length < 2) return v[0] || "This session";
  return v.slice(0, -1).join(", ") + " and " + v[v.length - 1];
}
