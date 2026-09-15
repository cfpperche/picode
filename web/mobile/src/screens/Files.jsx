import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { createDocumentGuard, createFileDocument } from "../lib/fileDocument.js";
import { fileMessage, ownerFileURL, readFile } from "../lib/fileIO.js";
import { folderPage, parentFolder, searchFileFolders } from "../lib/fileBrowser.js";
import { previewKind } from "@picode/shared/domain/filePreview.js";
import { useLaunchGuard } from "../components/CliLaunchSettings.jsx";
import { askConfirm } from "../lib/confirm.js";
import ScreenHeader from "../components/ScreenHeader.jsx";
import FileDocument from "../components/FileDocument.jsx";
import FileLeaveDialog from "../components/FileLeaveDialog.jsx";
import * as Sheet from "../components/MobileSheet.jsx";
import { IconFile, IconFolder, IconChevronRight, IconMore } from "../components/Icons.jsx";
import "../styles/mobile-files.css";

const EMPTY = { kind: "load", dirty: false, saving: false, refreshing: false, error: "" };
const subscribeEmpty = () => () => {};
const emptySnapshot = () => EMPTY;
const moved = error => /folder changed/i.test(error || "");

// A phone uses one tool at a time: folders -> document -> folder. The
// document and pinned working folder share every navigation guard (0074).
export default function Files(props) {
  return <FileScreen key={props.owner.kind + ":" + props.owner.id} {...props} />;
}

