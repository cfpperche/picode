import { useEffect, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import InstructionsFix from "./InstructionsFix.jsx";
import { api } from "@picode/shared/client/api.js";
import { STATUS_LABEL, cellTitle, groupFiles, rowMatters, shortPath, sizeLabel, visibleClis } from "@picode/shared/domain/instructions.js";
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
  const [overflow, setOverflow] = useState(false);
  const [fixId, setFixId] = useState("");
  const scroller = useRef(null);
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

  // Whether the matrix is wider than the card: then one line says so and the
  // right edge fades until the last column is in view.
  useEffect(() => {
    const el = scroller.current;
    if (!el || typeof ResizeObserver === "undefined") return;
    const measure = () => {
      setOverflow(el.scrollWidth > el.clientWidth + 2);
      el.classList.toggle("at-end", el.scrollLeft + el.clientWidth >= el.scrollWidth - 2);
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    el.addEventListener("scroll", measure, { passive: true });
    return () => { ro.disconnect(); el.removeEventListener("scroll", measure); };
  });

  const data = state.data;
  const clis = data ? visibleClis(data.clis, showAll) : [];
  const hiddenClis = data ? data.clis.length - clis.length : 0;
  const groups = data ? groupFiles(data.files).map((g) => ({ ...g, rows: g.files.filter((f) => showAll || rowMatters(f, clis)) })).filter((g) => g.rows.length) : [];
  const quietRows = data ? data.files.length - groups.reduce((n, g) => n + g.rows.length, 0) : 0;
  const noProjectFile = data && !data.files.some((f) => f.scope === "project");
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
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button type="button" className="btn instr-refresh">Add personal file</button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content className="um-popover" side="bottom" align="end" sideOffset={6} collisionPadding={12}>
                  {[["CLAUDE.local.md", "Claude Code, Grok"], ["AGENTS.override.md", "Codex, Pi, Hermes"]].map(([name, who]) => (
                    <DropdownMenu.Item key={name} className="um-item" onSelect={() => setFixId("personal:" + (start ? start + "/" : "") + name)}>
                      {name} <span className="instr-muted">· read by {who}</span>
                    </DropdownMenu.Item>
                  ))}
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
            <button className="btn" disabled={state.busy} onClick={() => load()}>Refresh</button>
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

          {noProjectFile ? (
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
                  {f.fix ? (
                    <button className="btn btn-primary" onClick={() => setFixId(f.fix)}>Review change</button>
                  ) : f.action && f.action.kind === "open" ? (
                    <button className="btn" onClick={() => onOpenFile && onOpenFile(f.action.path)}>{f.action.label}</button>
                  ) : f.action && f.action.kind === "settings" ? (
                    <a className="btn" href={f.action.url}>{f.action.label}</a>
                  ) : f.action && f.action.kind === "link" ? (
                    <a className="btn" href={f.action.url} target="_blank" rel="noreferrer">{f.action.label}</a>
                  ) : null}
                </li>
              ))}
            </ul>
          ) : null}

          {groups.length > 0 && overflow ? <p className="instr-more instr-muted">Scroll the table sideways to see all {clis.length} CLIs.</p> : null}
          {groups.length > 0 ? (
            <div className="instr-scroll" ref={scroller}>
              <table className="instr-matrix">
                <thead>
                  <tr>
                    <th scope="col" className="instr-file-col">File</th>
                    {clis.map((c) => (
                      <th key={c.id} scope="col" title={[c.source, ...(c.notes || [])].join(" · ")}>
                        {c.name}{c.installed ? null : <span className="instr-muted"> · not installed</span>}
                      </th>
                    ))}
                  </tr>
                </thead>
                {groups.map((g) => (
                  <tbody key={g.id}>
                    <tr className="instr-group"><th colSpan={clis.length + 1} scope="colgroup"><span>{g.title}</span></th></tr>
                    {g.rows.map((f) => (
                      <tr key={f.path}>
                        <th scope="row" className="instr-file-col">
                          {f.rel ? (
                            <button className="instr-file" title={"Open " + f.path + " · " + sizeLabel(f.bytes)} onClick={() => onOpenFile && onOpenFile(f.rel)}>{f.path}</button>
                          ) : <span className="instr-file-text" title={f.path + " · " + sizeLabel(f.bytes)}>{shortPath(f.path)}</span>}
                          {f.ignored ? <span className="instr-tag" title="Excluded by .gitignore">ignored</span> : null}
                        </th>
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
                                {cell.cut ? <span className="instr-cut" aria-hidden="true" title={cell.cut}>cut</span> : null}
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

          {data && data.agents && data.agents.length > 0 ? (
            <section className="instr-agents" aria-label="What agents here read">
              <h3>What agents here read</h3>
              <ul>
                {data.agents.map((a) => {
                  const cli = data.clis.find((c) => c.id === a.cli);
                  return (
                    <li key={a.terminalId}>
                      <span className="instr-agent-name">{a.name}</span>
                      <span className="instr-muted"> · {cli ? cli.name : a.cli}, latest session</span>
                      <span className="instr-agent-files">
                        {a.files.length === 0 ? <span className="instr-muted">no instruction file</span> : a.files.map((f) => (
                          f.startsWith("/") || f.startsWith("~")
                            ? <span key={f} className="instr-file-text" title={f}>{shortPath(f)}</span>
                            : <button key={f} className="instr-file" title={"Open " + f} onClick={() => onOpenFile && onOpenFile(f)}>{f}</button>
                        ))}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </section>
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
      <InstructionsFix
        workspaceId={workspace.id}
        fixId={fixId}
        onClose={() => setFixId("")}
        onWritten={() => { setFixId(""); load(); }}
      />
    </section>
  );
}
