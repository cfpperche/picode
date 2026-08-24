import { toast } from "./toast.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("toast dispatches picode-toast", () => {
  const w = new EventTarget();
  globalThis.window = w;
  const seen = [];
  const on = (e) => seen.push(e.detail);
  w.addEventListener("picode-toast", on);
  toast("hello", "ok");
  w.removeEventListener("picode-toast", on);
  assert.equal(seen.length, 1);
  assert.equal(seen[0].message, "hello");
  assert.equal(seen[0].kind, "ok");
});
