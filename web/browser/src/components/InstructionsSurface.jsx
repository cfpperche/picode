import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { STATUS_LABEL, cellTitle, groupFiles, rowMatters, sizeLabel, visibleClis } from "@picode/shared/domain/instructions.js";
import "../styles/instructions.css";

// Instructions: which instruction files each agent CLI reads for a session
// started in this workspace (docs/architecture/cli-instructions.md). A
// workspace tab that draws the page frame every system route draws
// (.settings-wrap + .settings-head + .settings-card, ADR-0103), like Git ▸
// Delivery. Read-only: a finding opens the file in the editor, nothing here
// writes. Files change outside PiCode's store, so the view re-reads when it
// is shown and on Refresh — no feed event covers them and no timer polls.
export default function InstructionsSurface({ workspace, hidden, onOpenFile }) {
  const [start, setStart] = useState("");
  const [showAll, setShowAll] = useState(false);
  const [picked, setPicked] = useState(null);
  const [state, setState] = useState({ data: null, busy: true, error: "" });
  const seq = useRef(0);

  function load(from = start) {
    const n = ++seq.current;
    setState((cur) => ({ ...cur, busy: true, error: "" }));
    const q = from ? "?start=" + encodeURIComponent(from) : "";
    api("/api/workspaces/" + encodeURIComponent(workspace.id) + "/instructions" + q)
      .then((data) => { if (n === seq.current) setState({ data, busy: false, error: "" }); })
      .catch((e) => { if (n === seq.current) setState((cur) => ({ data: cur.data, busy: false, error: e.message || "failed" })); });
  }

  useEffect(() => {
    if (!hidden) load();
  }, [hidden, workspace.id, start]);

  const data = state.data;
  const clis = data ? visibleClis(data.clis, showAll) : [];
  const hiddenClis = data ? data.clis.length - clis.length : 0;
  const groups = data ? groupFiles(data.files).map((g) => ({ ...g, rows: g.files.filter((f) => showAll || rowMatters(f, clis)) })).filter((g) => g.rows.length) : [];
  const quietRows = data ? data.files.length - groups.reduce((n, g) => n + g.rows.length, 0) : 0;
  const pickedFile = picked && data ? data.files.find((f) => f.path === picked.path) : null;
  const pickedCli = picked && data ? data.clis.find((c) => c.id === picked.cli) : null;
  const pickedCell = pickedFile && pickedCli ? pickedFile.cells[pickedCli.id] : null;

  return (
    <section className="instr-page" aria-label="Instructions" hidden={!!hidden}>
      <div className="settings-wrap">
        <header className="settings-head"><h2>Instructions</h2></header>
        <div className="settings-card">
          <div className="instr-toolbar" data-align-row>
            {data && data.folders.length > 1 ? (
              <label data-align-row>
                Start in
                <select aria-label="Start folder" value={start} onChange={(e) => { setPicked(null); setStart(e.target.value); }}>
                  {data.folders.map((f) => <option key={f} value={f}>{f ? f + "/" : "Workspace folder"}</option>)}
                </select>
              </label>
            ) : null}
            {data && (hiddenClis > 0 || quietRows > 0 || showAll) ? (
              <label className="instr-toggle" data-align-row>
                <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} />
                Show every CLI and file
              </label>
            ) : null}
            <button className="btn instr-refresh" disabled={state.busy} onClick={() => load()}>Refresh</button>
          </div>

          {state.error ? (
            <div className="instr-warning" role="alert">
              <p>{data ? "Could not update; showing the last read." : state.error === "This workspace's folder is gone." ? state.error : "Could not read this workspace's instructions."}</p>
              <button className="btn" onClick={() => load()}>Retry</button>
            </div>
          ) : null}

          {!data && state.busy ? (
            <div className="instr-skeleton" aria-label="Reading instruction files">{[1, 2, 3, 4].map((n) => <div key={n} />)}</div>
          ) : null}

          {data && data.files.length === 0 ? (
            <div className="mcp-empty">
              <p>No instruction files in this workspace.</p>
              <a className="btn" href="https://agents.md" target="_blank" rel="noreferrer">What is AGENTS.md?</a>
            </div>
          ) : null}

          {data && data.findings.length > 0 ? (
            <ul className="instr-findings" aria-label="Findings">
              {data.findings.map((f, i) => (
                <li key={f.id + i}>
                  <span>{f.text}</span>
                  {f.action && f.action.kind === "open" ? (
                    <button className="btn" onClick={() => onOpenFile && onOpenFile(f.action.path)}>{f.action.label}</button>
                  ) : f.action && f.action.kind === "link" ? (
                    <a className="btn" href={f.action.url} target="_blank" rel="noreferrer">{f.action.label}</a>
                  ) : null}
                </li>
              ))}
            </ul>
          ) : null}

          {groups.length > 0 ? (
            <div className="instr-scroll">
              <table className="instr-matrix">
                <thead>
                  <tr>
                    <th scope="col" className="instr-file-col">File</th>
                    <th scope="col" className="instr-num">Size</th>
                    {clis.map((c) => (
                      <th key={c.id} scope="col" title={[c.source, ...(c.notes || [])].join(" · ")}>
                        {c.name}{c.installed ? null : <span className="instr-muted"> · not installed</span>}
                      </th>
                    ))}
                  </tr>
                </thead>
                {groups.map((g) => (
                  <tbody key={g.id}>
                    <tr className="instr-group"><th colSpan={clis.length + 2} scope="colgroup">{g.title}</th></tr>
                    {g.rows.map((f) => (
                      <tr key={f.path}>
                        <th scope="row" className="instr-file-col">
                          {f.rel ? (
                            <button className="instr-file" title={"Open " + f.path} onClick={() => onOpenFile && onOpenFile(f.rel)}>{f.path}</button>
                          ) : <span className="instr-file-text" title={f.path}>{f.path}</span>}
                          {f.ignored ? <span className="instr-tag" title="Excluded by .gitignore">ignored</span> : null}
                        </th>
                        <td className="instr-num">{sizeLabel(f.bytes)}</td>
                        {clis.map((c) => {
                          const cell = f.cells[c.id] || { status: "unknown" };
                          const on = picked && picked.path === f.path && picked.cli === c.id;
                          return (
                            <td key={c.id}>
                              <button
                                className={"instr-cell s-" + cell.status + (on ? " is-on" : "")}
                                title={cellTitle(cell)}
                                aria-label={c.name + ", " + f.path + ": " + (cell.status === "not-read" ? "not read" : STATUS_LABEL[cell.status] || cell.status)}
                                aria-pressed={on}
                                onClick={() => setPicked(on ? null : { path: f.path, cli: c.id })}
                              >
                                {STATUS_LABEL[cell.status] || cell.status}
                                {cell.cut ? <span className="instr-cut" aria-hidden="true">cut</span> : null}
                              </button>
                            </td>
                          );
                        })}
                      </tr>
                    ))}
                  </tbody>
                ))}
              </table>
            </div>
          ) : null}

          {pickedCell ? (
            <p className="instr-detail" role="status">
              <strong>{pickedCli.name} · {pickedFile.path}</strong> — {cellTitle(pickedCell)}
            </p>
          ) : data && groups.length > 0 ? (
            <p className="instr-detail instr-muted" role="status">Pick a cell to see why.</p>
          ) : null}
        </div>
      </div>
    </section>
  );
}
