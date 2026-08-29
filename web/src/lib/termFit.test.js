import assert from "node:assert/strict";
import { test } from "node:test";
import { sendTermResize } from "./termFit.js";

test("sendTermResize no-ops until the socket is open and the term has a size", () => {
  assert.equal(sendTermResize(null), false);
  assert.equal(sendTermResize({}), false);
  const sent = [];
  const sock = { readyState: 0, send: (s) => sent.push(s) };
  assert.equal(sendTermResize({ term: { cols: 80, rows: 24 }, sock }), false);
  sock.readyState = 1;
  assert.equal(sendTermResize({ term: { cols: 1, rows: 24 }, sock }), false);
  assert.equal(sendTermResize({ term: { cols: 80, rows: 24 }, sock }), true);
  assert.equal(sent[0], JSON.stringify({ type: "resize", cols: 80, rows: 24 }));
});
