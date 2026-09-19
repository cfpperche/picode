import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import {
  PICK_STYLES,
  batchMessage,
  cropRect,
  parseAnnotMessage,
  parsePick,
  parseStatePayload,
  pickLabel,
  pickScript,
  resolveSendTarget,
  sendLabel,
  stateItems,
  stillToViewport,
  stylesToCSS,
} from "./annotate.js";

test("a click on the still becomes a viewport point, scale only", () => {
  // 1:1
  assert.deepEqual(
    stillToViewport({ clickX: 100, clickY: 50, naturalWidth: 900, naturalHeight: 600, displayWidth: 900, displayHeight: 600 }),
    { x: 100, y: 50 },
  );
  // the still is displayed at half size
  assert.deepEqual(
    stillToViewport({ clickX: 449, clickY: 299, naturalWidth: 900, naturalHeight: 600, displayWidth: 450, displayHeight: 300 }),
    { x: 898, y: 598 },
  );
  // an unmeasurable still, and a click outside it, pick nothing
  assert.equal(stillToViewport({ clickX: 10, clickY: 10, naturalWidth: 0, naturalHeight: 600, displayWidth: 900, displayHeight: 600 }), null);
  assert.equal(stillToViewport({ clickX: 950, clickY: 10, naturalWidth: 900, naturalHeight: 600, displayWidth: 900, displayHeight: 600 }), null);
  assert.equal(stillToViewport({ clickX: -1, clickY: 10, naturalWidth: 900, naturalHeight: 600, displayWidth: 900, displayHeight: 600 }), null);
});

test("the crop is the element plus padding, clamped to the picture", () => {
  const base = { naturalWidth: 900, naturalHeight: 600, displayWidth: 900, displayHeight: 600 };
  assert.deepEqual(cropRect({ ...base, rect: { x: 100, y: 100, width: 200, height: 40 } }), { x: 92, y: 92, width: 216, height: 56 });
  // at the edge the padding is clipped, never negative
  assert.deepEqual(cropRect({ ...base, rect: { x: 0, y: 0, width: 60, height: 40 } }), { x: 0, y: 0, width: 68, height: 48 });
  // a half-size still scales the rect too
  assert.deepEqual(
    cropRect({ naturalWidth: 900, naturalHeight: 600, displayWidth: 450, displayHeight: 300, rect: { x: 50, y: 50, width: 100, height: 20 } }),
    { x: 92, y: 92, width: 216, height: 56 },
  );
  // a sliver (or a page that moved) crops nothing
  assert.equal(cropRect({ ...base, rect: { x: 10, y: 10, width: 0, height: 0 }, padding: 0 }), null);
  assert.equal(cropRect({ naturalWidth: 0, naturalHeight: 0, displayWidth: 0, displayHeight: 0, rect: { x: 1, y: 1, width: 5, height: 5 } }), null);
});

