import { useEffect, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { appTile } from "../lib/nativeApps.js";
import { visibleApps } from "../lib/appsGrid.js";
import { webappError, submitWebappForm } from "../lib/webapps.js";
import { toast } from "../lib/toast.js";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { SortableList, SortableRow } from "./SortableRows.jsx";
import AppIcon from "./AppIcon.jsx";
import * as Dialog from "./ResponsiveDialog.jsx";
import { Alert as AlertDialog } from "./ResponsiveDialog.jsx";
import { IconSearch, IconPlus, IconEllipsis, IconPencil, IconTrash, IconReload } from "./Icons.jsx";

// The icon endpoint URL. `version` (a timestamp bumped on refresh) busts
// the browser cache so a re-fetched icon shows without a page reload.
// Kept local: the same-shaped export in lib/webapps.js built a dangling
// global in the minified bundle once (vite workspace resolution quirk).
function webappIconURL(id, version) {
  const base = "/api/webapps/" + encodeURIComponent(id) + "/icon";
  return version ? base + "?v=" + encodeURIComponent(version) : base;
}

function WebappDialog({ app, anotherAccount, onClose, onSaved, onOpen }) {
  const editing = !!app;
  const another = !!anotherAccount;
  const [url, setUrl] = useState(editing ? app.url : another ? anotherAccount.url : "");
  const [name, setName] = useState(editing ? app.name : another ? anotherAccount.name + " (2)" : "");
  const [preview, setPreview] = useState(editing ? { url: app.url } : another ? { url: anotherAccount.url } : null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [existing, setExisting] = useState(null);
  async function run(fn) {
    if (busy) return;
    setBusy(true);
    setError("");
    setExisting(null);
    try { await fn(); } catch (e) {
      setError(webappError(e));
      if (e?.body?.reason === "duplicate") setExisting(e.body.existing);
    } finally { setBusy(false); }
  }
  function check(e) {
    e.preventDefault();
    run(async () => {
      const { preview: resolved } = await submitWebappForm(api, { mode: "install", preview: null, url, name });
      setPreview(resolved);
      if (!name.trim()) setName(resolved.name || "");});
  }
  async function install(force) {
    await run(async () => {
      const { saved } = await submitWebappForm(api, { mode: "install", preview, url, name, allowDuplicate: another || force });
      onSaved(saved);
      onClose();
    });
  }
  async function save(e) {
    e.preventDefault();
    await run(async () => {
      const { saved } = await submitWebappForm(api, { mode: "rename", app, name });
      onSaved(saved);
      onClose();
    });
  }
  const resolvedUrl = preview?.url || "";
  return (
    <Dialog.Root open onOpenChange={(open) => { if (!open && !busy) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-webapp" aria-busy={busy}>
          <Dialog.Title className="dlg-title">{editing ? "Rename web app" : another ? "Add another account" : "Add a web app"}</Dialog.Title>
          <Dialog.Description className="dlg-body">
            {editing
              ? "Choose a name for this shortcut."
              : another
                ? `Another independent copy of ${anotherAccount.name} — its logins and data stay separate.`
                : "Enter the address of a web app to add it to Apps."}
          </Dialog.Description>
          {editing || preview ? (
            <form noValidate onSubmit={editing ? save : (e) => { e.preventDefault(); install(); }}>
              <label className="webapp-field">
                <span>Name</span>
                <input
                  className="dlg-input"
                  type="text"
                  placeholder={editing ? "Name" : "Filled from the site"}
                  value={name}
                  autoFocus
                  disabled={busy}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              {!editing && resolvedUrl ? <p className="webapp-resolved" role="status">Will open {preview.pwa && preview.startUrl ? preview.startUrl : resolvedUrl}</p> : null}
              {!editing && preview.pwa ? <p className="webapp-resolved" role="note">Web app detected — it installs with its own name, icon and start page.</p> : null}
              {error ? <p className="form-error" role="alert">{error}</p> : null}
              <div className="dlg-actions" data-align-row>
                {!editing && <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setPreview(null); setError(""); setExisting(null); }} disabled={busy}>Back</button>}
                <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={busy}>Cancel</button>
                {existing ? (
                  <>
                    <button type="button" className="btn btn-primary btn-sm" onClick={() => { onOpen(existing); onClose(); }}>Open existing</button>
                    <button type="button" className="btn btn-sm" onClick={() => install(true)} disabled={busy}>Add as new account</button>
                  </>
                ) : <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? editing ? "Saving…" : "Adding…" : editing ? "Save" : "Add app"}</button>}
              </div>
            </form>
          ) : (
            <form noValidate onSubmit={check}>
              <label className="webapp-field">
                <span>Address</span>
                <input
                  className="dlg-input"
                  type="text"
                  inputMode="url"
                  placeholder="example.com"
                  value={url}
                  autoFocus
                  disabled={busy}
                  onChange={(e) => setUrl(e.target.value)}
                />
              </label>
              {error ? <p className="form-error" role="alert">{error}</p> : null}
              <div className="dlg-actions">
                <button type="button" className="btn btn-ghost btn-sm" onClick={onClose} disabled={busy}>Cancel</button>
                <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>{busy ? "Checking…" : "Continue"}</button>
              </div>
            </form>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

export default function AppsGrid({ apps, webapps = [], webappsErr = "", webappsLoaded = false, nativeApps, onOpen, onOpenWebapp, onSavedWebapp, onRemoveWebapp, onRefreshWebapp, onClearWebappData, iconVersions = {}, onRetryWebapps, desktop }) {
  const [q, setQ] = useState("");
  const [adding, setAdding] = useState(false);
  const [renaming, setRenaming] = useState(null);
  const [anotherFor, setAnotherFor] = useState(null);
  const [removing, setRemoving] = useState(null);
  const [removeBusy, setRemoveBusy] = useState(false);
  const [removeError, setRemoveError] = useState("");
  const [refreshingId, setRefreshingId] = useState(null);
  const [clearing, setClearing] = useState(null);
  const [clearBusy, setClearBusy] = useState(false);
  const [clearError, setClearError] = useState("");
  const [order, setOrder] = useState(null);
  useEffect(() => {
    let stop = false;
    function load() {
      api("/api/apps/order").then((d) => { if (!stop) setOrder((d && d.ids) || []); }).catch(() => { if (!stop) setOrder([]); });
    }
    load();
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "setting.updated" && ev.data && ev.data.key === "apps.grid.order") load();
    });
    return () => { stop = true; unsub(); };
  }, []);
  async function commitOrder(ids) {
    const prev = order;
    setOrder(ids);
    try {
      await api("/api/apps/order", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ids }) });
    } catch (e) {
      setOrder(prev);
      toast(webappError(e));
    }
  }
  async function refresh(a) {
    if (refreshingId) return;
    setRefreshingId(a.id);
    try { await onRefreshWebapp(a); }
    catch (e) { toast(webappError(e)); }
    finally { setRefreshingId(null); }
  }
  async function clearData() {
    if (clearBusy) return;
    setClearBusy(true);
    setClearError("");
    try { await onClearWebappData(clearing); setClearing(null); }
    catch (e) { setClearError(webappError(e)); }
    finally { setClearBusy(false); }
  }
  async function remove() {
    if (removeBusy) return;
    setRemoveBusy(true);
    setRemoveError("");
    try { await onRemoveWebapp(removing); setRemoving(null); }
    catch (e) { setRemoveError(webappError(e)); }
    finally { setRemoveBusy(false); }
  }
  const catalog = [
    ...(apps || []).filter((a) => a && a.id).map((a) => ({ ...a, key: "app:" + a.id, kind: "app" })),
    ...(webapps || []).filter((a) => a && a.id).map((a) => ({ ...a, key: "web:" + a.id, kind: "web" })),
  ];
  const tiles = visibleApps(catalog.map((t) => ({ id: t.key, name: t.name, tile: t })), q, order || []);
  const searching = !!q.trim();
  const empty = webappsLoaded && tiles.length === 0;
  return (
    <div className="side-section">
      <div className="pins-head">
        <span className="pins-title">Apps</span>
        <button type="button" className="ws-icon-btn" title="Add a web app" aria-label="Add a web app" onClick={() => setAdding(true)}><IconPlus /></button>
      </div>
      <label className="pins-search">
        <IconSearch />
        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search apps"
          aria-label="Search apps"
          onKeyDown={(e) => { if (e.key === "Escape") setQ(""); }}
        />
      </label>
      <div className="side-scroll">
      {webappsErr ? (
        <p className="side-empty pins-empty" role="alert">
          {webappsErr} <button type="button" className="side-empty-act" onClick={onRetryWebapps}>Retry</button>
        </p>
      ) : null}
      {!webappsLoaded && !webappsErr && <p className="side-empty" role="status">Loading web apps…</p>}
      {empty ? (
        <p className="side-empty pins-empty">
          {searching ? "No apps match." : "No apps yet."}{" "}
          <button type="button" className="side-empty-act" onClick={() => setAdding(true)}>Add a web app</button>
        </p>
      ) : (
        <SortableList ids={tiles.map((t) => t.id)} grid onReorder={(ids) => { if (!searching) commitOrder(ids); }}>
        <div className="app-grid" role="list">
          {tiles.map((row) => {
            const a = row.tile;
            if (a.kind === "web") return (
              <SortableRow key={a.key} id={a.key}>
              {(drag) => (
              <div
                ref={drag.setNodeRef}
                style={drag.style}
                data-drag-id={a.key}
                className={"app-tile-wrap" + (drag.isDragging ? " is-placeholder" : "")}
                role="listitem"
              >
                <button
                  type="button"
                  className="app-tile"
                  title={a.url}
                  onPointerDown={searching ? undefined : drag.onPointerDown}
                  onClick={() => onOpenWebapp(a)}
                >
                  <span className="app-tile-face">
                    <AppIcon name="globe" label={a.name} size={20} iconUrl={a.hasIcon ? webappIconURL(a.id, iconVersions[a.id]) : null} />
                    {a.badge > 0 ? <span className="app-tile-badge">{a.badge > 99 ? "99+" : a.badge}</span> : null}
                  </span>
                  <span className="app-tile-name">{a.name}</span>
                </button>
                <DropdownMenu.Root>
                  <DropdownMenu.Trigger asChild>
                    <button type="button" className="app-tile-menu" aria-label={"Actions for " + a.name} title="Actions"><IconEllipsis size={13} /></button>
                  </DropdownMenu.Trigger>
                  <DropdownMenu.Portal>
                    <DropdownMenu.Content className="ws-row-menu" side="right" align="start" sideOffset={4} collisionPadding={8}>
                      <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => onOpenWebapp(a)}>Open</DropdownMenu.Item>
                      <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => refresh(a)} disabled={refreshingId === a.id}><IconReload size={13} /> {refreshingId === a.id ? "Refreshing…" : "Refresh"}</DropdownMenu.Item>
                      <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => setAnotherFor(a)}>Add another account</DropdownMenu.Item>
                      <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => setRenaming(a)}><IconPencil size={13} /> Rename</DropdownMenu.Item>
                      {desktop && <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => { setClearError(""); setClearing(a); }}><IconTrash size={13} /> Clear data</DropdownMenu.Item>}
                      <DropdownMenu.Separator className="ws-row-menu-sep" />
                      <DropdownMenu.Item className="ws-row-menu-item danger" onSelect={() => { setRemoveError(""); setRemoving(a); }}><IconTrash size={13} /> Remove</DropdownMenu.Item>
                    </DropdownMenu.Content>
                  </DropdownMenu.Portal>
                </DropdownMenu.Root>
              </div>
              )}
              </SortableRow>
            );
            const tile = appTile(a, nativeApps);
            const ok = tile.ok;
            return (
              <SortableRow key={a.key} id={a.key}>
              {(drag) => (
              <button
                ref={drag.setNodeRef}
                style={drag.style}
                data-drag-id={a.key}
                type="button"
                role="listitem"
                className={"app-tile" + (ok ? "" : " app-tile-unsupported") + (drag.isDragging ? " is-placeholder" : "")}
                title={tile.title}
                onPointerDown={searching ? undefined : drag.onPointerDown}
                onClick={() => { if (ok && onOpen) onOpen(a.id); }}
              >
                <span className="app-tile-face">
                  <AppIcon name={a.icon} label={a.name} size={20} />
                  {a.badge.count > 0 ? <span className="app-tile-badge">{a.badge.count > 99 ? "99+" : a.badge.count}</span> : a.badge.dot ? <span className="app-tile-dot" /> : null}
                </span>
                <span className="app-tile-name">{a.name}</span>
              </button>
              )}
              </SortableRow>
            );
          })}
        </div>
        </SortableList>
      )}
      </div>
      {adding && (
        <WebappDialog onClose={() => setAdding(false)} onSaved={onSavedWebapp} onOpen={onOpenWebapp} />
      )}
      {renaming && (
        <WebappDialog app={renaming} onClose={() => setRenaming(null)} onSaved={onSavedWebapp} onOpen={onOpenWebapp} />
      )}
      {anotherFor && (
        <WebappDialog anotherAccount={anotherFor} onClose={() => setAnotherFor(null)} onSaved={onSavedWebapp} onOpen={onOpenWebapp} />
      )}
      {clearing && (
        <AlertDialog.Root open onOpenChange={(open) => { if (!open && !clearBusy) setClearing(null); }}>
          <AlertDialog.Portal>
            <AlertDialog.Overlay className="dlg-overlay" />
            <AlertDialog.Content className="dlg">
              <AlertDialog.Title className="dlg-title">Clear {clearing.name}'s data</AlertDialog.Title>
              <AlertDialog.Description className="dlg-body">
                Signs you out of {clearing.name} and deletes everything it stored inside PiCode. The app itself stays installed.
              </AlertDialog.Description>
              {clearError && <p className="form-error" role="alert">{clearError}</p>}
              <div className="dlg-actions" data-align-row>
                <AlertDialog.Close className="btn btn-ghost btn-sm" disabled={clearBusy}>Cancel</AlertDialog.Close>
                <button type="button" className="btn btn-danger btn-sm" disabled={clearBusy} onClick={clearData}>{clearBusy ? "Clearing…" : "Clear data"}</button>
              </div>
            </AlertDialog.Content>
          </AlertDialog.Portal>
        </AlertDialog.Root>
      )}
      {removing && (
        <AlertDialog.Root open onOpenChange={(open) => { if (!open && !removeBusy) setRemoving(null); }}>
          <AlertDialog.Portal>
            <AlertDialog.Overlay className="dlg-overlay" />
            <AlertDialog.Content className="dlg">
              <AlertDialog.Title className="dlg-title">Remove web app</AlertDialog.Title>
              <AlertDialog.Description className="dlg-body">
                Remove {removing.name}? Its open tab closes and the shortcut is uninstalled.
              </AlertDialog.Description>
              {removeError && <p className="form-error" role="alert">{removeError}</p>}
              <div className="dlg-actions" data-align-row>
                <AlertDialog.Close className="btn btn-ghost btn-sm" disabled={removeBusy}>Cancel</AlertDialog.Close>
                <button type="button" className="btn btn-danger btn-sm" disabled={removeBusy} onClick={remove}>{removeBusy ? "Removing…" : "Remove"}</button>
              </div>
            </AlertDialog.Content>
          </AlertDialog.Portal>
        </AlertDialog.Root>
      )}
    </div>
  );
}
