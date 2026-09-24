import { useCallback, useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { normalizeView, supportedApp, SUPPORTED_API, activeAppTab } from "@picode/shared/contracts/appPrimitives.js";
import { useKeptScroll } from "../lib/keepScroll.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import { filterListBlocks, countListItems } from "@picode/shared/domain/appSearch.js";
import { IconBack, IconChevronLeft, IconChevronRight, IconCheck, IconClock, IconTrash, IconPackage } from "./Icons.jsx";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { surfaceLinkClick } from "@picode/shared/client/surfaceLinks.js";
import { touches } from "@picode/shared/domain/feedReducers.js";
import { createRefreshQueue } from "@picode/shared/domain/appRefreshQueue.js";
import { appFormSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { readGroupPreferences, writeGroupPreferences, groupIsOpen, resetGroupSearch, toggleGroup } from "@picode/shared/domain/appGroups.js";

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
// The chrome is a **route page** (2026-09-14): .settings-wrap +
// .settings-head + .settings-card, the same frame PageFrame draws, with the
// view's own tabs as an underline nav inside the card. A list app is a page;
// the canvas surfaces (terminal, file tree, git graph) keep their own.
// paneMode (ADR-0044): the phone renders a split app as two screens —
// "list" (rows + tabs + search; an item row calls onOpenItem instead of
// selecting) and "detail" (one item's panes under the shell's own Back
// header; an action that returns to the root calls onClose). Undefined
// keeps the desktop's split. onGoto receives an action's goto directive
// ("agent:<id>"); each shell opens its
// own agent terminal surface. nativeSurfaces is the shell's registry of
// native app ids (ADR-0109): a native app this build did not compile in
// reaches this surface only by deep link and gets the honest line.
// onOpenUrl receives an external http(s) URL clicked inside the surface —
// the shell opens it as a work-browser tab (2026-09-20/21: an Inbox github
// link navigated this whole document and took the shell with it; an app
// surface never navigates its host).
export default function AppSurface({ appId, apiBase, backHref, hidden, manifest, onClose, initialPath, onPathChange, refreshKey, paneMode, onOpenItem, onGoto, nativeSurfaces, onOpenUrl }) {
  // initialPath (ADR-0044): a deep link — the phone's #/inbox/<id> — lands
  // on that item instead of the list. Later changes to it navigate too.
  const [path, setPath] = useState(initialPath || "");
  useEffect(() => { if (initialPath != null) setPath(initialPath); }, [initialPath]);
  const changePath = (next) => { setPath(next); setQuery(""); if (onPathChange) onPathChange(next); };
  const [view, setView] = useState(null); // normalized tree
  const [unsupported, setUnsupported] = useState(false);
  const [error, setError] = useState("");
  const [missing, setMissing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [pending, setPending] = useState("");
  const actionRef = useRef(false);
  const [query, setQuery] = useState("");
  const [groups, setGroups] = useState(() => ({ saved: readGroupPreferences(), search: { query: "", values: {} } }));
  useEffect(() => { writeGroupPreferences(groups.saved); }, [groups.saved]);
  useEffect(() => { setGroups((state) => resetGroupSearch(state, query)); }, [query]);
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
  // The one link guard (shared/client/surfaceLinks.js), delegated on this
  // root in the capture phase: every block renderer is covered — above all
  // the markdown that mints plain anchors — present and future. External
  // http(s) links open as work-browser tabs (onOpenUrl) in the shell, or a
  // _blank tab outside it; the hosting document never navigates.
  const onSurfaceLinkClick = surfaceLinkClick({ origin: window.location.origin, shell: !!window.__TAURI__, onOpenUrl });
  const loadRef = useRef(() => {});
  const pathRef = useRef("");
  // Mounting counts as a read: the effect below loads immediately, and the
  // reveal check must not fire a second load on top of it.
  const lastLoadRef = useRef(Date.now());

  const load = useCallback(async (p) => {
    const seq = ++seqRef.current;
    setBusy(true);
    try {
      const raw = await api((apiBase || "/api/apps/" + encodeURIComponent(appId)) + "/view" + (p ? "?path=" + encodeURIComponent(p) : ""));
      if (seq !== seqRef.current) return; // a newer load superseded this one
      const v = normalizeView(raw);
      if (v) { setView(v); setUnsupported(false); setError(""); setMissing(false); }
      else { setView(null); setUnsupported(true); setError(""); setMissing(false); }
    } catch (e) {
      if (seq === seqRef.current) {
        // A read of a path that is gone is not a *failing* read: the item was
        // answered or removed elsewhere, and a stale deep link (ADR-0044) is
        // the easy way to hit it. A retry cannot succeed, so the card offers
        // the way back to the list and says so in a sentence — the raw store
        // text ("not found") is a fragment. The apps report this as a 404 or
        // as a not-found message (the inbox answers 500 with "not found",
        // which internal/server/apps.go should map to 404 one day).
        const gone = !!p && ((Number(e && e.status) || 0) === 404 || /^\s*not found\s*$/i.test(String((e && e.message) || "")));
        setMissing(gone);
        setError(gone ? "This item is no longer in the list." : humanizeError(e.message || String(e)));
      }
    } finally {
      if (seq === seqRef.current) {
        setBusy(false);
        lastLoadRef.current = Date.now();
      }
    }
  }, [appId, apiBase]);

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
    const queue = createRefreshQueue(() => loadRef.current(pathRef.current));
    const unsubscribe = subscribeFeed((ev) => {
      if (ev.type === "feed.reset" || ev.type === "feed.open" || touches(ev, [appId])) queue.request();
    });
    return () => { unsubscribe(); queue.stop(); };
  }, [appId]);
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
    if (actionRef.current) return;
    if (action.confirm) {
      const ok = await askConfirm({ title: action.label, message: action.confirm, confirmLabel: action.label, danger: !!action.danger });
      if (!ok) return;
    }
    if (actionRef.current) return;
    actionRef.current = true;
    setPending(action.label + " in progress…");
    try {
      const res = await api((apiBase || "/api/apps/" + encodeURIComponent(appId)) + "/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: action.id, path, args: { requestKey: crypto.randomUUID(), ...(action.args || {}), ...(args || {}) } }),
      });
      if (res && res.toast) toast(res.toast, "ok");
      // A goto directive outranks every in-app outcome: the action asks
      // the shell to leave the app ("agent:" → the agent's tab).
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
        changePath(res.path);
      } else if (res && typeof res.path === "string") {
        await load(res.path);
      }
    } catch (e) {
      toastError(e);
    } finally {
      actionRef.current = false;
      setPending("");
    }
  }

  const title = (manifest && manifest.name) || appId;
  const badVersion = manifest && !supportedApp(manifest, nativeSurfaces);
  const split = !!view && view.layout === "split";
  const listBlocks = split ? view.blocks.filter((b) => b.pane === "list") : [];
  const detailBlocks = split ? view.blocks.filter((b) => b.pane !== "list") : [];
  const navigate = (p) => {
    if (paneMode === "list" && onOpenItem && typeof p === "string" && p.startsWith("item/")) { onOpenItem(p); return; }
    changePath(p);
  };
  const groupKey = (id) => JSON.stringify([appId, id]);
  const ctx = {
    onNavigate: navigate, onAction: fire, selected: path, pending: !!pending,
    groupOpen: (id) => groupIsOpen(groups, groupKey(id), query),
    onGroupToggle: (id) => setGroups((state) => toggleGroup(state, groupKey(id), query)),
    filtering: !!query.trim(),
  };
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
  // A list-pane actions row is the list's own bulk action (Inbox's "Clear
  // all done"), so it rides the card toolbar at the filter's right — the
  // toolbar row every route page keeps its list actions in — instead of
  // trailing the list. Detail-pane rows (an item's Done/Snooze) and unpaned
  // rows (an empty state's "Check again") are not list chrome and stay put.
  const isBulkRow = (b) => b.type === "actions" && b.pane === "list";
  const bulkActions = filteredBodyBlocks.filter(isBulkRow).flatMap((b) => b.actions || []);
  const listPaneBlocks = filteredBodyBlocks.filter((b) => !isBulkRow(b));

  if (paneMode === "detail") {
    return (
      <section className="app-surface" aria-label={title} hidden={!!hidden} ref={rootRef} onClickCapture={onSurfaceLinkClick}>
        <div className="settings-wrap">
          <div className="settings-card app-card">
            <PendingNotice text={pending} />
            {unsupported || badVersion ? (
              <p className="app-card-msg">This app needs a newer PiCode.</p>
            ) : error ? (
              <p className="app-card-msg">{error}{" "}<button type="button" className="btn btn-sm" onClick={() => (missing ? changePath("") : load(path))}>{missing ? "Back to the list" : "Try again"}</button></p>
            ) : view === null ? (
              <Skeleton />
            ) : (
              <div className="app-body">
                {(split ? detailBlocks : view.blocks).length === 0 ? (
                  <Blank text={view.empty || "Nothing here."} />
                ) : (
                  <PaneBlocks blocks={split ? detailBlocks : view.blocks} ctx={ctx} badge={selectedRow} />
                )}
              </div>
            )}
          </div>
        </div>
      </section>
    );
  }
  const listOnly = paneMode === "list";
  // A path inside a stacked view needs its way back to the list; a split
  // keeps the list in view, so it does not.
  const backToRoot = !!path && !split && paneMode !== "list";
  return (
    <section className={"app-surface" + (split && !listOnly ? " app-surface-split" : "")} aria-label={title} hidden={!!hidden} ref={rootRef} onClickCapture={onSurfaceLinkClick}>
      <div className="settings-wrap">
        <header className="settings-head">
          {backHref ? <a href={backHref} className="btn btn-ghost btn-sm"><IconBack />Back</a> : null}
          <h2 className="app-page-title" title={title}>{title}</h2>
          <span className="ft-spacer" />
          <div className="app-page-actions" data-align-row>
            <button type="button" className="btn btn-sm btn-ghost" onClick={() => load(path)} disabled={busy}>
              Refresh
            </button>
            {onClose && !backHref ? (
              <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
                Close
              </button>
            ) : null}
          </div>
        </header>

        <div className="settings-card app-card">
          <PageTabs tabs={view ? view.tabs : []} path={path} onNavigate={changePath} />

          <PendingNotice text={pending} />

          {unsupported || badVersion ? (
            <p className="app-card-msg">
              {badVersion && manifest.surface && manifest.apiVersion === SUPPORTED_API
                ? <>This app needs a newer PiCode — its “{manifest.surface}” surface is not in this build.</>
                : <>This app needs a newer PiCode — it speaks primitives v{badVersion ? manifest.apiVersion : "?"}, this build renders v{SUPPORTED_API}.</>}
            </p>
          ) : error ? (
            <p className="app-card-msg">
              {error}{" "}
              <button type="button" className="btn btn-sm" onClick={() => (missing ? changePath("") : load(path))}>
                {missing ? "Back to the list" : "Try again"}
              </button>
            </p>
          ) : view === null ? (
            <Skeleton />
          ) : (
            <>
              {/* Search sits in the card, not in the page head: the rows it
                  filters live in the card (2026-09-14 parity). The list's own
                  bulk action shares the row, right-aligned — one toolbar,
                  like every route page. */}
              {backToRoot || totalItems > 0 || bulkActions.length > 0 ? (
                <div className="app-card-toolbar" data-align-row>
                  {backToRoot ? (
                    <button type="button" className="app-card-back" onClick={() => changePath("")}>
                      <IconChevronLeft size={13} />
                      {title}
                    </button>
                  ) : null}
                  {totalItems > 0 ? (
                    <input
                      type="search"
                      className="app-search"
                      placeholder={`Filter ${title.toLowerCase()}`}
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      onKeyDown={(e) => { if (e.key === "Escape" && query) { e.preventDefault(); setQuery(""); } }}
                      aria-label={`Filter ${title} by title or details`}
                    />
                  ) : null}
                  {bulkActions.length > 0 ? (
                    <div className="app-card-bulk">
                      {bulkActions.map((a) => <ActionButton key={a.id} action={a} onAction={ctx.onAction} pending={ctx.pending} />)}
                    </div>
                  ) : null}
                </div>
              ) : null}

              {split && listOnly ? (
                <div className="app-body">
                  {noMatches ? <SearchEmpty query={query} onClear={() => setQuery("")} /> : null}
                  {listPaneBlocks.map((b, i) => <AppBlock key={b.id || i} block={b} {...ctx} />)}
                  {listBlocks.length === 0 ? <Blank text={view.empty} /> : null}
                </div>
              ) : split ? (
                <div className={"app-split" + (resizing ? " resizing" : "")}>
                  <div className="app-pane app-pane-list" style={stacked ? undefined : { flexBasis: listW }}>
                    {noMatches ? <SearchEmpty query={query} onClear={() => setQuery("")} /> : null}
                    {listPaneBlocks.map((b, i) => <AppBlock key={b.id || i} block={b} {...ctx} />)}
                  </div>
                  {stacked ? null : <div className="app-split-sizer" title="Drag to resize" onPointerDown={onSizerDown} />}
                  <div className="app-pane app-pane-detail" ref={detailRef}>
                    {detailBlocks.length === 0 ? (
                      <Blank text="Nothing selected — Pick an item from the list." />
                    ) : (
                      <PaneBlocks blocks={detailBlocks} ctx={ctx} badge={selectedRow} />
                    )}
                  </div>
                </div>
              ) : (
                <div className="app-body">
                  {noMatches ? <SearchEmpty query={query} onClear={() => setQuery("")} /> : null}
                  {listPaneBlocks.map((b, i) => <AppBlock key={b.id || i} block={b} {...ctx} />)}
                  {view.blocks.length === 0 ? <Blank text={view.empty} /> : null}
                </div>
              )}
            </>
          )}
        </div>
      </div>
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
        return <AppBlock key={b.id || i} block={b} {...ctx} extraActions={extra} badge={i === 0 ? badge : null} />;
      })}
    </>
  );
}