test("the page script carries the point and only the point", () => {
  const s = pickScript(120, 45);
  assert.ok(s.includes("document.elementFromPoint(120, 45)"));
  for (const p of PICK_STYLES) assert.ok(s.includes(`"${p}"`), p);
  assert.ok(!/\$\{/.test(s.replace(/\$\{JSON.stringify\(PICK_STYLES\)\}/g, "")));
});

test("the bridge's answer is parsed in both shapes, and junk is refused", () => {
  const payload = JSON.stringify({
    selector: "button.save",
    tag: "button",
    html: "<button class=\"save\">Save</button>",
    rect: { x: 10, y: 20, width: 80, height: 32 },
    styles: { color: "rgb(0, 0, 0)", "font-size": "14px" },
  });
  const direct = parsePick(payload);
  const enveloped = parsePick({ result: { value: payload } });
  assert.equal(direct.selector, "button.save");
  assert.deepEqual(direct.rect, { x: 10, y: 20, width: 80, height: 32 });
  assert.deepEqual(enveloped.rect, direct.rect);
  assert.equal(parsePick(""), null);
  assert.equal(parsePick("not json"), null);
  assert.equal(parsePick({ result: { value: "" } }), null);
  assert.equal(parsePick(JSON.stringify({ selector: "x" })), null); // no rect, no pick
});

test("styles become readable lines and the label is one line", () => {
  assert.equal(stylesToCSS({ color: "rgb(0, 0, 0)", "font-size": " 14px ", display: "" }), "color: rgb(0, 0, 0)\nfont-size: 14px");
  assert.equal(stylesToCSS(null), "");
  assert.equal(
    pickLabel({ selector: "button.save", rect: { width: 79.6, height: 32.2 } }),
    "button.save — 80×32",
  );
  assert.equal(pickLabel(null), "");
});

test("the Send button carries the pending count", () => {
  assert.equal(sendLabel(0), "Send");
  assert.equal(sendLabel(1), "Send 1");
  assert.equal(sendLabel(5), "Send 5");
  assert.equal(sendLabel(undefined), "Send");
});

test("a state sync becomes trusted items, junk dropped", () => {
  const msg = {
    kind: "state",
    count: 2,
    items: [
      { n: 1, selector: "h1", tag: "h1", html: "<h1>Hi</h1>", rect: { x: 1, y: 2, width: 30, height: 10 }, styles: { color: "red" }, vw: 800, vh: 600, comment: "fix", saved: true },
      { n: 2, selector: "p", rect: { x: 1, y: 2, width: 30, height: 10 }, comment: "", saved: false },
      { n: 3, selector: "junk" }, // no rect, no item
      null,
      "x",
    ],
  };
  const items = stateItems(msg);
  assert.equal(items.length, 2);
  assert.equal(items[0].comment, "fix");
  assert.equal(items[0].saved, true);
  assert.equal(items[1].saved, false);
  assert.deepEqual(items[0].rect, { x: 1, y: 2, width: 30, height: 10 });
  assert.equal(stateItems(null).length, 0);
  assert.equal(stateItems({}).length, 0);
});

test("the injected page script never reads open-state from inline styles", () => {
  // Owner 2026-09-18: inline display starts as "", so `!== "none"` is true
  // before anything ever opened — hover dead until the first pick, hover
  // stuck on after it. Open/closed travels in flags, asserted here so the
  // pattern cannot creep back in.
  const src = readFileSync(
    new URL("../../../../desktop-shell/src/annotate.js", import.meta.url),
    "utf8",
  );
  assert.ok(!/\.style\.display !==/.test(src), "state travels in flags, not inline styles");
  for (const flag of ["cardOpen", "menuOpen"]) assert.ok(src.includes(flag), flag);
});

test("the batch is one context: a head line plus numbered pins", () => {
  assert.equal(
    batchMessage({ url: "https://x.test/", items: [
      { n: 1, selector: "h1", comment: "Fix this" },
      { n: 2, selector: "p.lede", comment: "" },
    ] }),
    "2 annotations on https://x.test/\n1. Fix this\n2. p.lede",
  );
  assert.equal(batchMessage({ url: "", items: [{ n: 1, selector: "", comment: "" }] }), "1 annotation\n1. element");
  assert.equal(batchMessage({}), "0 annotations");
});

test("the injected page script shields its own keys from page hotkeys", () => {
  // Owner 2026-09-18: typing "s" in the card opened GitHub's search over it.
  // Outside the shadow root the event retargets to the host — a div, not a
  // form field — so the page's "ignore while typing" check fails and its
  // hotkey fires. The shield stops propagation at the shadow root, after the
  // input took the key. Reproduced against @github/hotkey in a real browser.
  const src = readFileSync(
    new URL("../../../../desktop-shell/src/annotate.js", import.meta.url),
    "utf8",
  );
  assert.ok(src.includes("const SHIELD_EVENTS = ["), "the shield list is gone");
  for (const ev of ["\"keydown\"", "\"keypress\"", "\"keyup\""]) {
    assert.ok(src.includes(ev), `the shield lost ${ev}`);
  }
  assert.ok(
    src.includes("root.addEventListener(type, (e) => e.stopPropagation(), false)"),
    "the shield must stop propagation at the shadow root",
  );
  // Enter-to-save cannot ask "is the target the card's input?": a document
  // listener sees the target retargeted to the host. The open card is the
  // answer (this read as "Enter does nothing" until 2026-09-18).
  assert.ok(!src.includes("e.target === input"), "the retargeted target decides nothing");
  assert.ok(src.includes('if (e.key === "Enter" && cardOpen)'), "Enter rides the card flag");
});

test("page events unwrap to {id, inner} at any encoding depth", () => {
  const state = { kind: "state", count: 1, items: [] };
  // the shell's envelope as text, the page's object as text: the normal path
  const single = parseAnnotMessage(JSON.stringify({ id: "w:2", raw: JSON.stringify(state) }));
  assert.equal(single.id, "w:2");
  assert.equal(single.inner.kind, "state");
  // the page's JSON text JSON-encoded a second time (a quoted string):
  // unwrap once more instead of dropping the sync silently
  const doubly = JSON.stringify({ id: "w:2", raw: JSON.stringify(JSON.stringify(state)) });
  assert.equal(parseAnnotMessage(doubly).inner.kind, "state");
  // already objects all the way down
  const plain = parseAnnotMessage({ id: "w:2", raw: state });
  assert.equal(plain.inner.count, 1);
  // junk in, null out, never a throw
  assert.equal(parseAnnotMessage(""), null);
  assert.equal(parseAnnotMessage("{bad"), null);
  assert.equal(parseAnnotMessage(null), null);
  assert.equal(parseAnnotMessage(42), null);
  assert.equal(parseAnnotMessage({ id: "w:2" }), null);
  assert.equal(parseAnnotMessage({ id: "w:2", raw: 42 }), null);
  assert.equal(parseAnnotMessage(JSON.stringify({ id: "w:2", raw: "{bad" })), null);
});

test("Send delivers to the bound session, never to a stranger", () => {
  const terms = [
    { id: "a", name: "aaa", running: false },
    { id: "desktop", name: "desktop", running: true },
    { id: "other", name: "other", running: true },
  ];
  // term-bound split: its own terminal, even when it is not first
  assert.deepEqual(
    resolveSendTarget({ boundSession: "t:desktop", terminals: terms }),
    { kind: "terminal", id: "desktop", name: "desktop" },
  );
  // term-bound but stopped: named reason, NO fallback to "other"
  const stopped = resolveSendTarget({ boundSession: "t:a", terminals: terms });
  assert.equal(stopped.none, true);
  assert.ok(stopped.reason.includes("aaa"));
  // term-bound but gone: named reason, NO fallback
  assert.equal(resolveSendTarget({ boundSession: "t:gone", terminals: terms }).none, true);
  // agent-bound with a running same-id terminal: agent door + terminal row
  assert.deepEqual(
    resolveSendTarget({ boundSession: "desktop", terminals: terms }),
    { kind: "agent", agentId: "desktop", terminalId: "desktop", name: "desktop" },
  );
  // agent-bound with none: named reason, NO fallback
  assert.equal(resolveSendTarget({ boundSession: "ghost", terminals: terms }).none, true);
  // standalone tab: first running, as before
  assert.deepEqual(
    resolveSendTarget({ boundSession: "", terminals: terms }),
    { kind: "terminal", id: "desktop", name: "desktop" },
  );
  // standalone, nothing running
  const empty = resolveSendTarget({ boundSession: "", terminals: [{ id: "a", running: false }] });
  assert.equal(empty.none, true);
  assert.ok(empty.reason.includes("Nothing to send to"));
  assert.equal(resolveSendTarget({}).none, true);
});

test("the pull door's answer is parsed, however it arrives", () => {
  const state = { kind: "state", url: "https://x.test/", count: 2, total: 2, items: [{ n: 1, rect: { x: 0, y: 0, width: 10, height: 10 }, comment: "a" }] };
  // the shell decodes one layer already: the payload arrives as JSON text
  assert.equal(parseStatePayload(JSON.stringify(state)).count, 2);
  // a doubly-encoded answer (ExecuteScript wraps a string result once more)
  assert.equal(parseStatePayload(JSON.stringify(JSON.stringify(state))).count, 2);
  // an object, for a caller that already decoded it
  assert.equal(parseStatePayload(state).count, 2);
  // an unarmed document answers "" — and junk never becomes a state
  assert.equal(parseStatePayload(""), null);
  assert.equal(parseStatePayload('""'), null);
  assert.equal(parseStatePayload("null"), null);
  assert.equal(parseStatePayload("{bad"), null);
  assert.equal(parseStatePayload(JSON.stringify({ kind: "pick" })), null);
  // the page's own mode being off is an answer, not junk
  assert.equal(parseStatePayload(JSON.stringify({ kind: "off" })).kind, "off");
  assert.equal(parseStatePayload(undefined), null);
});

test("the crop asks the shell for the preview with the native tab id", () => {
  // The annotation crop came back empty for every Send until this call site
  // stopped passing the React tab id ("w:3") to a command that only resolved
  // the native one ("3") — the failure was a silent `.catch` (2026-09-19).
  const surface = readFileSync(
    new URL("../components/WebTab.jsx", import.meta.url),
    "utf8",
  );
  assert.ok(
    surface.includes('invoke("btab_preview", { id })'),
    "the preview takes the native tab id",
  );
  assert.ok(
    !surface.includes('invoke("btab_preview", { id: tabId })'),
    "the React tab id belongs to the annotate commands, not to the preview",
  );
});
