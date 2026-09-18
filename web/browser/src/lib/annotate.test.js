import assert from "node:assert/strict";
import { test } from "node:test";

import {
  PICK_STYLES,
  cropRect,
  parsePick,
  pickLabel,
  pickScript,
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
