import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveShell, shellURL, pickShell, setShell } from "./shell.js";

const cases = [
  ["explicit desktop survives narrow viewport and competing choices", { pathname: "/desktop/", search: "?mobile=1", saved: "mobile", narrow: true }, "desktop"],
  ["explicit mobile survives wide viewport and competing choices", { pathname: "/mobile/", search: "?desktop=1", saved: "desktop" }, "mobile"],
  ["explicit path without trailing slash", { pathname: "/mobile" }, "mobile"],
  ["legacy desktop query", { search: "?desktop=1", saved: "mobile", narrow: true }, "desktop"],
  ["legacy mobile query", { search: "?mobile=1", saved: "desktop" }, "mobile"],
  ["both legacy queries retain desktop precedence", { search: "?mobile=1&desktop=1" }, "desktop"],
  ["saved desktop", { saved: "desktop", narrow: true }, "desktop"],
  ["saved mobile", { saved: "mobile" }, "mobile"],
  ["system narrow", { narrow: true }, "mobile"],
  ["system wide", {}, "desktop"],
  ["invalid preference falls back to viewport", { saved: "unknown", narrow: true }, "mobile"],
  ["lookalike path does not select an app", { pathname: "/mobile-extra" }, "desktop"],
];
for (const [name, input, expected] of cases) test(name, () => assert.equal(resolveShell(input), expected));

for (const app of ["mobile", "desktop", "system"]) {
  test(`switch to ${app} preserves the deep link and unrelated query`, () => {
    assert.equal(shellURL("https://picode.test/desktop/?mobile=1&desktop=1&token=a%2Fb#/agent/a%2Fb", app),
      (app === "system" ? "/" : `/${app}/`) + "?token=a%2Fb#/agent/a%2Fb");
  });
}

test("blocked storage does not prevent initial selection or switching", () => {
  let assigned;
  const previous = Object.fromEntries(["location", "localStorage", "window"].map(key => [key, Object.getOwnPropertyDescriptor(globalThis, key)]));
  Object.defineProperties(globalThis, {
    localStorage: { configurable: true, value: { getItem() { throw Error("denied"); }, setItem() { throw Error("denied"); } } },
    location: { configurable: true, value: { href: "https://picode.test/?mobile=1#/agent/a", pathname: "/", search: "?mobile=1", assign: url => { assigned = url; } } },
    window: { configurable: true, value: { matchMedia: () => ({ matches: false }) } },
  });
  try {
    assert.equal(pickShell(), "mobile");
    setShell("desktop");
    assert.equal(assigned, "/desktop/#/agent/a");
  } finally {
    for (const [key, descriptor] of Object.entries(previous)) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor);
      else delete globalThis[key];
    }
  }
});