function FileScreen({ owner, title, initialPath = "", root: initialRoot = "", onPathChange, onBack, onOpenGit }) {
  const [root, setRoot] = useState(initialRoot);
  const [path, setPath] = useState(initialPath);
  const [folder, setFolder] = useState(null);
  const [folderError, setFolderError] = useState("");
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState({ results: [], loading: false, error: "" });
  const [options, setOptions] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const leaveResolver = useRef(null), browseFlight = useRef(null), browseGeneration = useRef(0), navigation = useRef(0);
  const stateRef = useRef(null);
  stateRef.current = { owner, root, path, folder };
  const callbacks = useRef(null);
  callbacks.current = { onPathChange, onBack, onOpenGit };

  const doc = useMemo(() => path && root ? createFileDocument({
    read: signal => readFile(owner, path, root, signal),
    write: (text, mtime) => api(ownerFileURL(owner, "text", "", root), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ path, text, mtime }) }),
    release: page => { if (page.src) URL.revokeObjectURL(page.src); },
  }) : null, [owner.kind, owner.id, path, root]);
  const documentRef = useRef(doc); documentRef.current = doc;
  const view = useSyncExternalStore(doc?.subscribe || subscribeEmpty, doc?.getSnapshot || emptySnapshot);
  const guard = useMemo(() => doc ? createDocumentGuard(doc, () => new Promise(resolve => { leaveResolver.current = resolve; setLeaving(true); })) : async () => true, [doc]);
  const allowNavigation = useLaunchGuard(view.dirty || view.saving, guard);
  const guardRef = useRef(guard); guardRef.current = guard;

  function pickLeave(choice) {
    const resolve = leaveResolver.current;
    leaveResolver.current = null;
    setLeaving(false);
    resolve?.(choice);
  }
  async function navigate(action, external = false) {
    const generation = ++navigation.current;
    if (await guardRef.current() && generation === navigation.current) {
      // Local path/root changes use replaceState. A failed async Follow
      // must leave the dirty hash guard armed for the next navigation.
      if (external) allowNavigation();
      await action();
    }
  }
  function selectFile(next) {
    if (next === path) return;
    return navigate(() => { setPath(next); setOptions(false); callbacks.current.onPathChange?.(next, root); });
  }

  const browse = useCallback(async (dir = "", follow = false) => {
    browseFlight.current?.abort();
    const signal = new AbortController();
    browseFlight.current = signal;
    const generation = ++browseGeneration.current;
    const current = stateRef.current;
    const document = documentRef.current;
    const text = document?.getSnapshot().text;
    setLoading(true);
    try {
      const page = folderPage(await api(ownerFileURL(current.owner, "browse", dir, follow ? "" : current.root), { signal: signal.signal }));
      if (generation !== browseGeneration.current) return false;
      if (follow && (document !== documentRef.current || text !== document?.getSnapshot().text)) {
        setFolderError("Your edits changed while opening the folder. Try again.");
        return false;
      }
      setRoot(page.root);
      setFolder(page);
      setFolderError("");
      if (follow || page.dir !== current.folder?.dir || page.root !== current.root) setQuery("");
      if (follow) {
        setPath("");
        callbacks.current.onPathChange?.("", page.root);
      } else if (!current.root) callbacks.current.onPathChange?.(current.path, page.root);
      return true;
    } catch (error) {
      if (generation === browseGeneration.current && !signal.signal.aborted) setFolderError(error.message || "Could not read this folder.");
      return false;
    } finally {
      if (generation === browseGeneration.current) setLoading(false);
    }
  }, []);
  useEffect(() => {
    void browse(initialPath ? parentFolder(initialPath) : "");
    return () => { browseGeneration.current++; navigation.current++; browseFlight.current?.abort(); leaveResolver.current?.("cancel"); };
  }, [browse]);
  // Hash navigation is guarded before the shell changes this prop.
  useEffect(() => { setPath(initialPath); }, [initialPath]);
  useEffect(() => { if (doc) void doc.refresh(); return () => doc?.dispose(); }, [doc]);

  useEffect(() => {
    const refresh = () => {
      if (document.hidden) return;
      if (doc) void doc.refresh();
      else if (stateRef.current.folder) void browse(stateRef.current.folder.dir);
    };
    window.addEventListener("focus", refresh);
    const off = subscribeFeed(event => {
      if (event.type === "feed.open" || event.type === "feed.reset") refresh();
      else if (event.type === "git.updated" && event.data?.path === root) refresh();
    });
    return () => { window.removeEventListener("focus", refresh); off(); };
  }, [doc, browse, root]);

  useEffect(() => {
    if (!query.trim() || !root || !folder) { setSearch({ results: [], loading: false, error: "" }); return; }
    const reader = new AbortController();
    const timer = setTimeout(async () => {
      setSearch({ results: [], loading: true, error: "" });
      try {
        const found = await searchFileFolders({
          read: async (dir, signal) => folderPage(await api(ownerFileURL(owner, "browse", dir, root), { signal })),
          dir: folder.dir, query, signal: reader.signal,
        });
        if (!reader.signal.aborted) setSearch({ ...found, loading: false, error: "" });
      } catch (error) {
        if (!reader.signal.aborted) setSearch({ results: [], loading: false, error: error.message || "Could not search this folder." });
      }
    }, 200);
    return () => { clearTimeout(timer); reader.abort(); };
  }, [query, root, folder, owner.kind, owner.id]);

  async function reloadFile() {
    if (!doc || view.saving) return;
    if (doc.getSnapshot().dirty && !await askConfirm({ title: "Reload file?", message: "Discard your edits and read the file again?", confirmLabel: "Reload", danger: true })) return;
    await doc.refresh({ discard: true });
  }
  const follow = () => navigate(() => browse("", true));
  const back = () => path ? selectFile("") : navigate(() => callbacks.current.onBack?.(), true);
  const openGit = () => navigate(() => callbacks.current.onOpenGit?.(owner, root), true);
  const error = view.error || folderError;
  const fileError = !!view.error && path && doc;
  // A too-large HTML page previews from disk (FileDocument owns it): the
  // "too large" notice and its Browse folder answer would be wrong for it.
  const previewOnly = fileError && previewKind(path) === "html" && /too large/i.test(error || "");
  const unavailablePreview = fileError && view.kind === "msg" && !previewOnly && /too large|can't show|can't write|unsupported/i.test(error);
  const errorAction = moved(error) ? follow : unavailablePreview ? () => selectFile("") : fileError ? view.dirty && !/changed on disk/i.test(error) ? () => doc.save() : reloadFile : () => browse(folder?.dir || parentFolder(path));
  const errorLabel = moved(error) ? "Follow folder" : unavailablePreview ? "Browse folder" : fileError ? view.dirty && !/changed on disk/i.test(error) ? "Retry save" : "Reload" : "Try again";
  const searching = !!query.trim();
  const rows = searching ? search.results : folder ? [...folder.dirs.map(row => ({ ...row, isDir: true })), ...folder.files] : [];
  function option(action) { setOptions(false); void action(); }

  return <div className="m-files-screen" aria-busy={view.saving || loading || view.refreshing}>
    <ScreenHeader title={path ? path.split("/").at(-1) : "Files"} sub={path ? view.saving ? "Saving…" : view.dirty ? "Unsaved changes" : "" : title} onBack={back} right={<>
      {view.kind === "text" && path ? <button type="button" className="btn btn-primary btn-sm" disabled={!view.dirty || view.saving} onClick={() => doc.save()}>Save</button> : null}
      <button type="button" className="btn btn-ghost btn-sm m-file-options" aria-label="File actions" onClick={() => setOptions(true)}><IconMore /></button>
    </>} />
    {view.saving ? <div className="m-file-saving" role="progressbar" aria-label="Saving file" /> : null}
    {error && !previewOnly ? <div className="m-file-notice" role="alert"><p>{moved(error) ? "The working folder changed." : fileMessage(error, fileError ? "file" : "folder")}</p><button type="button" className="btn btn-sm" disabled={view.saving || loading || view.refreshing} onClick={errorAction}>{errorLabel}</button></div> : null}
    {path ? <>
      <div className="m-file-path" title={root + "/" + path}>{path}</div>
      <FileDocument doc={doc} view={!doc && folderError ? { kind: "msg" } : view} path={path} owner={owner} root={root} />
    </> : <div className="m-files-browser">
      <div className="m-file-folder-bar" data-align-row>
        <button type="button" className="btn btn-sm btn-ghost" disabled={!folder?.dir || loading} onClick={() => browse(folder.parent)}>Up</button>
        <span title={root + (folder?.dir ? "/" + folder.dir : "")}>{folder?.dir || title || "Working folder"}</span>
        <button type="button" className="btn btn-sm btn-ghost" disabled={loading} onClick={() => browse(folder?.dir || "")}>{loading ? "Refreshing…" : "Refresh"}</button>
      </div>
      <div className="m-file-search" data-align-row><input type="search" aria-label="Find files in this folder" placeholder="Find files in this folder…" value={query} onChange={event => setQuery(event.target.value)} /><button type="button" className="btn btn-sm" disabled={!query} onClick={() => setQuery("")}>Clear</button></div>
      {search.error ? <div className="m-file-notice" role="alert"><p>{moved(search.error) ? "The working folder changed." : fileMessage(search.error, "folder")}</p><button type="button" className="btn btn-sm" onClick={moved(search.error) ? follow : () => setQuery("")}>{moved(search.error) ? "Follow folder" : "Clear search"}</button></div> : null}
      {search.limited || search.skipped ? <p className="m-file-search-note" role="status">{search.limited ? "Search limit reached. Open a folder to narrow the search." : "Some folders could not be searched."}</p> : null}
      {(!folder && loading) || (searching && search.loading) ? <div className="m-files-loading" role="status" aria-label={searching ? "Searching files" : "Loading folder"}><span /><span /><span /></div> : null}
      <div className="m-files-list">
        {rows.length ? <ul aria-label={searching ? "Search results" : "Folder contents"}>{rows.map(row => <li key={row.path}><button type="button" className={row.ignored ? "is-ignored" : ""} onClick={() => row.isDir ? browse(row.path) : selectFile(row.path)} title={row.path}>
          {row.isDir ? <IconFolder /> : <IconFile />}<span>{searching ? row.path : row.name}</span>{row.isDir ? <IconChevronRight /> : null}
        </button></li>)}</ul> : !loading && !search.loading && !error && !search.error ? <div className="m-file-empty"><p>{searching ? "No matching files." : "This folder is empty."}</p><button type="button" className="btn btn-sm" onClick={searching ? () => setQuery("") : () => browse(folder?.dir || "")}>{searching ? "Clear search" : "Refresh"}</button></div> : null}
      </div>
    </div>}
    <Sheet.Root open={options} onOpenChange={setOptions}><Sheet.Portal><Sheet.Overlay className="dlg-overlay" /><Sheet.Content className="dlg m-file-sheet" onCloseAutoFocus={event => event.preventDefault()}>
      <Sheet.Title className="dlg-title">File actions</Sheet.Title><Sheet.Description className="dlg-body m-file-root">{root || "Working folder"}</Sheet.Description>
      {path ? <><button type="button" className="btn" disabled={!doc || view.saving || view.refreshing} onClick={() => option(reloadFile)}>Reload file</button><button type="button" className="btn" onClick={() => option(() => selectFile(""))}>Browse folder</button></> : null}
      {onOpenGit ? <button type="button" className="btn" onClick={() => option(openGit)}>Open Git</button> : null}
      <button type="button" className="btn" onClick={() => option(follow)}>Follow working folder</button>
      <button type="button" className="btn btn-primary" onClick={() => setOptions(false)}>Done</button>
    </Sheet.Content></Sheet.Portal></Sheet.Root>
    <FileLeaveDialog open={leaving} path={path} onPick={pickLeave} />
  </div>;
}
