import test from "node:test";
import assert from "node:assert/strict";
import { installHashGuard, registerHashGuard } from "./hashGuard.js";

test("dirty hash guard runs before route observers and releases clean navigation", () => {
  const target = new EventTarget();
  const stop = installHashGuard(target);
  let routes = 0, prompts = 0;
  target.addEventListener("hashchange", () => routes++);
  const clear = registerHashGuard((event) => { prompts++; event.stopImmediatePropagation(); });
  target.dispatchEvent(new Event("hashchange"));
  assert.equal(prompts, 1); assert.equal(routes, 0);
  clear();
  target.dispatchEvent(new Event("hashchange"));
  assert.equal(routes, 1);
  stop();
});

test("cleanup of an old editor does not unregister the new editor's guard", () => {
  const target = new EventTarget();
  const stop = installHashGuard(target);
  let prompts = 0;
  const old = registerHashGuard(() => assert.fail("stale guard"));
  const current = registerHashGuard(() => prompts++);
  old(); target.dispatchEvent(new Event("hashchange"));
  assert.equal(prompts, 1);
  current(); stop();
});