// The view's own tabs, optional per view (ADR-0036: an additive View field,
// not a new block type), as one underline nav inside the card — the Agent
// CLIs rhythm (.cli-tabs). It was a segmented radio group while it sat in
// the app's own toolbar; on a page frame the product has one tab idiom
// (2026-09-14, owner's call). Buttons, not links: navigation is this
// surface's own state and the host owns the hash.
function PageTabs({ tabs, path, onNavigate }) {
  if (!tabs || tabs.length === 0) return null;
  const active = activeAppTab(tabs, path);
  return (
    <nav className="app-page-tabs" aria-label="Views">
      {tabs.map((t) => (
        <button
          key={t.id}
          type="button"
          className="app-page-tab"
          aria-current={t.id === active ? "page" : undefined}
          onClick={() => onNavigate(t.path)}
        >
          {t.label}
          {t.badge ? <span className="app-tab-count">{t.badge}</span> : null}
        </button>
      ))}
    </nav>
  );
}

function PendingNotice({ text }) {
  const last = useRef("");
  if (text) last.current = text;
  return <div className={"app-operation-wrap" + (text ? " is-active" : "")} aria-hidden={!text}>
    <div><p className="app-operation" role={text ? "status" : undefined}>{last.current}</p></div>
  </div>;
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
// blankslate looks — one line, where the reader is already looking, the same
// idiom a route's empty state uses (.mcp-empty), never a poster centred in
// the card.
function Blank({ text }) {
  const line = text || "Nothing here yet.";
  const [head, ...rest] = line.split(" — ");
  return (
    <div className="app-empty">
      <p className="app-empty-title">{head}</p>
      {rest.length ? <p className="app-empty-sub">{rest.join(" — ")}</p> : null}
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
      {!heading && typeof count === "number" && count > 0 ? <span className="app-sect-count">{count}</span> : null}
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

function AppBlock({ block, onNavigate, onAction, selected, extraActions, badge, pending, groupOpen, onGroupToggle, filtering }) {
  if (block.type === "detail") {
    return (
      <div className={"app-block" + (block.busy ? " app-block-busy" : "")} aria-busy={block.busy || undefined}>
        <BlockHead block={block} badge={badge} />
        {typeof block.text === "string" ? <pre className="app-output">{block.text || "No output yet."}</pre> : <div className="app-detail md">
          <Markdown remarkPlugins={[remarkGfm]}>{block.markdown}</Markdown>
        </div>}
      </div>
    );
  }
  if (block.type === "list") {
    const content = <>
        {block.items.length === 0 ? <p className="app-list-empty">{block.empty || "No items yet."}</p> : null}
        <ul className="app-list">
          {block.items.map((it) => (
            <Row key={it.id} item={it} onNavigate={onNavigate} onAction={onAction} pending={pending} active={!!it.path && it.path === selected} />
          ))}
        </ul>
        {block.actions.length ? <div className="app-actions app-list-actions">
          {block.actions.map((action) => <ActionButton key={action.id} action={action} onAction={onAction} pending={pending} />)}
        </div> : null}
      </>;
    if (block.collapsible) {
      const open = groupOpen(block.id);
      const count = block.items.length;
      return <details className="app-block app-group" open={open} aria-busy={block.busy || undefined}>
        <summary className="app-group-summary" onClick={(event) => { event.preventDefault(); onGroupToggle(block.id); }}>
          <IconChevronRight className="app-group-chevron" size={14} />
          <IconPackage className="app-group-icon" size={16} />
          <span className="app-group-title">{block.title}</span>
          {count > 0 ? <span className="app-group-count">{count}{filtering ? (count === 1 ? " match" : " matches") : ""}</span> : null}
          {block.meta.length > 0 ? <span className="app-group-meta">{block.meta.join(" · ")}</span> : null}
        </summary>
        <div className="app-group-items">{content}</div>
      </details>;
    }
    return <div className="app-block"><BlockHead block={block} count={block.items.length} />{content}</div>;
  }
  if (block.type === "form") {
    return (
      <div className="app-block">
        <BlockHead block={block} />
        <AppForm key={(selected || "") + ":" + block.form.id} form={block.form} onAction={onAction} extraActions={extraActions} pending={pending} />
      </div>
    );
  }
  // actions
  return (
    <div className="app-block">
      <BlockHead block={block} />
      <div className="app-actions">
        {block.actions.map((a) => (
          <ActionButton key={a.id} action={a} onAction={onAction} pending={pending} />
        ))}
      </div>
    </div>
  );
}

// Emphasis is the app's call (Action.primary), not a guess from position:
// on an approval the decision deserves the fill, on a result nothing does.
function ActionButton({ action, onAction, pending }) {
  return (
    <button
      type="button"
      className={"btn btn-sm" + (action.danger ? " btn-danger" : action.primary ? " btn-primary" : "")}
      onClick={() => onAction(action)}
      disabled={pending}
    >
      {action.label}
    </button>
  );
}

const ROW_ICONS = { check: IconCheck, clock: IconClock, trash: IconTrash };

// One dense row: unread dot, title, relative time, then a meta strip.
// Row actions stay hidden until hover or keyboard focus, so they cost no
// width and never set the row's height.
function Row({ item, onNavigate, onAction, active, pending }) {
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
      className={"app-row" + (active ? " app-row-on" : "") + (item.unread ? " app-row-unread" : "") + (swiped ? " app-row-swiped" : "") + (item.busy ? " app-row-busy" : "") + (item.wrap ? " app-row-wrap" : "")}
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
                disabled={pending}
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

function AppForm({ form, onAction, extraActions, pending }) {
  const [error, setError] = useState("");
  const [values, setValues] = useState(() => {
    const v = {};
    for (const f of form.fields) v[f.name] = f.method === "confirm" ? "no" : (f.prefill || (f.method === "select" ? f.options[0] || "" : ""));
    return v;
  });
  const set = (name, val) => setValues((cur) => ({ ...cur, [name]: val }));
  const submit = async () => {
    const parsed = parseForm(appFormSchema(form.fields), values);
    setError(parsed.error);
    if (!parsed.ok) return;
    // A TUI reply lands in the agent's running terminal (ADR-0060); the
    // host owns delivery and durable proof, so the form just submits.
    return onAction({ id: form.id, label: form.submit || "Submit", args: {} }, parsed.value);
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
      {error ? <p className="app-form-error" role="alert">{error}</p> : null}
      <div className="app-actions">
        {/* Only one filled button per row: if the app marked a decision as
            primary, submitting the reply is the side channel. */}
        <button type="submit" disabled={pending} className={"btn btn-sm" + (extraPrimary ? "" : " btn-primary")}>
          {form.submit || "Submit"}
        </button>
        {(extraActions || []).map((a) => (
          <ActionButton key={a.id} action={a} onAction={fireExtra} pending={pending} />
        ))}
        {hasEditor ? (
          <span className="app-field-hint"><span className="app-key">Ctrl</span>+<span className="app-key">Enter</span> to send</span>
        ) : null}
      </div>
    </form>
  );
}
