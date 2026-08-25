import { test } from "node:test";
import assert from "node:assert/strict";
import { readRecents, pushRecent, removeRecent, clearRecents, rememberProviders } from "./providerRecents.js";

test("recents order and remove", () => {
  const mem = {};
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
  clearRecents();
  pushRecent("anthropic");
  pushRecent("xai");
  assert.deepEqual(readRecents(), ["xai", "anthropic"]);
  rememberProviders(["openai"]);
  assert.ok(readRecents().includes("openai"));
  removeRecent("xai");
  assert.deepEqual(readRecents().filter((id) => id !== "openai"), ["anthropic"]);
});
