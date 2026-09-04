// Apps host primitives (ADR-0036): pure normalizers between the server's
// JSON and the renderer. The renderer only ever sees clean trees — junk,
// unknown block types and unknown field methods are dropped here, and a
// whole view is refused when its apiVersion isn't ours.

export const SUPPORTED_API = 1;

const BLOCK_TYPES = new Set(["list", "detail", "form", "actions"]);
const FIELD_METHODS = new Set(["select", "confirm", "input", "editor"]);
const LAYOUTS = new Set(["", "split"]);
const PANES = new Set(["", "list", "detail"]);
const TONES = new Set(["", "info", "ok", "warn", "danger"]);

function str(v) {
  return typeof v === "string" ? v : "";
}

function strList(v) {
  return Array.isArray(v) ? v.filter((x) => typeof x === "string" && x) : [];
}

export function normalizeManifests(payload) {
  const list = Array.isArray(payload?.apps) ? payload.apps : [];
  const out = [];
  for (const m of list) {
    if (!m || typeof m.id !== "string" || !m.id || typeof m.name !== "string" || !m.name) continue;
    out.push({
      id: m.id,
      name: m.name,
      icon: typeof m.icon === "string" ? m.icon : "",
      apiVersion: Number(m.apiVersion) || 0,
      badge: {
        count: Number(m.badge?.count) || 0,
        dot: !!m.badge?.dot,
      },
    });
  }
  return out;
}

export function supportedApp(manifest) {
  return !!manifest && manifest.apiVersion === SUPPORTED_API;
}

function normalizeAction(a) {
  if (!a || typeof a.id !== "string" || !a.id || typeof a.label !== "string" || !a.label) return null;
  return {
    id: a.id,
    label: a.label,
    icon: str(a.icon),
    primary: !!a.primary,
    confirm: str(a.confirm),
    danger: !!a.danger,
    args: a.args && typeof a.args === "object" ? a.args : {},
  };
}

function normalizeActions(list) {
  return (Array.isArray(list) ? list : []).map(normalizeAction).filter(Boolean);
}

const TABS_MAX = 8; // a sane cap — the header strip isn't a menu

function normalizeTab(t) {
  if (!t || typeof t.id !== "string" || !t.id || typeof t.label !== "string" || !t.label) return null;
  return { id: t.id, label: t.label, path: str(t.path), badge: str(t.badge) };
}

function normalizeTabs(list) {
  return (Array.isArray(list) ? list : []).map(normalizeTab).filter(Boolean).slice(0, TABS_MAX);
}

// head carries the fields every block may decorate itself with. Anything
// not listed here is dropped before the renderer sees it — the reason a
// new Go field needs a matching edit in this file.
function blockHead(b) {
  return {
    title: str(b.title),
    meta: strList(b.meta),
    at: str(b.at),
    pane: PANES.has(b.pane) ? str(b.pane) : "",
  };
}

function normalizeBlock(b) {
  if (!b || !BLOCK_TYPES.has(b.type)) return null;
  const head = blockHead(b);
  if (b.type === "list") {
    const items = (Array.isArray(b.items) ? b.items : [])
      .filter((it) => it && typeof it.id === "string" && it.id && typeof it.title === "string" && it.title)
      .map((it) => ({
        id: it.id,
        title: it.title,
        subtitle: str(it.subtitle),
        meta: strList(it.meta),
        at: str(it.at),
        icon: str(it.icon),
        badge: str(it.badge),
        tone: TONES.has(it.tone) ? str(it.tone) : "",
        unread: !!it.unread,
        path: str(it.path),
        actions: normalizeActions(it.actions),
      }));
    return { type: "list", ...head, items };
  }
  if (b.type === "detail") {
    if (typeof b.markdown !== "string" || !b.markdown) return null;
    return { type: "detail", ...head, markdown: b.markdown };
  }
  if (b.type === "form") {
    const f = b.form;
    if (!f || typeof f.id !== "string" || !f.id) return null;
    const fields = (Array.isArray(f.fields) ? f.fields : [])
      .filter((x) => x && typeof x.name === "string" && x.name && FIELD_METHODS.has(x.method))
      .map((x) => ({
        name: x.name,
        method: x.method,
        title: typeof x.title === "string" ? x.title : "",
        message: typeof x.message === "string" ? x.message : "",
        options: Array.isArray(x.options) ? x.options.filter((o) => typeof o === "string") : [],
        placeholder: typeof x.placeholder === "string" ? x.placeholder : "",
        prefill: typeof x.prefill === "string" ? x.prefill : "",
      }));
    return { type: "form", ...head, form: { id: f.id, submit: str(f.submit), burst: !!f.burst, fields } };
  }
  // actions
  const actions = normalizeActions(b.actions);
  return actions.length ? { type: "actions", ...head, actions } : null;
}

// normalizeView returns null when the view can't be rendered by this
// build (unsupported apiVersion or not an object) — the caller shows the
// "needs a newer PiCode" card instead.
export function normalizeView(view) {
  if (!view || typeof view !== "object") return null;
  if (Number(view.apiVersion) !== SUPPORTED_API) return null;
  const blocks = (Array.isArray(view.blocks) ? view.blocks : []).map(normalizeBlock).filter(Boolean);
  const layout = LAYOUTS.has(view.layout) ? str(view.layout) : "";
  return {
    apiVersion: SUPPORTED_API, title: str(view.title), layout,
    empty: str(view.empty), tabs: normalizeTabs(view.tabs), blocks,
  };
}

// aggregateBadge folds every app badge into the sidebar tab pill:
// numeric when anything actionable exists, dot when only activity does.
export function aggregateBadge(apps) {
  let count = 0;
  let dot = false;
  for (const a of Array.isArray(apps) ? apps : []) {
    count += a?.badge?.count || 0;
    dot = dot || !!a?.badge?.dot;
  }
  return { count, dot: dot && count === 0 };
}
