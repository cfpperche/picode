import test from "node:test";
import assert from "node:assert/strict";
import { crossFolderConfirm, edgeGrant, edgeLinks, endOf, enrolOffer, linkChipTitle, linkCounts, peerIndex, peerKey, removeConfirm } from "./canvasGrants.js";

const panels = [
  { id: "p1", kind: "agent", ref: "a1" },
  { id: "p2", kind: "terminal", ref: "t1" },
  { id: "p3", kind: "agent", ref: "a2" },
  { id: "p4", kind: "note", ref: "pin1" },
];
const owner = (kind, ownerId, label, workspaceId) => ({ kind, ownerId, label, workspaceId, cli: "pi", sessionKey: "s" });
const conn = (kind, ownerId, extra) => ({ id: "peer_" + ownerId, kind, ownerId, label: "conn " + ownerId, workspaceId: "w1", createdAt: "2026-09-10T10:00:00Z", revokedAt: null, active: true, ...extra });
const payload = {
  owners: [owner("agent", "a1", "Atlas", "w1"), owner("terminal", "t1", "shell", "w1"), owner("agent", "a2", "Borg", "w2")],
  connections: [conn("agent", "a1"), conn("terminal", "t1"), conn("agent", "a2", { workspaceId: "w2" })],
};
const edge = (id, a, b) => ({ id, aPanel: a, bPanel: b, createdAt: "" });

test("peerIndex keys a panel's (kind, ref) and reads the store's own active flag", () => {
  const ix = peerIndex(payload);
  assert.equal(peerKey("agent", "a1"), "agent:a1");
  assert.equal(ix.get("agent:a1").state, "on");
  assert.equal(ix.get("agent:a2").workspaceId, "w2");
  // Every reason an ADR-0104 connection stops granting, and the one that
  // never started: each must be told apart, because each has a different
  // next action for the owner.
  for (const [rows, state] of [
    [[], "off"],
    [[conn("agent", "a1", { active: false, revokedAt: "2026-09-10T11:00:00Z" })], "revoked"],
    [[conn("agent", "a1", { active: false })], "stale"],
    [[conn("agent", "a1")], "on"],
  ]) {
    assert.equal(peerIndex({ owners: [owner("agent", "a1", "Atlas", "w1")], connections: rows }).get("agent:a1").state, state);
  }
  // A live row wins over an older revoked one; a newer revoked row wins over
  // an older stale one.
  const mixed = peerIndex({
    owners: [owner("agent", "a1", "Atlas", "w1")],
    connections: [conn("agent", "a1", { id: "old", active: false, createdAt: "2026-09-09T00:00:00Z" }), conn("agent", "a1", { id: "new", active: false, revokedAt: "z", createdAt: "2026-09-10T00:00:00Z" })],
  });
  assert.equal(mixed.get("agent:a1").state, "revoked");
  // A connection whose session left the fleet is gone, not merely off.
  const orphan = peerIndex({ owners: [], connections: [conn("agent", "a1", { active: false })] });
  assert.equal(orphan.get("agent:a1").state, "gone");
  assert.equal(endOf({ kind: "agent", ref: "nobody" }, orphan).state, "gone");
});

test("an edge grants only when both ends hold a live connection", () => {
  const ix = peerIndex(payload);
  const g = edgeGrant(edge("e1", "p1", "p2"), panels, ix, { p1: "Atlas", p2: "shell" });
  assert.equal(g.grants, true);
  assert.equal(g.reason, "");
  assert.equal(g.cross, false);
  assert.deepEqual(g.ends.map((e) => e.name), ["Atlas", "shell"]);
  // Two folders is the one new power; it is reported, never hidden.
  assert.equal(edgeGrant(edge("e2", "p1", "p3"), panels, ix).cross, true);
  // An end that is not on the board is the one case the canvas must not
  // draw: null, not a half-edge.
  assert.equal(edgeGrant(edge("e3", "p1", "gone"), panels, ix), null);
});

test("a broken end reads broken, with the reason that end actually has", () => {
  const cases = [
    [{ active: false }, "Atlas’s session changed; the link grants nothing."],
    [{ active: false, revokedAt: "2026-09-10T11:00:00Z" }, "Atlas’s connection was revoked; the link grants nothing."],
  ];
  for (const [extra, reason] of cases) {
    const ix = peerIndex({ owners: payload.owners, connections: [conn("agent", "a1", extra), conn("terminal", "t1")] });
    const g = edgeGrant(edge("e1", "p1", "p2"), panels, ix, { p1: "Atlas", p2: "shell" });
    assert.equal(g.grants, false);
    assert.equal(g.reason, reason);
  }
  // Never enrolled, and the session gone from the fleet entirely.
  const off = peerIndex({ owners: payload.owners, connections: [conn("terminal", "t1")] });
  assert.equal(edgeGrant(edge("e1", "p1", "p2"), panels, off, { p1: "Atlas" }).reason, "Atlas is not connected; the link grants nothing.");
  const gone = peerIndex({ owners: [owner("terminal", "t1", "shell", "w1")], connections: [conn("terminal", "t1")] });
  assert.equal(edgeGrant(edge("e1", "p1", "p2"), panels, gone, { p1: "Atlas" }).reason, "Atlas is gone; the link grants nothing.");
  // With no name from the caller, the reason still names something: the
  // peer's own label, else the ref.
  assert.match(edgeGrant(edge("e1", "p1", "p2"), panels, off).reason, /^Atlas is not connected/);
});

