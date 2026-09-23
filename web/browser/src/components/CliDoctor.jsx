import { useEffect, useMemo, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { filterEntries, formatValue, sortFindings, sourceLabel } from "@picode/shared/domain/cliDoctor.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

// Checks for one CLI in one folder (slice 4): the problems PiCode measured,
// each one line and one action, and — behind one deliberate reveal — every
// setting the CLI resolves here with the file it comes from. Read-only: the
// fixes are the rows below it, or the file itself.

export default function CliDoctor({ cli, workspaceId = "", workspaceName = "" }) {
  const label = terminalCliLabel(cli);
  const [rep, setRep] = useState(null);
  const [error, setError] = useState(null);
  const [tick, setTick] = useState(0);
  const [query, setQuery] = useState("");
  const [onlySet, setOnlySet] = useState(true);

  const q = useMemo(() => {
    const p = new URLSearchParams({ cli });
    if (workspaceId) p.set("workspace", workspaceId);
    return p.toString();
  }, [cli, workspaceId]);

  useEffect(() => {
    let active = true;
    api("/api/cli-doctor?" + q)
      .then((r) => { if (active) { setRep(r); setError(null); } })
      .catch((err) => { if (active) setError(err); });
    return () => { active = false; };
  }, [q, tick]);

  // A save in this pane or another changes what the checks say.
  useEffect(() => {
    let timer;
    const stop = subscribeFeed((event) => {
      if (event.type === "cli.settings" && event.data?.cli === cli) {
        clearTimeout(timer);
        timer = setTimeout(() => setTick((n) => n + 1), 120);
      }
    });
    return () => { clearTimeout(timer); stop(); };
  }, [cli]);

  const goToSetting = (key) => {
    const row = document.querySelector('#cli-settings-view [data-key="' + CSS.escape(key) + '"]');
    if (row) {
      row.scrollIntoView({ block: "center" });
      row.querySelector("select, input, button")?.focus();
    }
  };

  if (error) {
    return (
      <section className="doctor-card" aria-label="Checks">
        <div className="cli-notice is-error" role="alert">
          <span>{error.message}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setTick((n) => n + 1)}>Try again</button>
        </div>
      </section>
    );
  }
  if (!rep) {
    return <section className="doctor-card" aria-label="Checks"><div className="cli-loading" aria-label={"Checking " + label}><div /><div /></div></section>;
  }

  const findings = sortFindings(rep.findings || []);
  const entries = filterEntries(rep.entries || [], { query, onlySet });
  const setCount = (rep.entries || []).filter((e) => e.source).length;

  return (
    <section className="doctor-card" aria-label="Checks">
      <h3>Checks</h3>
      {findings.length === 0 ? (
        <p className="set-src">No problems found in {label}'s files for this folder.</p>
      ) : (
        <ul className="doctor-list">
          {findings.map((f) => (
            <li key={f.id + (f.file || "")} className={"doctor-item is-" + f.severity}>
              <span className="doctor-mark" aria-hidden="true">{f.severity === "warn" ? "!" : "i"}</span>
              <span className="doctor-text">{f.text}</span>
              {f.setting ? (
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => goToSetting(f.setting)}>Change it</button>
              ) : f.file ? (
                <a className="btn btn-ghost btn-sm" href={"#/file/" + encodeURIComponent(f.file)}>Open the file</a>
              ) : null}
            </li>
          ))}
        </ul>
      )}
      <details className="doctor-resolved">
        <summary>All {(rep.entries || []).length} settings {label} resolves here · {setCount} set in a file</summary>
        <div className="doctor-bar" data-align-row>
          <input className="cli-memory-search" type="search" placeholder="Filter settings" aria-label="Filter settings" value={query} onChange={(e) => setQuery(e.target.value)} />
          <div className="pkg-scope" role="radiogroup" aria-label="Which settings">
            <button type="button" className="pkg-scope-btn" role="radio" aria-checked={onlySet} onClick={() => setOnlySet(true)}>Set in a file</button>
            <button type="button" className="pkg-scope-btn" role="radio" aria-checked={!onlySet} onClick={() => setOnlySet(false)}>All</button>
          </div>
        </div>
        {entries.length === 0 ? (
          <div className="cli-notice" role="status">
            <span>{onlySet && !query ? "Neither file sets anything; " + label + " runs on its defaults here." : "No setting matches this filter."}</span>
            {onlySet && !query ? (
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setOnlySet(false)}>Show all</button>
            ) : (
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setQuery(""); }}>Clear filter</button>
            )}
          </div>
        ) : (
          <div className="doctor-table" role="table" aria-label="Resolved settings">
            <div className="doctor-row doctor-head" role="row">
              <span role="columnheader">Setting · {entries.length}</span>
              <span role="columnheader">Value</span>
              <span role="columnheader" className="doctor-src">Comes from</span>
            </div>
            {entries.map((e) => (
              <div className="doctor-row" role="row" key={e.key} title={e.description || ""}>
                <code className="doctor-key" role="cell">{e.key}</code>
                <code className="doctor-val" role="cell" title={e.redacted ? "Hidden: this is a credential" : formatValue(e.value, 2000)}>{e.redacted ? "hidden" : formatValue(e.value)}</code>
                <span className="doctor-src" role="cell">{sourceLabel(e.source, workspaceName)}</span>
              </div>
            ))}
          </div>
        )}
      </details>
    </section>
  );
}
