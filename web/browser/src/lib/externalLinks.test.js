import test from "node:test";
import assert from "node:assert/strict";
import { OPEN_LINK_EVENT, shouldOpenExternally, installShellExternalLinks } from "./externalLinks.js";

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
    dispatched: [],
    dispatchEvent: (event) => { window.dispatched.push(event); },
    CustomEvent,
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

test("an external anchor click hands the URL to the app's browser tab", () => {
  const { window, document, calls } = stubShell();
  const uninstall = installShellExternalLinks({ window, document });
  assert.equal(typeof uninstall, "function");
  const target = { closest: (sel) => (sel === "a[href]" ? anchorStub("https://cfpperche.github.io/picode/") : null) };
  const event = document.fireClick(target);
  assert.equal(event.defaultPrevented, true);
  // The handoff is the app's own open-link event: the work browser tab opens
  // it, the system browser is no longer the destination.
  assert.deepEqual(calls, []);
  assert.equal(window.dispatched.length, 1);
  assert.equal(window.dispatched[0].type, OPEN_LINK_EVENT);
  assert.equal(window.dispatched[0].detail, "https://cfpperche.github.io/picode/");
  uninstall();
});

test('an external anchor without target="_blank" is caught too', () => {
  const { window, document, calls } = stubShell();
  installShellExternalLinks({ window, document });
  // In the shell this anchor used to navigate the webview in place — the
  // same dead end as a _blank click.
  const target = { closest: (sel) => (sel === "a[href]" ? anchorStub("https://docs.example/guide") : null) };
  const event = document.fireClick(target);
  assert.equal(event.defaultPrevented, true);
  assert.equal(window.dispatched.at(-1).detail, "https://docs.example/guide");
  assert.deepEqual(calls, []);
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
  assert.deepEqual(calls, []);
  assert.deepEqual(window.dispatched.map((e) => e.detail), [
    "https://docs.example/guide",
    "https://docs.example/guide",
  ]);
  // In-app targets keep the native contract (OAuth popups, same-tab).
  assert.deepEqual(window.open("https://localhost:8445/#/x", "_blank"), ["native", "https://localhost:8445/#/x", "_blank", undefined]);
  assert.deepEqual(window.open("https://auth.example/login", "picode-mcp-auth"), ["native", "https://auth.example/login", "picode-mcp-auth", undefined]);
  assert.deepEqual(window.open("https://docs.example/guide", "_self"), ["native", "https://docs.example/guide", "_self", undefined]);
  assert.deepEqual(calls, []);
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

test("a listener that never answers never breaks the click", () => {
  const { window, document } = stubShell();
  window.dispatchEvent = () => { throw new Error("no listener yet"); };
  installShellExternalLinks({ window, document });
  const target = { closest: () => anchorStub("https://docs.example/") };
  assert.doesNotThrow(() => document.fireClick(target));
  assert.doesNotThrow(() => window.open("https://docs.example/", "_blank"));
});
