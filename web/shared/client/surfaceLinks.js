// surfaceLinks.js — where a link clicked inside an app surface may land.
//
// App surfaces (ADR-0036) render server-driven markdown, and markdown mints
// plain anchors. On 2026-09-20/21 clicking a github.com link in an Inbox
// feed item navigated the shell's whole document — the owner lost the shell
// UI. The host contract since: an external http(s) link inside an app
// surface never touches the hosting document. In the desktop shell it opens
// as a work-browser tab (the same flow the address bar uses); outside the
// shell it opens in a new browser tab, rel=noopener noreferrer. Same-origin
// links, relative paths, hash routes, and non-web schemes (mailto:,
// javascript:) keep native behavior — the same decision table as the
// shell's system-browser bridge (externalLinks.js), a different
// destination: that door leaves the app, this one stays inside it.

// surfaceLinkPlan is the pure decision both surfaces share. Returns null
// (native behavior) or the action: "tab" hands the URL to the shell's work
// browser, "new-tab" asks the host document for a _blank open. The URL
// comes back absolute, so callers never re-resolve it.
export function surfaceLinkPlan(href, { origin, shell = false } = {}) {
  if (typeof href !== "string") return null;
  const trimmed = href.trim();
  if (!trimmed) return null;
  let url;
  try {
    url = new URL(trimmed, origin || undefined);
  } catch {
    return null;
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") return null;
  if (origin && url.origin === origin) return null; // in-app route
  return shell ? { action: "tab", url: url.href } : { action: "new-tab", url: url.href };
}

// surfaceLinkClick is the one guard both surfaces hang on their root
// element (capture phase, so it rules before any block-internal handler and
// covers every renderer, present and future). It steps aside when another
// door already took the click: the shell's system-browser bridge answers
// explicit target=_blank anchors at document capture, which runs before any
// surface. Outside the shell — or with no work browser to call — the guard
// sets the new-tab door and lets the native default walk it.
export function surfaceLinkClick({ origin, shell = false, onOpenUrl } = {}) {
  return (event) => {
    if (event.defaultPrevented || event.button !== 0) return;
    const anchor = event.target && typeof event.target.closest === "function"
      ? event.target.closest("a[href]")
      : null;
    if (!anchor) return;
    const plan = surfaceLinkPlan(anchor.getAttribute("href"), { origin, shell });
    if (!plan) return;
    if (plan.action === "tab" && typeof onOpenUrl === "function") {
      event.preventDefault();
      onOpenUrl(plan.url);
      return;
    }
    anchor.target = "_blank";
    anchor.rel = "noopener noreferrer";
  };
}
