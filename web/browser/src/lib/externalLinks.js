// externalLinks.js — the shell's way out for "leave the app" links.
//
// The desktop shell (Tauri + WebView2, ADR-0122) has no second window and no
// opener plugin: a `target="_blank"` click or a `window.open` in the main
// webview fires a new-window request nobody answers, so every chrome link
// that leaves the app (Documentation, setup guides, changelog, Open in
// browser) is silently dead there. The escape hatch is the
// `btab_open_external` command, which hands the URL to the system default
// browser. This bridge routes exactly those clicks to it; in a real browser
// it installs nothing and native behavior is untouched.

export const EXTERNAL_COMMAND = "btab_open_external";

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
// handoff wins over menu-close bookkeeping; window.open keeps its contract
// for in-app targets (_self, named OAuth popups) and only reroutes the
// new-context http(s) opens the shell would otherwise swallow. Returns an
// uninstall function, or undefined when there is no shell to bridge.
export function installShellExternalLinks({ window, document } = {}) {
  const invoke = window?.__TAURI__?.core?.invoke;
  if (typeof invoke !== "function") return undefined;
  if (!document || typeof document.addEventListener !== "function") return undefined;
  const origin = window?.location?.origin;
  const handOff = (url) => {
    try {
      const done = invoke(EXTERNAL_COMMAND, { url });
      if (done && typeof done.catch === "function") done.catch(() => {});
    } catch {
      // The click stays a no-op, as before the bridge.
    }
  };
  const onClick = (event) => {
    const anchor = event?.target?.closest?.('a[target="_blank"]');
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
