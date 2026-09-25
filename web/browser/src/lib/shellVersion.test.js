import assert from "node:assert/strict";
import { test } from "node:test";

import { shellInfo, shellSupports } from "./shellVersion.js";

const tauri = { core: { invoke: () => Promise.resolve() } };

test("outside the shell there is no shell and nothing is supported", () => {
  assert.equal(shellInfo({}), null);
  assert.equal(shellSupports(0, {}), false);
});

test("a shell from before the handshake is protocol 0", () => {
  assert.deepEqual(shellInfo({ __TAURI__: tauri }), { version: "", protocol: 0 });
  assert.equal(shellSupports(0, { __TAURI__: tauri }), true);
  assert.equal(shellSupports(1, { __TAURI__: tauri }), false);
});

test("an announcing shell is read as announced", () => {
  const win = { __TAURI__: tauri, __PICODE_SHELL__: { version: "0.1.0", protocol: 1 } };
  assert.deepEqual(shellInfo(win), { version: "0.1.0", protocol: 1 });
  assert.equal(shellSupports(1, win), true);
  assert.equal(shellSupports(2, win), false);
});

test("a malformed announcement is treated as the oldest shell", () => {
  for (const s of [{ protocol: "2" }, { protocol: -1 }, { protocol: 1.5 }, "x"]) {
    assert.equal(shellInfo({ __TAURI__: tauri, __PICODE_SHELL__: s }).protocol, 0);
  }
});
