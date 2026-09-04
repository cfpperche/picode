import { useCallback, useEffect, useId, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, humanizeError } from "../lib/api.js";
import { normalizeView, supportedApp, SUPPORTED_API } from "../lib/appPrimitives.js";
import { useKeptScroll } from "../lib/keepScroll.js";
import { relTime, absTime } from "../lib/relTime.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import { filterListBlocks, countListItems } from "../lib/appSearch.js";
import AppIcon from "./AppIcon.jsx";
import { IconChevronLeft, IconCheck, IconClock, IconInbox, IconTrash } from "./Icons.jsx";
import { subscribeFeed } from "../lib/feed.js";
import { touches } from "../lib/feedReducers.js";

const SKELETON_ROWS = 5;
// A hidden tab keeps its view; revealing it refetches only when the last read
// is old enough to have missed something. Flipping between two tabs must not
// re-ask the app on every switch.
const REVEAL_STALE_MS = 10_000;

// List-pane width for the split layout. Global, not per-app (same choice as
// FileTreeSurface's TREE_KEY): it's a host preference, not app content, so a
// second split app inherits the width the reader already tuned.
const LIST_MIN = 300;
const LIST_MAX = 640;
const LIST_KEY = "picode-app-split-w";

// One open app (ADR-0036). The app answers with a primitive tree; this
// surface renders it with host components — chrome (header, split,
// selection, timestamps) stays host-owned, and a tree this build can't
// speak is refused, never guessed at.
// paneMode (ADR-0044): the phone renders a split app as two screens —
// "list" (rows + tabs + search; an item row calls onOpenItem instead of
// selecting) and "detail" (one item's panes under the shell's own Back
// header; an action that returns to the root calls onClose). Undefined
// keeps the desktop's split. onGoto receives an action's goto directive
// ("agent:<id>"); each shell opens its
// own agent terminal surface.
export default function AppSurface({ appId, hidden, manifest, onClose, initialPath, refreshKey, paneMode, onOpenItem, onGoto }) {
  // Native radio `name` grouping is document-wide, not component-scoped —
  // without a per-mount id, a second open app (or the same app reopened)
  // would fight this one over which segment shows checked.
  const tabsName = useId();
  // initialPath (ADR-0044): a deep link — the phone's #/inbox/<id> — lands
  // on that item instead of the list. Later changes to it navigate too.
  const [path, setPath] = useState(initialPath || "");
  useEffect(() => { if (initialPath != null) setPath(initialPath); }, [initialPath]);
  const [view, setView] = useState(null); // normalized tree
  const [unsupported, setUnsupported] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [listW, setListW] = useState(() => {
    const n = parseInt(localStorage.getItem(LIST_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(LIST_MAX, Math.max(LIST_MIN, n)) : 380;
  });
  const [resizing, setResizing] = useState(false);
  const [stacked, setStacked] = useState(() => !window.matchMedia("(min-width: 881px)").matches);
  // Latest-wins, never skip: a click can navigate while a focus-triggered
  // refresh is in flight — dropping that load would eat the navigation.
  const seqRef = useRef(0);
  const detailRef = useRef(null);
  // Leaving the tab must not close the item the reader opened: the surface
  // stays mounted, so `path` and the loaded view survive the switch.
  const rootRef = useKeptScroll(hidden, [".app-body", ".app-pane-list"]);
  const loadRef = useRef(() => {});
  const pathRef = useRef("");
  // Mounting counts as a read: the effect below loads immediately, and the
  // reveal check must not fire a second load on top of it.
  const lastLoadRef = useRef(Date.now());

  const load = useCallback(async (p) => {
    const seq = ++seqRef.current;
    setBusy(true);
    try {
      const raw = await api("/api/apps/" + encodeURIComponent(appId) + "/view" + (p ? "?path=" + encodeURIComponent(p) : ""));
      if (seq !== seqRef.current) return; // a newer load superseded this one
      const v = normalizeView(raw);
      if (v) { setView(v); setUnsupported(false); setError(""); }
      else { setView(null); setUnsupported(true); setError(""); }
    } catch (e) {
      if (seq === seqRef.current) setError(humanizeError(e.message || String(e)));
    } finally {
      if (seq === seqRef.current) {
        setBusy(false);
        lastLoadRef.current = Date.now();
      }
    }
  }, [appId]);

  useEffect(() => { load(path); }, [path, load]);
  // refreshKey (ADR-0044 phase 3): the phone's pull-to-refresh bumps it.
  useEffect(() => { if (refreshKey) loadRef.current(pathRef.current); }, [refreshKey]);
  loadRef.current = load;
  pathRef.current = path;
  useEffect(() => {
    // Stacked (narrow) split: the detail sits below the list, so a pick
    // has to bring it into view — otherwise selecting looks like nothing.
    if (!path || !detailRef.current) return;
    if (window.matchMedia("(min-width: 881px)").matches) return;
    detailRef.current.scrollIntoView({ block: "start", behavior: "smooth" });
  }, [path]);
  useEffect(() => {
    const mql = window.matchMedia("(min-width: 881px)");
    const sync = () => setStacked(!mql.matches);
    sync();
    mql.addEventListener("change", sync);
    return () => mql.removeEventListener("change", sync);
  }, []);
  useEffect(() => {
    // Change feed (ADR-0048): the app's own entity changed → reload now,
    // even on a hidden tab (cheap, and the reveal then shows the truth).
    return subscribeFeed((ev) => {
      if (ev.type === "feed.reset" || ev.type === "feed.open" || touches(ev, [appId])) load(path);
    });
  }, [load, path, appId]);
  useEffect(() => {
    // Like the file tree: refresh when the page comes back, no polling. Apps on
    // hidden tabs sit this out — every open tab would re-ask. Their reveal is
    // the refresh.
    const onVisible = () => { if (!document.hidden && !hidden) load(path); };
    window.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    return () => {
      window.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
    };
  }, [load, path, hidden]);
  // Reappearing is the other "back to look at it" moment — the window never
  // lost focus, so nothing above fires.
  useEffect(() => {
    if (hidden) return;
    if (Date.now() - lastLoadRef.current > REVEAL_STALE_MS) loadRef.current(pathRef.current);
  }, [hidden]);

  function onSizerDown(e) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = listW;
    let latest = startW;
    setResizing(true);
    const move = (ev) => {
      latest = Math.min(LIST_MAX, Math.max(LIST_MIN, Math.round(startW + (ev.clientX - startX))));
      setListW(latest);
    };
    const up = () => {
      setResizing(false);
      try { localStorage.setItem(LIST_KEY, String(latest)); } catch { /* ignore */ }
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  async function fire(action, args) {
    if (action.confirm) {
      const ok = await askConfirm({ title: action.label, message: action.confirm, confirmLabel: action.label, danger: !!action.danger });
      if (!ok) return;
    }
    try {
      const res = await api("/api/apps/" + encodeURIComponent(appId) + "/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: action.id, path, args: { ...(action.args || {}), ...(args || {}) } }),
      });
      if (res && res.toast) toast(res.toast, "ok");
      // A goto directive outranks every in-app outcome: the action asks
      // the shell to leave the app (Open terminal → the agent's tab).
      if (res && res.goto && onGoto) {
        onGoto(String(res.goto));
        return;
      }
      // A detail screen whose action sends it back to the root is done:
      // the phone pops the screen instead of rendering the root here —
      // whether the app answered with a path, a view, or both.
      // ActionResult.path is omitempty: the root comes back as a view
      // with no path at all, so "went somewhere and it is not an item" is
      // the test, not "path is a string".
      if (paneMode === "detail" && onClose && res && (res.view || typeof res.path === "string") && !String(res.path || "").startsWith("item/")) { onClose(); return; }
      if (res && res.view) {
        const v = normalizeView(res.view);
        if (v) { setView(v); setUnsupported(false); }
        else setUnsupported(true);
      } else if (res && typeof res.path === "string" && res.path !== path) {
        setPath(res.path);
      }
    } catch (e) {
      toastError(e);
    }
  }

  const title = (manifest && manifest.name) || appId;
  const badVersion = manifest && !supportedApp(manifest);
  const split = !!view && view.layout === "split";
  const listBlocks = split ? view.blocks.filter((b) => b.pane === "list") : [];
  const detailBlocks = split ? view.blocks.filter((b) => b.pane !== "list") : [];
  const navigate = (p) => {
    if (paneMode === "list" && onOpenItem && typeof p === "string" && p.startsWith("item/")) { onOpenItem(p); return; }
    setPath(p);
  };
  const ctx = { onNavigate: navigate, onAction: fire, selected: path };
  // The detail header repeats the selected row's kind lozenge so the two
  // panes agree — read off the list the app already sent, no new field.
  // A list-pane block isn't always a list block (e.g. a bulk-action row
  // like Inbox's "Clear all done") — guard b.items so a non-list block
  // there never breaks this lookup.
  const selectedRow = split
    ? listBlocks.flatMap((b) => b.items || []).find((it) => it.path && it.path === path)
    : null;
  // Search (ADR-0036 amendment): host-generic, filters rather than dims
  // (unlike git graph's ADR-0038 — a list has no positional layout to
  // protect). totalItems must stay unfiltered: it decides whether the box
  // itself renders, and a filtered count would hide the only way to clear
  // an exhausted query.
  const bodyBlocks = split ? listBlocks : (view ? view.blocks : []);
  const totalItems = countListItems(bodyBlocks);
  const hasQuery = query.trim().length > 0;
  const filteredBodyBlocks = hasQuery ? filterListBlocks(bodyBlocks, query) : bodyBlocks;
  const noMatches = hasQuery && totalItems > 0 && countListItems(filteredBodyBlocks) === 0;

  if (paneMode === "detail") {
    return (
      <section className="app-surface app-surface-detail" aria-label={title} hidden={!!hidden} ref={rootRef}>
        {unsupported || badVersion ? (
          <p className="ft-msg">This app needs a newer PiCode.</p>
        ) : error ? (
          <p className="ft-msg">{error}{" "}<button type="button" className="btn btn-sm" onClick={() => load(path)}>Try again</button></p>
        ) : view === null ? (
          <Skeleton />
        ) : (
          <div className="app-body">
            {(split ? detailBlocks : view.blocks).length === 0 ? (
              <Blank icon={manifest ? manifest.icon : ""} label={title} text={view.empty || "Nothing here."} />
            ) : (
              <PaneBlocks blocks={split ? detailBlocks : view.blocks} ctx={ctx} badge={selectedRow} />
            )}
          </div>
        )}
      </section>
    );
  }
  const listOnly = paneMode === "list";
  return (
    <section className={"app-surface" + (split && !listOnly ? " app-surface-split" : "")} aria-label={title} hidden={!!hidden} ref={rootRef}>
      <header className="ft-head">
        {path && !split ? (
          <button type="button" className="btn btn-sm btn-ghost app-back" title="Back" onClick={() => setPath("")}>
            <IconChevronLeft size={13} />
          </button>
        ) : null}
        <span className="app-head-icon"><AppIcon name={manifest ? manifest.icon : ""} label={title} size={14} /></span>
        <h2 className="ft-title" title={title}>{title}</h2>
        <div className="app-head-left" data-align-row>
          <TabStrip tabs={view ? view.tabs : []} path={path} onNavigate={setPath} name={tabsName} />
          {totalItems > 0 ? (
            <span className="app-search-wrap">
              <input
                type="search"
                className="app-search"
                placeholder={`Filter ${title.toLowerCase()}`}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Escape" && query) { e.preventDefault(); setQuery(""); } }}
                aria-label={`Filter ${title} by title or details`}
              />
            </span>
          ) : null}
        </div>
        <span className="ft-spacer" />
        <div className="app-head-right" data-align-row>
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => load(path)} disabled={busy}>
            Refresh
          </button>
          {onClose ? (
            <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
              Close
            </button>
          ) : null}
        </div>
      </header>

      {unsupported || badVersion ? (
        <p className="ft-msg">
          This app needs a newer PiCode — it speaks primitives v{badVersion ? manifest.apiVersion : "?"}, this build renders v{SUPPORTED_API}.
        </p>
      ) : error ? (
        <p className="ft-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={() => load(path)}>Try again</button>
        </p>
      ) : view === null ? (
        <Skeleton />
      ) : split && listOnly ? (
        <div className="app-body">
          {noMatches ? <SearchEmpty query={query} onClear={() => setQuery("")} /> : null}
          {filteredBodyBlocks.map((b, i) => <AppBlock key={i} block={b} {...ctx} />)}
          {listBlocks.length === 0 ? <Blank icon={manifest ? manifest.icon : ""} label={title} text={view.empty} /> : null}
        </div>
      ) : split ? (
        <div className={"app-split" + (resizing ? " resizing" : "")}>
          <div className="app-pane app-pane-list" style={stacked ? undefined : { flexBasis: listW }}>
            {noMatches ? <SearchEmpty query={query} onClear={() => setQuery("")} /> : null}
            {filteredBodyBlocks.map((b, i) => <AppBlock key={i} block={b} {...ctx} />)}
          </div>
          {stacked ? null : <div className="app-split-sizer" title="Drag to resize" onPointerDown={onSizerDown} />}
          <div className="app-pane app-pane-detail" ref={detailRef}>
            {detailBlocks.length === 0 ? (
              <Blank icon={manifest ? manifest.icon : ""} label={title} text="Nothing selected — Pick an item on the left." />
            ) : (
              <PaneBlocks blocks={detailBlocks} ctx={ctx} badge={selectedRow} />
            )}
          </div>
        </div>
      ) : (
        <div className="app-body">
          {noMatches ? <SearchEmpty query={query} onClear={() => setQuery("")} /> : null}
          {filteredBodyBlocks.map((b, i) => <AppBlock key={i} block={b} {...ctx} />)}
          {view.blocks.length === 0 ? <Blank icon={manifest ? manifest.icon : ""} label={title} text={view.empty} /> : null}
        </div>
      )}
    </section>
  );
}

// A form and the actions that follow it are one decision, so they share
// one button row: submit first, then the secondary and destructive ones.
function PaneBlocks({ blocks, ctx, badge }) {
  const formAt = blocks.findIndex((b) => b.type === "form");
  const actionsAt = blocks.findIndex((b) => b.type === "actions");
  const merged = formAt >= 0 && actionsAt > formAt;
  return (
    <>
      {blocks.map((b, i) => {
        if (merged && i === actionsAt) return null;
        const extra = merged && i === formAt ? blocks[actionsAt].actions : undefined;
        return <AppBlock key={i} block={b} {...ctx} extraActions={extra} badge={i === 0 ? badge : null} />;
      })}
    </>
  );
}

// A segmented filter strip, optional per view (ADR-0036: an additive View
// field, not a new block type). Radio group, not a tablist: the owner's
// call — a button-group read clearer than an underline once it sat in the
// header next to the search box. `name` must be per-mount (native radios
// group by name across the whole document, not per component) — the
// caller passes a useId() value. Reuses the same onNavigate/path plumbing
// a list row's Path already drives, so no new wiring exists just for tabs.
function TabStrip({ tabs, path, onNavigate, name }) {
  if (!tabs || tabs.length === 0) return null;
  return (
    <div className="app-tabs" role="radiogroup" aria-label="Filter" data-align-row>
      {tabs.map((t) => (
        <label className="app-tab-opt" key={t.id}>
          <input type="radio" name={name} checked={t.path === path} onChange={() => onNavigate(t.path)} />
          <span className="app-tab-face">
            {t.label}
            {t.badge ? <span className="app-tab-badge">{t.badge}</span> : null}
          </span>
        </label>
      ))}
    </div>
  );
}

function Skeleton() {
  return (
    <div className="app-body" aria-busy="true">
      {Array.from({ length: SKELETON_ROWS }, (_, i) => (
        <div key={i} className="app-skel-row">
          <span className="skel-line app-skel-title" style={{ width: 40 + ((i * 17) % 40) + "%" }} />
          <span className="skel-line app-skel-meta" />
        </div>
      ))}
    </div>
  );
}

// The app names its own emptiness (view.empty); the host owns how a
// blankslate looks — a quiet mark, one line, no call to action.
function Blank({ icon, label, text }) {
  const line = text || "Nothing here yet.";
  const [head, ...rest] = line.split(" — ");
  return (
    <div className="app-blank">
      {icon ? <AppIcon name={icon} label={label} size={24} /> : <IconInbox size={24} />}
      <p className="app-blank-title">{head}</p>
      {rest.length ? <p className="app-blank-sub">{rest.join(" — ")}</p> : null}
    </div>
  );
}

// Distinct from Blank/view.empty: this is "the search hid everything", not
// "the app has nothing" — same shape as the error state above (one line,
// one action) so it never reads as a silent, actionless well.
function SearchEmpty({ query, onClear }) {
  return (
    <p className="app-search-empty">
      No items match "{query}".{" "}
      <button type="button" className="btn btn-sm" onClick={onClear}>Clear search</button>
    </p>
  );
}

// A block's optional header: a section label with the count, plus the
// meta strip (drawn separators, so an empty chip leaves no gap).
function BlockHead({ block, count, badge }) {
  if (!block.title && block.meta.length === 0 && !block.at) return null;
  const heading = block.pane === "detail";
  return (
    <div className={heading ? "app-detail-head" : "app-sect"}>
      {block.title ? <span className={heading ? "app-detail-title" : "app-sect-title"}>{block.title}</span> : null}
      {!heading && typeof count === "number" ? <span className="app-sect-count">{count}</span> : null}
      {block.meta.length || block.at || badge ? (
        <span className="app-meta">
          {badge && badge.badge ? (
            <span className={"app-kind" + (badge.tone ? " tone-" + badge.tone : "")}>{badge.badge}</span>
          ) : null}
          {block.meta.map((m, i) => <span key={i}>{m}</span>)}
          {block.at ? <span className="app-when" title={absTime(block.at)}>{relTime(block.at)}</span> : null}
        </span>
      ) : null}
    </div>
  );
}

function AppBlock({ block, onNavigate, onAction, selected, extraActions, badge }) {
  if (block.type === "detail") {
    return (
      <div className="app-block">
        <BlockHead block={block} badge={badge} />
        <div className="app-detail md">
          <Markdown remarkPlugins={[remarkGfm]}>{block.markdown}</Markdown>
        </div>
      </div>
    );
  }
  if (block.type === "list") {
    return (
      <div className="app-block">
        <BlockHead block={block} count={block.items.length} />
        <ul className="app-list">
          {block.items.map((it) => (
            <Row key={it.id} item={it} onNavigate={onNavigate} onAction={onAction} active={!!it.path && it.path === selected} />
          ))}
        </ul>
      </div>
    );
  }
  if (block.type === "form") {
    return (
      <div className="app-block">
        <BlockHead block={block} />
        <AppForm key={(selected || "") + ":" + block.form.id} form={block.form} onAction={onAction} extraActions={extraActions} />
      </div>
    );
  }
  // actions
  return (
    <div className="app-block">
      <BlockHead block={block} />
      <div className="app-actions">
        {block.actions.map((a) => (
          <ActionButton key={a.id} action={a} onAction={onAction} />
        ))}
      </div>
    </div>
  );
}

// Emphasis is the app's call (Action.primary), not a guess from position:
// on an approval the decision deserves the fill, on a result nothing does.
function ActionButton({ action, onAction }) {
  return (
    <button
      type="button"
      className={"btn btn-sm" + (action.danger ? " btn-danger" : action.primary ? " btn-primary" : "")}
      onClick={() => onAction(action)}
    >
      {action.label}
    </button>
  );
}

const ROW_ICONS = { check: IconCheck, clock: IconClock, trash: IconTrash };

// One dense row: unread dot, title, relative time, then a meta strip.
// Row actions stay hidden until hover or keyboard focus, so they cost no
// width and never set the row's height.
function Row({ item, onNavigate, onAction, active }) {
  const has = item.path && onNavigate;
  // Touch has no hover: a left swipe reveals the row's actions (Done,
  // Snooze, Delete), a tap anywhere else puts them away. The row follows
  // the finger while dragging (inline transform, no React state per
  // move) and snaps open or shut on release with the CSS transition.
  // Desktop keeps hover/focus; the class only adds a third way in.
  const [swiped, setSwiped] = useState(false);
  const touch = useRef(null);
  const mainRef = useRef(null);
  const reveal = item.actions.length * 44 + 8; // px the actions need
  const onTouchStart = (e) => {
    if (e.touches.length !== 1) return;
    touch.current = { x: e.touches[0].clientX, y: e.touches[0].clientY, axis: "", dx: 0 };
  };
  const onTouchMove = (e) => {
    const t = touch.current;
    if (!t || e.touches.length !== 1) return;
    const dx = e.touches[0].clientX - t.x;
    const dy = e.touches[0].clientY - t.y;
    if (!t.axis) {
      if (Math.abs(dx) < 10 && Math.abs(dy) < 10) return;
      t.axis = Math.abs(dx) > Math.abs(dy) ? "x" : "y";
    }
    if (t.axis !== "x") return;
    t.dx = dx;
    const base = swiped ? -reveal : 0;
    const x = Math.max(-reveal, Math.min(0, base + dx));
    const el = mainRef.current;
    if (el) { el.style.transition = "none"; el.style.transform = "translateX(" + x + "px)"; }
  };
  const onTouchEnd = () => {
    const t = touch.current;
    touch.current = null;
    const el = mainRef.current;
    if (el) { el.style.transition = ""; el.style.transform = ""; }
    if (!t || t.axis !== "x") return;
    if (t.dx < -reveal / 2) setSwiped(true);
    else if (t.dx > reveal / 2) setSwiped(false);
  };
  return (
    <li
      className={"app-row" + (active ? " app-row-on" : "") + (item.unread ? " app-row-unread" : "") + (swiped ? " app-row-swiped" : "")}
      style={item.actions.length ? { "--swipe-w": reveal + "px" } : undefined}
      onTouchStart={item.actions.length ? onTouchStart : undefined}
      onTouchMove={item.actions.length ? onTouchMove : undefined}
      onTouchEnd={item.actions.length ? onTouchEnd : undefined}
      onTouchCancel={item.actions.length ? onTouchEnd : undefined}
    >
      <button
        type="button"
        className="app-row-main"
        ref={mainRef}
        onClick={() => { if (swiped) { setSwiped(false); return; } if (has) onNavigate(item.path); }}
        disabled={!has}
      >
        <span className="app-row-line1">
          <span className="app-row-title">{item.title}</span>
          {item.at ? <span className="app-when" title={absTime(item.at)}>{relTime(item.at)}</span> : null}
        </span>
        <span className="app-row-line2">
          {item.badge ? <span className={"app-kind" + (item.tone ? " tone-" + item.tone : "")}>{item.badge}</span> : null}
          <span className="app-meta">
            {item.meta.map((m, i) => <span key={i}>{m}</span>)}
            {item.subtitle ? <span>{item.subtitle}</span> : null}
          </span>
        </span>
      </button>
      {item.actions.length ? (
        <span className="app-row-actions">
          {item.actions.map((a) => {
            const Glyph = ROW_ICONS[a.icon];
            return (
              <button
                key={a.id}
                type="button"
                className={"ws-icon-btn" + (a.danger ? " danger" : "")}
                title={a.label}
                aria-label={a.label}
                onClick={() => onAction(a)}
              >
                {Glyph ? <Glyph size={13} /> : a.label}
              </button>
            );
          })}
        </span>
      ) : null}
    </li>
  );
}

function AppForm({ form, onAction, extraActions }) {
  const [values, setValues] = useState(() => {
    const v = {};
    for (const f of form.fields) v[f.name] = f.method === "confirm" ? "no" : (f.prefill || (f.method === "select" ? f.options[0] || "" : ""));
    return v;
  });
  const set = (name, val) => setValues((cur) => ({ ...cur, [name]: val }));
  const submit = async () => {
    // A TUI reply lands in the agent's running terminal (ADR-0060); the
    // host owns delivery and durable proof, so the form just submits.
    return onAction({ id: form.id, label: form.submit || "Submit", args: {} }, values);
  };
  const fireExtra = (action) => onAction(action);
  const hasEditor = form.fields.some((f) => f.method === "editor");
  const extraPrimary = (extraActions || []).some((a) => a.primary);
  return (
    <form
      className="app-form"
      noValidate
      onSubmit={(e) => { e.preventDefault(); submit(); }}
    >
      {form.fields.map((f) => (
        <label key={f.name} className="app-field">
          {f.title ? <span className="app-field-title">{f.title}</span> : null}
          {f.method === "select" ? (
            <select className="dlg-input" value={values[f.name]} onChange={(e) => set(f.name, e.target.value)}>
              {f.options.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
          ) : f.method === "confirm" ? (
            <span className="app-field-confirm">
              <input type="checkbox" checked={values[f.name] === "yes"} onChange={(e) => set(f.name, e.target.checked ? "yes" : "no")} />
            </span>
          ) : f.method === "editor" ? (
            <textarea
              className="dlg-input app-field-editor"
              rows={3}
              value={values[f.name]}
              placeholder={f.placeholder}
              onChange={(e) => set(f.name, e.target.value)}
              onKeyDown={(e) => {
                // Send without reaching for the mouse — the shortcut every
                // composer in this app already answers to.
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); submit(); }
              }}
            />
          ) : (
            <input className="dlg-input" type="text" value={values[f.name]} placeholder={f.placeholder} onChange={(e) => set(f.name, e.target.value)} />
          )}
          {f.message ? <span className="app-field-msg">{f.message}</span> : null}
        </label>
      ))}
      <div className="app-actions">
        {/* Only one filled button per row: if the app marked a decision as
            primary, submitting the reply is the side channel. */}
        <button type="submit" className={"btn btn-sm" + (extraPrimary ? "" : " btn-primary")}>
          {form.submit || "Submit"}
        </button>
        {(extraActions || []).map((a) => (
          <ActionButton key={a.id} action={a} onAction={fireExtra} />
        ))}
        {hasEditor ? (
          <span className="app-field-hint"><span className="app-key">Ctrl</span>+<span className="app-key">Enter</span> to send</span>
        ) : null}
      </div>
    </form>
  );
}