test("a panel header counts the links it carries, and says what the count means", () => {
  const ix = peerIndex({ owners: payload.owners, connections: [conn("agent", "a1"), conn("terminal", "t1"), conn("agent", "a2", { workspaceId: "w2", active: false })] });
  const edges = [edge("e1", "p1", "p2"), edge("e2", "p1", "p3"), edge("e3", "p1", "nowhere")];
  assert.deepEqual(edgeLinks(edges, "p3").map((e) => e.id), ["e2"]);
  assert.deepEqual(edgeLinks(edges, "").length, 0);
  const counts = linkCounts(edges, panels, ix);
  // e3 has an end that is not on the board and is counted nowhere.
  assert.deepEqual(counts.p1, { count: 2, broken: 1 });
  assert.deepEqual(counts.p2, { count: 1, broken: 0 });
  assert.deepEqual(counts.p3, { count: 1, broken: 1 });
  assert.equal(counts.nowhere, undefined);
  assert.equal(linkChipTitle(counts.p1), "2 links, 1 broken: it grants nothing. Read them in Messages.");
  assert.equal(linkChipTitle(counts.p2), "1 link: these sessions may message each other. Read them in Messages.");
  assert.equal(linkChipTitle(counts.p3), "1 link, broken: it grants nothing. Read them in Messages.");
  assert.equal(linkChipTitle(undefined), "");
});

test("nothing is written until the owner answers the right question", () => {
  const ix = peerIndex(payload);
  const both = edgeGrant(edge("e1", "p1", "p2"), panels, ix, { p1: "Atlas", p2: "shell" });
  // Both ends live and one folder: there is nothing to ask, so it does not.
  assert.equal(enrolOffer(both.ends), null);
  assert.equal(removeConfirm({ grants: false }), null);
  assert.equal(removeConfirm(both).message, "Atlas and shell will no longer be able to message each other.");
  assert.equal(removeConfirm(both).danger, true);

  const one = peerIndex({ owners: payload.owners, connections: [conn("terminal", "t1")] });
  const offer = enrolOffer(edgeGrant(edge("e1", "p1", "p2"), panels, one, { p1: "Atlas", p2: "shell" }).ends);
  assert.equal(offer.title, "Connect Atlas?");
  assert.equal(offer.confirmLabel, "Connect and link");
  assert.match(offer.message, /lets Atlas and shell message each other/);
  // The limit is in the sentence, every time: an edge is never a transcript.
  assert.match(offer.message, /Neither can read the other’s history\./);

  const neither = peerIndex({ owners: payload.owners, connections: [] });
  const two = enrolOffer(edgeGrant(edge("e1", "p1", "p2"), panels, neither, { p1: "Atlas", p2: "shell" }).ends);
  assert.equal(two.title, "Connect Atlas and shell?");
  assert.equal(two.confirmLabel, "Connect both and link");

  // A session that is gone cannot be enrolled, so it is refused rather than
  // offered an action that would fail.
  const dead = peerIndex({ owners: [owner("terminal", "t1", "shell", "w1")], connections: [conn("terminal", "t1")] });
  const blocked = enrolOffer(edgeGrant(edge("e1", "p1", "p2"), panels, dead, { p1: "Atlas", p2: "shell" }).ends);
  assert.equal(blocked.blocked, true);
  assert.match(blocked.message, /^Atlas is gone/);

  // Nor can a session the store would refuse to enrol: EnablePeer needs a
  // recorded session, a workspace and a CLI before it mints a credential.
  const green = peerIndex({ owners: [{ ...owner("agent", "a1", "Atlas", "w1"), sessionKey: "" }, owner("terminal", "t1", "shell", "w1")], connections: [conn("terminal", "t1")] });
  const early = enrolOffer(edgeGrant(edge("e1", "p1", "p2"), panels, green, { p1: "Atlas", p2: "shell" }).ends);
  assert.equal(early.blocked, true);
  assert.equal(early.message, "Atlas has no conversation yet; open it once, then draw the link.");
});

test("the cross-folder question is asked on its own and names both folders", () => {
  const ix = peerIndex(payload);
  const g = edgeGrant(edge("e2", "p1", "p3"), panels, ix, { p1: "Atlas", p3: "Borg" });
  const ask = crossFolderConfirm(g.ends, { p1: "~/picode", p3: "~/other" });
  assert.equal(ask.title, "Link across two folders?");
  assert.equal(ask.confirmLabel, "Link across folders");
  assert.match(ask.message, /Atlas works in ~\/picode and Borg in ~\/other\./);
  assert.match(ask.message, /neither can read the other’s history/);
  // It never merges into the enrolment sentence, which asks its own thing.
  assert.equal(enrolOffer(g.ends), null);
  // A folder the client cannot name is said to be unnamed, not invented.
  assert.match(crossFolderConfirm(g.ends, {}).message, /Atlas works in an unnamed folder and Borg in an unnamed folder\./);
  assert.equal(crossFolderConfirm([], {}), null);
});
