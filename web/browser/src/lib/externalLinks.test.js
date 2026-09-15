import test from "node:test";
import assert from "node:assert/strict";
import { EXTERNAL_COMMAND, shouldOpenExternally, installShellExternalLinks } from "./externalLinks.js";

// Decision table: href × origin → opens in the system browser?
// | href                              | origin              | external |
// |-----------------------------------|---------------------|----------|
// | https://docs.example/guide        | https://app:8445    | yes      |
// | http://docs.example/guide         | https://app:8445    | yes      |
// | https://app:8445/#/agents         | https://app:8445    | no       |
// | /guide/local                      | https://app:8445    | no       |
// | #/agents                          | https://app:8445    | no       |
// | mailto:hi@example.com             | https://app:8445    | no       |
// | javascript:alert(1)               | https://app:8445    | no       |
// | https://[invalid                  | https://app:8445    | no       |
// | (empty / null / undefined)        | https://app:8445    | no       |

const APP = "https://localhost:8445";

test("shouldOpenExternally routes absolute http(s) URLs on another origin", () => {
  assert.equal(shouldOpenExternally("https://cfpperche.github.io/picode/", APP), true);
  assert.equal(shouldOpenExternally("http://docs.example/guide", APP), true);
  assert.equal(shouldOpenExternally("  https://docs.example/guide  ", APP), true);
});

test("shouldOpenExternally keeps same-origin, relative and non-web hrefs", () => {
  assert.equal(shouldOpenExternally("https://localhost:8445/#/agents", APP), false);
  assert.equal(shouldOpenExternally("http://localhost:8445/x", "http://localhost:8445"), false);
  assert.equal(shouldOpenExternally("/guide/local", APP), false);
  assert.equal(shouldOpenExternally("#/agents", APP), false);
  assert.equal(shouldOpenExternally("mailto:hi@example.com", APP), false);
  assert.equal(shouldOpenExternally("javascript:alert(1)", APP), false);
  assert.equal(shouldOpenExternally("https://[invalid", APP), false);
  assert.equal(shouldOpenExternally("", APP), false);
  assert.equal(shouldOpenExternally(null, APP), false);
  assert.equal(shouldOpenExternally(undefined, APP), false);
});

function stubShell() {
  const listeners = new Map();
  const calls = [];
  const window = {
    location: { origin: APP, href: APP + "/desktop/" },
    __TAURI__: { core: { invoke: (cmd, args) => { calls.push([cmd, args]); return Promise.resolve(); } } },
    open: (...args) => ["native", ...args],
  };
  const document = {
    addEventListener: (type, fn, capture) => listeners.set(type + !!capture, fn),
    removeEventListener: (type, fn, capture) => { if (listeners.get(type + !!capture) === fn) listeners.delete(type + !!capture); },
    fireClick: (target) => {
      const fn = listeners.get("clicktrue");
      if (!fn) return null;
      const event = { target, defaultPrevented: false, preventDefault() { this.defaultPrevented = true; } };
      fn(event);
      return event;
    },
  };
  return { window, document, calls, listeners };
}

function anchorStub(href) {
  return { getAttribute: (name) => (name === "href" ? href : null), href };
}

test("outside the shell nothing is installed and native behavior is untouched", () => {
  const { window, document, listeners } = stubShell();
  delete window.__TAURI__;
  const nativeOpen = window.open;
  assert.equal(installShellExternalLinks({ window, document }), undefined);
  assert.equal(listeners.size, 0);
  assert.equal(window.open, nativeOpen);
});

test("a _blank docs click hands the URL to the system browser", async () => {
  const { window, document, calls } = stubShell();
  const uninstall = installShellExternalLinks({ window, document });
  assert.equal(typeof uninstall, "function");
  const target = { closest: (sel) => (sel === 'a[target="_blank"]' ? anchorStub("https://cfpperche.github.io/picode/") : null) };
  const event = document.fireClick(target);
  assert.equal(event.defaultPrevented, true);
  assert.deepEqual(calls, [[EXTERNAL_COMMAND, { url: "https://cfpperche.github.io/picode/" }]]);
  uninstall();
});

test("same-origin and non-anchor clicks pass through untouched", () => {
  const { window, document, calls } = stubShell();
  installShellExternalLinks({ window, document });
  const sameOrigin = { closest: () => anchorStub("https://localhost:8445/#/agents") };
  assert.equal(document.fireClick(sameOrigin).defaultPrevented, false);
  const mailto = { closest: () => anchorStub("mailto:hi@example.com") };
  assert.equal(document.fireClick(mailto).defaultPrevented, false);
  assert.equal(document.fireClick({ closest: () => null }).defaultPrevented, false);
  assert.equal(document.fireClick({}).defaultPrevented, false);
  assert.deepEqual(calls, []);
});

test("window.open reroutes _blank http(s) and keeps in-app targets", () => {
  const { window, document, calls } = stubShell();
  installShellExternalLinks({ window, document });
  assert.equal(window.open("https://docs.example/guide", "_blank", "noopener"), null);
  assert.equal(window.open("https://docs.example/guide"), null);
  assert.deepEqual(calls, [
    [EXTERNAL_COMMAND, { url: "https://docs.example/guide" }],
    [EXTERNAL_COMMAND, { url: "https://docs.example/guide" }],
  ]);
  // In-app targets keep the native contract (OAuth popups, same-tab).
  assert.deepEqual(window.open("https://localhost:8445/#/x", "_blank"), ["native", "https://localhost:8445/#/x", "_blank", undefined]);
  assert.deepEqual(window.open("https://auth.example/login", "picode-mcp-auth"), ["native", "https://auth.example/login", "picode-mcp-auth", undefined]);
  assert.deepEqual(window.open("https://docs.example/guide", "_self"), ["native", "https://docs.example/guide", "_self", undefined]);
  assert.equal(calls.length, 2);
});

test("uninstall restores the document listener and window.open", () => {
  const { window, document, calls, listeners } = stubShell();
  const nativeOpen = window.open;
  const uninstall = installShellExternalLinks({ window, document });
  uninstall();
  assert.equal(listeners.size, 0);
  assert.equal(window.open, nativeOpen);
  assert.equal(document.fireClick({ closest: () => anchorStub("https://docs.example/") }), null);
  window.open("https://docs.example/", "_blank");
  assert.deepEqual(calls, []);
});

test("a failing invoke never breaks the click", () => {
  const { window, document } = stubShell();
  window.__TAURI__.core.invoke = () => Promise.reject(new Error("gone"));
  installShellExternalLinks({ window, document });
  const target = { closest: () => anchorStub("https://docs.example/") };
  assert.doesNotThrow(() => document.fireClick(target));
  assert.doesNotThrow(() => window.open("https://docs.example/", "_blank"));
});
