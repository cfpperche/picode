// externalLinks.js — the shell's door for "leave the app" links.
//
// The desktop shell (Tauri + WebView2, ADR-0122) has no second window and no
// opener plugin: a `target="_blank"` click or a `window.open` in the main
// webview fires a new-window request nobody answers, so every chrome link
// that leaves the app (Documentation, setup guides, changelog, Open in
// browser) is silently dead there. Since 2026-09-21 (owner) those links do
// not leave the app at all: this bridge hands the URL to the app itself
// (OPEN_LINK_EVENT), which opens it in a PiCode browser tab. In a real
// browser it installs nothing and native behavior is untouched.

export const OPEN_LINK_EVENT = "picode-open-link";

// shouldOpenExternally decides whether href means "leave the app": an
// absolute http(s) URL on another origin. Same-origin links (in-app routes),
// relative links, and non-web schemes (mailto:, javascript:) stay put —
// decision table in the test file.
export function shouldOpenExternally(href, origin) {
  if (typeof href !== "string") return false;
  const trimmed = href.trim();
  if (!/^https?:\/\//i.test(trimmed)) return false;
  let url;
  try {
    url = new URL(trimmed);
  } catch {
    return false;
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") return false;
  if (origin && url.origin === origin) return false;
  return true;
}

// installShellExternalLinks wires the bridge when (and only when) the page
// runs inside the shell — the presence of the Tauri invoke is the capability
// check, so both entries (/browser/ plain, /desktop/ chromed) share it and
// only the shell pays. Anchor clicks are caught in the capture phase so the
// handoff wins over menu-close bookkeeping — and on any external anchor,
// with or without target="_blank": an anchor that navigates the webview in
// place is the same dead end. window.open keeps its contract for in-app
// targets (_self, named OAuth popups) and only reroutes the new-context
// http(s) opens the shell would otherwise swallow. Returns an uninstall
// function, or undefined when there is no shell to bridge.
export function installShellExternalLinks({ window, document } = {}) {
  const invoke = window?.__TAURI__?.core?.invoke;
  if (typeof invoke !== "function") return undefined;
  if (!document || typeof document.addEventListener !== "function") return undefined;
  const origin = window?.location?.origin;
  const handOff = (url) => {
    // No invoke here on purpose: the destination is the app's own browser
    // tab, and the listener that opens it lives in App.jsx.
    try {
      window.dispatchEvent(new CustomEvent(OPEN_LINK_EVENT, { detail: url }));
    } catch {
      // A window that cannot answer yet must not break the click.
    }
  };
  const onClick = (event) => {
    const anchor = event?.target?.closest?.("a[href]");
    if (!anchor) return;
    const href = typeof anchor.getAttribute === "function" ? anchor.getAttribute("href") : anchor.href;
    if (!shouldOpenExternally(href, origin)) return;
    event.preventDefault();
    handOff(href.trim());
  };
  document.addEventListener("click", onClick, true);

  const originalOpen = window.open;
  if (typeof originalOpen === "function") {
    window.open = (url, target, features) => {
      if ((target === undefined || target === "" || target === "_blank") && shouldOpenExternally(url, origin)) {
        handOff(String(url).trim());
        return null;
      }
      return originalOpen.call(window, url, target, features);
    };
  }

  return () => {
    document.removeEventListener("click", onClick, true);
    if (typeof originalOpen === "function") window.open = originalOpen;
  };
}
