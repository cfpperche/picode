import { useCallback, useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import { cliMemoryHash, memoryEmptyLine, memoryKindLabel } from "@picode/shared/domain/cliNative.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

// What an agent CLI has remembered between sessions (ADR-0163). The pane can
// do exactly what the CLI's tier allows: read everything, edit and delete only
// where the vendor treats these files as the user's.

export default function CliMemory({ route, workspaceId = "" }) {
  const cli = route.id;
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const [scope, setScope] = useState(route.scope || "");
  const [open, setOpen] = useState("");
  const [filter, setFilter] = useState("");

  const load = useCallback(() => {
    const query = new URLSearchParams({ cli });
    if (workspaceId) query.set("workspace", workspaceId);
    if (scope) query.set("scope", scope);
    return api("/api/cli-memory?" + query);
  }, [cli, workspaceId, scope]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    load().then((next) => {
      if (!active) return;
      setData(next);
      setError(null);
      if (!scope && next.scope) setScope(next.scope);
    }).catch((err) => {
      if (active) setError(err);
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [load, reload]);

  useEffect(() => {
    let timer;
    const stop = subscribeFeed((event) => {
      if ((event.type === "cli.memory" || event.type === "cli.settings") && event.data?.cli === cli) {
        clearTimeout(timer);
        timer = setTimeout(() => setReload((n) => n + 1), 80);
      }
    });
    return () => { clearTimeout(timer); stop(); };
  }, [cli]);

  if (loading && !data) {
    return <div className="cli-loading" aria-label={"Loading " + terminalCliLabel(cli) + " memory"}><div /><div /><div /></div>;
  }
  if (error) {
    return (
      <div className="cli-notice is-error" role="alert">
        <span>{error.message}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setReload((n) => n + 1)}>Try again</button>
      </div>
    );
  }

  const report = data?.report || {};
  const stores = report.stores || [];
  const active = stores.find((s) => s.scope === (data?.scope || scope)) || stores[0];
  const items = (data?.items || []).filter((item) => {
    if (!filter.trim()) return true;
    const needle = filter.trim().toLowerCase();
    return (item.title + " " + (item.summary || "") + " " + item.id).toLowerCase().includes(needle);
  });
  const editable = report.tier === "editable";
  const running = data?.running || [];
  const empty = memoryEmptyLine(report, active);

  // A CLI with no memory is one line and, where there is one, the action that
  // would change it. Never an empty well.
  if (report.tier === "none" || report.tier === "unknown") {
    return <div className="cli-notice" role="status"><span>{report.note}</span></div>;
  }

  return (
    <div className="cli-memory">
      <div className="cli-memory-bar" data-align-row>
        {stores.length > 1 ? (
          <div className="pkg-scope" role="radiogroup" aria-label="Which memory to show">
            {stores.map((s) => (
              <a
                key={s.scope}
                className="pkg-scope-btn"
                role="radio"
                aria-checked={s.scope === active?.scope}
                href={cliMemoryHash(cli, { workspaceId, scope: s.scope })}
                onClick={() => { setScope(s.scope); setOpen(""); }}
              >{s.label}</a>
            ))}
          </div>
        ) : <span />}
        {(data?.items || []).length || filter.trim() ? (
          <div className="cli-memory-filter">
            <input
              className="set-text"
              type="search"
              placeholder="Filter"
              value={filter}
              aria-label="Filter memories"
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>
        ) : <span />}
      </div>

      {active?.path ? <p className="settings-file">Kept in {active.path}</p> : null}

      {!editable ? (
        <div className="cli-notice" role="status">
          <span>{report.note}</span>
          {report.clear ? <CopyCommand command={report.clear} /> : null}
        </div>
      ) : null}

      {running.length && editable ? (
        <div className="cli-notice" role="status">
          <span>{terminalCliLabel(cli)} is running in {running.map((t) => t.name).join(", ")} and may write these files too.</span>
          <a className="btn btn-ghost btn-sm" href={"#/term/" + running[0].id}>Open terminal</a>
        </div>
      ) : null}

      {data?.itemsError ? (
        <div className="cli-notice" role="status">
          <span>{data.itemsError}</span>
          {report.toggle ? <a className="btn btn-ghost btn-sm" href={cliSettingsHash(cli)}>Open settings</a> : null}
        </div>
      ) : items.length === 0 ? (
        <div className="cli-notice" role="status">
          <span>{filter.trim() ? "No memory matches this filter." : empty.line}</span>
          {!filter.trim() && empty.action === "settings" ? (
            <a className="btn btn-ghost btn-sm" href={cliSettingsHash(cli)}>Open settings</a>
          ) : null}
        </div>
      ) : (
        <ul className="cli-memory-list">
          {items.map((item) => (
            <MemoryRow
              key={item.id}
              cli={cli}
              item={item}
              scope={active?.scope || ""}
              workspaceId={workspaceId}
              editable={editable && !item.generated}
              open={open === item.id}
              onToggle={() => setOpen(open === item.id ? "" : item.id)}
              onChanged={() => setReload((n) => n + 1)}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

// The vendor's own command, on the clipboard. Chrome carries state and the
// next action; handing a terminal-averse reader a string to retype is neither.
function CopyCommand({ command }) {
  const [state, setState] = useState("");
  return (
    <button
      type="button"
      className="btn btn-ghost btn-sm"
      onClick={async () => {
        // A clipboard a browser refuses (no permission, insecure origin) must
        // not read as a dead click: the button says so and keeps the command
        // on its face so it can still be read and typed.
        try {
          await navigator.clipboard.writeText(command);
          setState("Copied");
        } catch {
          setState("Copy it by hand");
        }
        setTimeout(() => setState(""), 2000);
      }}
    >{state || "Copy " + command}</button>
  );
}

function MemoryRow({ cli, item, scope, workspaceId, editable, open, onToggle, onChanged }) {
  const [body, setBody] = useState(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirming, setConfirming] = useState(false);

  useEffect(() => {
    if (!open || body !== null) return;
    let active = true;
    const query = new URLSearchParams({ cli, scope, id: item.id });
    if (workspaceId) query.set("workspace", workspaceId);
    api("/api/cli-memory/item?" + query).then((res) => {
      if (!active) return;
      setBody(res.body || "");
      setDraft(res.body || "");
      setError("");
    }).catch((err) => { if (active) setError(err.message); });
    return () => { active = false; };
  }, [open, body, cli, scope, item.id, workspaceId]);

  const save = async () => {
    setBusy(true);
    setError("");
    try {
      await api("/api/cli-memory/item", { method: "PUT", body: { cli, workspaceId, scope, id: item.id, body: draft } });
      setBody(draft);
      onChanged();
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    setError("");
    try {
      const query = new URLSearchParams({ cli, scope, id: item.id });
      if (workspaceId) query.set("workspace", workspaceId);
      await api("/api/cli-memory/item?" + query, { method: "DELETE" });
      onChanged();
    } catch (err) {
      setError(err.message);
      setBusy(false);
    }
  };

  const kind = memoryKindLabel(item.kind);
  return (
    <li className={"cli-memory-item" + (open ? " is-open" : "")}>
      <button type="button" className="cli-memory-head" aria-expanded={open} onClick={onToggle}>
        <span className="cli-memory-title">{item.title}</span>
        {item.index ? <span className="cli-memory-tag">Loaded every session</span> : null}
        {kind ? <span className="cli-memory-tag">{kind}</span> : null}
        {item.summary && !open ? <span className="cli-memory-sum">{item.summary}</span> : null}
      </button>
      {open ? (
        <div className="cli-memory-body">
          {error ? <div className="cli-notice is-error" role="alert"><span>{error}</span></div> : null}
          {body === null ? (
            <div className="cli-loading" aria-label="Loading memory"><div /><div /><div /></div>
          ) : editable ? (
            <>
              <span className="cli-memory-label">Editing {item.id}</span>
              <textarea className="cli-memory-edit" value={draft} disabled={busy} onChange={(e) => setDraft(e.target.value)} />
              <div className="cli-memory-actions" data-align-row>
                <button type="button" className="btn btn-sm" disabled={busy || draft === body} onClick={save}>Save</button>
                <button type="button" className="btn btn-ghost btn-sm" disabled={busy || draft === body} onClick={() => setDraft(body)}>Discard</button>
                {item.index ? null : confirming ? (
                  <>
                    <button type="button" className="btn btn-danger btn-sm" disabled={busy} onClick={remove}>Delete for good</button>
                    <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => setConfirming(false)}>Cancel</button>
                  </>
                ) : (
                  <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => setConfirming(true)}>Delete</button>
                )}
              </div>
            </>
          ) : (
            <pre className="cli-memory-text">{body}</pre>
          )}
        </div>
      ) : null}
    </li>
  );
}
