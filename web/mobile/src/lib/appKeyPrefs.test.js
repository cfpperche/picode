import { test } from "node:test";
import assert from "node:assert/strict";
import { readAppKeyOverrides, persistAppKeyOverride } from "./appKeyPrefs.js";

function mockStorage(seed) {
  const mem = { ...seed };
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
}

test("appKeyOverrides round-trip", () => {
  mockStorage();
  assert.deepEqual(readAppKeyOverrides(), {});
  persistAppKeyOverride("app.palette.toggle", ["alt+p"]);
  assert.deepEqual(readAppKeyOverrides(), { "app.palette.toggle": ["alt+p"] });
});

test("appKeyOverrides keeps other actions when writing one", () => {
  mockStorage({ "picode-app-keys": JSON.stringify({ a: ["x"] }) });
  persistAppKeyOverride("b", ["y"]);
  assert.deepEqual(readAppKeyOverrides(), { a: ["x"], b: ["y"] });
});

test("appKeyOverrides: keys === null deletes the override, reverting to defaults", () => {
  mockStorage({ "picode-app-keys": JSON.stringify({ a: ["x"], b: ["y"] }) });
  persistAppKeyOverride("a", null);
  assert.deepEqual(readAppKeyOverrides(), { b: ["y"] });
});

test("appKeyOverrides tolerates corrupt storage", () => {
  mockStorage({ "picode-app-keys": "not json" });
  assert.deepEqual(readAppKeyOverrides(), {});
});
