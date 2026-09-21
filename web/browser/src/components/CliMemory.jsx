import { useCallback, useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import { cliMemoryHash, memoryEmptyLine, memoryKindLabel } from "@picode/shared/domain/cliNative.js";
import { agoLabel, blastRadius, bytesLabel, columnsFor, facets, filterItems, health, indexBudget, sortItems } from "@picode/shared/domain/memoryTable.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

// What an agent CLI has remembered between sessions (ADR-0163). The pane can
// do exactly what the CLI's tier allows: read everything, edit and delete only
// where the vendor treats these files as the user's.
//
// A table rather than a list, because the questions a memory store raises are
// comparisons: which of these is stale, which one grew, which one nothing
// points at any more. The columns are the server's survey; the header sorts
// them; nothing here computes a score of its own.

export default function CliMemory({ route, workspaceId = "" }) {
  const cli = route.id;
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const [scope, setScope] = useState(route.scope || "");
  const [open, setOpen] = useState("");
  const [filter, setFilter] = useState("");
  const [sort, setSort] = useState({ key: "modified", dir: "desc" });
  const [kinds, setKinds] = useState([]);
  const [picked, setPicked] = useState([]);
  const [confirmBulk, setConfirmBulk] = useState(false);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [bulkError, setBulkError] = useState("");

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
  const editable = report.tier === "editable";
  const all = data?.items || [];
  const items = sortItems(filterItems(all, { text: filter, kinds }), sort);
  const columns = columnsFor(all);
  const kindFacets = facets(all);
  const budget = indexBudget(data?.index);
  const indexItem = all.find((i) => i.index);
  const running = data?.running || [];
  const empty = memoryEmptyLine(report, active);
  const narrowed = Boolean(filter.trim() || kinds.length);

  // Selection covers what this pane may actually delete: never the index the
  // CLI loads, never a file the CLI regenerates.
  const selectable = editable ? items.filter((i) => !i.index && !i.generated) : [];
  const chosen = selectable.filter((i) => picked.includes(i.id));
  const allChosen = selectable.length > 0 && chosen.length === selectable.length;
  const citedTotal = chosen.reduce((n, i) => n + (i.citedBy || 0), 0);
  const stillIndexed = chosen.filter((i) => i.indexed).length;

  const toggleKind = (kind) => {
    setPicked([]);
    setKinds((prev) => (prev.includes(kind) ? prev.filter((k) => k !== kind) : [...prev, kind]));
  };

  const toggleSort = (key) => {
    setSort((prev) => (prev.key === key ? { key, dir: prev.dir === "asc" ? "desc" : "asc" } : { key, dir: key === "title" ? "asc" : "desc" }));
  };

  const removeChosen = async () => {
    setBulkBusy(true);
    setBulkError("");
    let done = 0;
    let failure = "";
    for (const item of chosen) {
      try {
        const query = new URLSearchParams({ cli, scope: active?.scope || "", id: item.id });
        if (workspaceId) query.set("workspace", workspaceId);
        await api("/api/cli-memory/item?" + query, { method: "DELETE" });
        done++;
      } catch (err) {
        // Say how far it got: a partial delete the reader cannot see is a lie
        // about what is still on disk.
        failure = err.message;
        break;
      }
    }
    setBulkBusy(false);
    setConfirmBulk(false);
    setPicked([]);
    if (failure) setBulkError(done ? "Deleted " + done + " of " + chosen.length + ", then stopped: " + failure : failure);
    setReload((n) => n + 1);
  };

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
                onClick={() => { setScope(s.scope); setOpen(""); setPicked([]); }}
              >{s.label}</a>
            ))}
          </div>
        ) : <span />}
        {all.length || filter.trim() ? (
          <div className="cli-memory-filter">
            <input
              className="set-text"
              type="search"
              placeholder="Filter"
              value={filter}
              aria-label="Filter memories"
              onChange={(e) => { setFilter(e.target.value); setPicked([]); }}
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

      {budget ? (
        <IndexBudget
          cli={cli}
          budget={budget}
          onOpenIndex={indexItem ? () => setOpen(indexItem.id) : null}
        />
      ) : null}

      {kindFacets.length > 1 ? (
        <div className="cli-memory-facets" role="group" aria-label="Filter by kind">
          {kindFacets.map((f) => (
            <button
              key={f.kind}
              type="button"
              className={"cli-memory-facet" + (kinds.includes(f.kind) ? " is-on" : "")}
              aria-pressed={kinds.includes(f.kind)}
              onClick={() => toggleKind(f.kind)}
            >
              {memoryKindLabel(f.kind) || f.kind}
              <span className="cli-memory-facet-n">{f.count}</span>
            </button>
          ))}
        </div>
      ) : null}

      {bulkError ? <div className="cli-notice is-error" role="alert"><span>{bulkError}</span></div> : null}

      {chosen.length ? (
        <div className="cli-memory-bulk" role="status">
          {/* The cost of the delete is a sentence, not a control: it sits
              above the row so the row stays one line of controls at one
              height, which is what the align auditor holds it to. */}
          {confirmBulk ? <p className="cli-memory-bulk-warn">{blastRadius(chosen.length, citedTotal, stillIndexed)}</p> : null}
          <div className="cli-memory-bulk-row" data-align-row>
            <span className="cli-memory-bulk-n">{chosen.length} selected</span>
            {confirmBulk ? (
              <>
                <button type="button" className="btn btn-danger btn-sm" disabled={bulkBusy} onClick={removeChosen}>
                  {bulkBusy ? "Deleting…" : "Delete " + chosen.length + " for good"}
                </button>
                <button type="button" className="btn btn-ghost btn-sm" disabled={bulkBusy} onClick={() => setConfirmBulk(false)}>Cancel</button>
              </>
            ) : (
              <>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => setConfirmBulk(true)}>Delete selected</button>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => setPicked([])}>Clear selection</button>
              </>
            )}
          </div>
        </div>
      ) : null}

      {data?.itemsError ? (
        <div className="cli-notice" role="status">
          <span>{data.itemsError}</span>
          {report.toggle ? <a className="btn btn-ghost btn-sm" href={cliSettingsHash(cli)}>Open settings</a> : null}
        </div>
      ) : items.length === 0 ? (
        <div className="cli-notice" role="status">
          <span>{narrowed ? "No memory matches this filter." : empty.line}</span>
          {narrowed ? (
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setFilter(""); setKinds([]); }}>Clear filters</button>
          ) : empty.action === "settings" ? (
            <a className="btn btn-ghost btn-sm" href={cliSettingsHash(cli)}>Open settings</a>
          ) : null}
        </div>
      ) : (
        <div className="cli-memory-scroll">
          <table className="cli-memory-table">
            <thead>
              <tr>
                {selectable.length ? (
                  <th className="cli-memory-pick" scope="col">
                    <input
                      type="checkbox"
                      aria-label={allChosen ? "Clear selection" : "Select every memory shown"}
                      checked={allChosen}
                      ref={(el) => { if (el) el.indeterminate = chosen.length > 0 && !allChosen; }}
                      onChange={() => setPicked(allChosen ? [] : selectable.map((i) => i.id))}
                    />
                  </th>
                ) : null}
                {columns.map((column) => (
                  <th
                    key={column.id}
                    scope="col"
                    className={cellClass(column)}
                    aria-sort={column.sort ? (sort.key === column.sort ? (sort.dir === "asc" ? "ascending" : "descending") : "none") : undefined}
                  >
                    {column.sort ? (
                      <button type="button" className="cli-memory-sort" onClick={() => toggleSort(column.sort)}>
                        {column.label}
                        <span aria-hidden="true" className="cli-memory-caret">{sort.key === column.sort ? (sort.dir === "asc" ? "↑" : "↓") : ""}</span>
                      </button>
                    ) : column.label}
                  </th>
                ))}
              </tr>
            </thead>
            {items.map((item) => (
              <MemoryRow
                key={item.id}
                cli={cli}
                item={item}
                columns={columns}
                span={columns.length + (selectable.length ? 1 : 0)}
                scope={active?.scope || ""}
                workspaceId={workspaceId}
                editable={editable && !item.generated}
                pickable={selectable.some((i) => i.id === item.id)}
                picked={picked.includes(item.id)}
                onPick={() => setPicked((prev) => (prev.includes(item.id) ? prev.filter((id) => id !== item.id) : [...prev, item.id]))}
                open={open === item.id}
                onToggle={() => setOpen(open === item.id ? "" : item.id)}
                onChanged={() => setReload((n) => n + 1)}
              />
            ))}
          </table>
        </div>
      )}
    </div>
  );
}

