// Track C actuation helpers (ADR-0053). Pure logic + grant storage; the
// orchestration lives in sidepanel.js.

function originOf(url) {
  try {
    const u = new URL(url);
    return u.protocol + "//" + u.host;
  } catch (_) {
    return "";
  }
}

const GRANTS_KEY = "picode-act-grants";

async function loadGrants() {
  const got = await chrome.storage.local.get(GRANTS_KEY);
  return got[GRANTS_KEY] || {};
}

async function grantOrigin(origin) {
  const grants = await loadGrants();
  grants[origin] = true;
  await chrome.storage.local.set({ [GRANTS_KEY]: grants });
}

async function revokeOrigin(origin) {
  const grants = await loadGrants();
  delete grants[origin];
  await chrome.storage.local.set({ [GRANTS_KEY]: grants });
}

/**
 * actDriver runs INSIDE the page, one action per executeScript call.
 * init stashes the queue on window; each next call highlights, performs
 * one action and returns its outcome. Self-contained on purpose.
 */
function actDriver(cmd) {
  if (cmd && cmd.init) {
    window.__picodeAct = { queue: JSON.parse(JSON.stringify(cmd.init)), i: 0, outcomes: [] };
    const css = document.getElementById("picode-act-style");
    if (!css) {
      const s = document.createElement("style");
      s.id = "picode-act-style";
      s.textContent =
        "[data-picode-hl]{outline:2px solid #7c8cf8 !important;outline-offset:1px;" +
        "background:rgba(124,140,248,.16) !important;transition:background .15s}";
      document.documentElement.appendChild(s);
    }
  }
  const st = window.__picodeAct;
  if (!st) return { done: true, outcomes: [] };
  document.querySelectorAll("[data-picode-hl]").forEach((el) => el.removeAttribute("data-picode-hl"));
  if (st.i >= st.queue.length) {
    const outcomes = st.outcomes;
    window.__picodeAct = null;
    document.querySelectorAll("[data-picode-hl]").forEach((el) => el.removeAttribute("data-picode-hl"));
    return { done: true, outcomes };
  }
  const a = st.queue[st.i];
  const out = { act: a.act, selector: a.selector || "", ok: true };
  const el = a.selector ? document.querySelector(a.selector) : null;
  if (a.act !== "scroll" && !el) {
    out.ok = false;
    out.error = "no match";
  } else {
    try {
      if (el) el.setAttribute("data-picode-hl", "1");
      if (a.act === "click") {
        el.click();
      } else if (a.act === "fill") {
        const proto = el instanceof HTMLTextAreaElement
          ? HTMLTextAreaElement.prototype
          : HTMLInputElement.prototype;
        const setter = Object.getOwnPropertyDescriptor(proto, "value").set;
        setter.call(el, a.value || "");
        el.dispatchEvent(new Event("input", { bubbles: true }));
        el.dispatchEvent(new Event("change", { bubbles: true }));
      } else if (a.act === "press") {
        const key = a.key || "Enter";
        el.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
        el.dispatchEvent(new KeyboardEvent("keyup", { key, bubbles: true }));
      } else if (a.act === "read") {
        out.text = (el.innerText || "").slice(0, 2000);
      } else if (a.act === "scroll") {
        window.scrollTo({
          top: a.to === "bottom" ? document.body.scrollHeight : 0,
          behavior: "instant",
        });
      }
    } catch (e) {
      out.ok = false;
      out.error = String((e && e.message) || e);
    }
  }
  st.outcomes.push(out);
  st.i += 1;
  return { done: false, i: st.i, outcome: out };
}

window.PiCodeAct = { originOf, loadGrants, grantOrigin, revokeOrigin, actDriver };