// The index is the file the CLI reads at the start of every session, and it
// reads only so much of it. Past the limit the rest is dropped in silence,
// which is the one thing about a memory store nobody can see from the CLI.
function IndexBudget({ cli, budget, onOpenIndex }) {
  const limited = Boolean(budget.byteLimit || budget.lineLimit);
  const label = terminalCliLabel(cli);
  const tone = budget.over ? " is-over" : budget.near ? " is-near" : "";
  return (
    <div className={"cli-memory-budget" + tone} role="status">
      {limited ? (
        <div className="cli-memory-budget-bar" aria-hidden="true"><span style={{ width: budget.percent + "%" }} /></div>
      ) : null}
      <div className="cli-memory-budget-text">
        <span>
          {limited ? (
            <>
              The index is at <strong>{budget.percent}%</strong> of what {label} reads each session
              {" — "}{budget.lines} of {budget.lineLimit} lines, {bytesLabel(budget.bytes)} of {bytesLabel(budget.byteLimit)}.
              {budget.over ? " Everything past the limit is dropped: shorten it." : budget.near ? " Near the limit; shorten it before it starts dropping lines." : ""}
            </>
          ) : (
            <>The index is {budget.lines} lines, {bytesLabel(budget.bytes)}.</>
          )}
          {budget.dangling.length ? (
            <> {budget.dangling.length === 1 ? "One row points at a file that is gone" : budget.dangling.length + " rows point at files that are gone"} ({budget.dangling.join(", ")}).</>
          ) : null}
        </span>
        {onOpenIndex ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={onOpenIndex}>Open the index</button>
        ) : null}
      </div>
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

function cellClass(column) {
  return "cli-memory-c-" + column.id + (column.numeric ? " is-num" : "");
}

function MemoryRow({ cli, item, columns, span, scope, workspaceId, editable, pickable, picked, onPick, open, onToggle, onChanged }) {
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
  const chips = health(item);
  return (
    <tbody className={"cli-memory-item" + (open ? " is-open" : "") + (item.index ? " is-index" : "")}>
      <tr className="cli-memory-head" onClick={onToggle}>
        {span > columns.length ? (
          <td className="cli-memory-pick" onClick={(e) => e.stopPropagation()}>
            {pickable ? (
              <input
                type="checkbox"
                checked={picked}
                aria-label={"Select " + item.title}
                onChange={onPick}
              />
            ) : null}
          </td>
        ) : null}
        {columns.map((column) => (
          <td key={column.id} className={cellClass(column)}>
            {column.id === "memory" ? (
              <button type="button" className="cli-memory-open" aria-expanded={open} onClick={(e) => { e.stopPropagation(); onToggle(); }}>
                <span className="cli-memory-title">{item.title}</span>
                {item.index ? <span className="cli-memory-tag">Loaded every session</span> : null}
                {kind ? <span className="cli-memory-tag">{kind}</span> : null}
                {item.summary ? <span className="cli-memory-sum">{item.summary}</span> : null}
              </button>
            ) : column.id === "citedBy" ? (
              item.index ? "" : (
                <span
                  className={item.citedBy === 0 ? "is-zero" : ""}
                  title={item.citedBy === 0 ? "No other memory points at this one — a candidate to merge or drop." : undefined}
                >{item.citedBy ?? ""}</span>
              )
            ) : column.id === "bytes" ? (
              bytesLabel(item.bytes)
            ) : column.id === "modified" ? (
              <span title={modifiedTitle(item)}>{agoLabel(item.modified)}</span>
            ) : column.id === "health" ? (
              chips.length ? chips.map((chip) => (
                <span key={chip.id} className={"cli-memory-chip is-" + chip.tone} title={chip.detail || undefined}>{chip.label}</span>
              )) : ""
            ) : ""}
          </td>
        ))}
      </tr>
      {open ? (
        <tr className="cli-memory-open-row">
          <td colSpan={span}>
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
          </td>
        </tr>
      ) : null}
    </tbody>
  );
}

// Where the date came from, said plainly: a memory the CLI stamped itself is
// worth more than a file whose mtime a backup touched.
function modifiedTitle(item) {
  if (!item.modified) return "";
  const from = item.modifiedFrom === "memory" ? " (the date the CLI wrote into the file)" : " (when the file was last written)";
  return item.modified + from;
}
