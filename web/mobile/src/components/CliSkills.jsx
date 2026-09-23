import { useCallback, useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import {
  alsoLoadedLine, cliSkillsHash, skillOrigin, skillStatus, skillsEmptyLine, skillsReportPath,
  skillsSummary, tokensLabel, trustLine, visibleSkills,
} from "@picode/shared/domain/cliSkills.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { toast } from "../lib/toast.js";

// The phone's Skills pane (ADR-0196): the desktop's rows as a list, the same
// words from cliSkills.js. Read-only in slice 1 (docs/plans/skills.md).

function copy(command) {
  navigator.clipboard.writeText(command)
    .then(() => toast.ok("Command copied."))
    .catch(() => toast.error("Clipboard blocked — copy it by hand."));
}

export default function CliSkills({ route, workspaceId = "", workspaceName = "" }) {
  const cli = route.id;
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const [scope, setScope] = useState(route.scope || "all");
  const [filter, setFilter] = useState("");
  const [open, setOpen] = useState("");

  const load = useCallback(() => api(skillsReportPath(cli, workspaceId)), [cli, workspaceId]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    load().then((next) => {
      if (!active) return;
      setData(next);
      setError(null);
    }).catch((err) => {
      if (active) setError(err);
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, [load, reload]);

  if (loading && !data) {
    return <div className="cli-loading" aria-label={"Loading " + terminalCliLabel(cli) + " skills"}><div /><div /><div /></div>;
  }
  if (error && !data) {
    return (
      <div className="cli-notice is-error" role="alert">
        <span>{error.message}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setReload((n) => n + 1)}>Try again</button>
      </div>
    );
  }

  const rows = data?.rows || [];
  const hasWorkspace = !!data?.workspacePath;
  const shown = visibleSkills(rows, { scope, text: filter });
  const summary = skillsSummary(rows);
  const trust = trustLine(data, rows);
  const scopes = [
    { id: "all", label: "All" },
    ...(hasWorkspace ? [{ id: "workspace", label: workspaceName || "This workspace" }] : []),
    { id: "machine", label: "This machine" },
  ];

  if (!rows.length) {
    return (
      <div className="cli-notice" role="status">
        <span>{skillsEmptyLine(cli, { workspaceName, hasWorkspace })}</span>
        <button type="button" className="btn btn-ghost btn-sm" onClick={() => setReload((n) => n + 1)}>Check again</button>
      </div>
    );
  }

  return (
    <div className="cli-skills">
      <div className="pkg-scope" role="radiogroup" aria-label="Which skills to show">
        {scopes.map((s) => (
          <a
            key={s.id}
            className="pkg-scope-btn"
            role="radio"
            aria-checked={scope === s.id}
            href={cliSkillsHash(cli, { workspaceId, scope: s.id })}
            onClick={() => { setScope(s.id); setOpen(""); }}
          >{s.label}</a>
        ))}
      </div>
      <input
        className="cli-memory-search cli-skills-search"
        type="search"
        placeholder="Filter"
        value={filter}
        aria-label="Filter skills"
        onChange={(e) => setFilter(e.target.value)}
      />
      <p className="cli-memory-note">
        {summary.loaded} of {summary.total} load in {terminalCliLabel(cli)}
        {summary.tokens ? <> · {tokensLabel(summary.tokens)} at every start</> : null}
      </p>
      {trust ? (
        <div className="cli-notice" role="status">
          <span>{trust.text}</span>
          {trust.command ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => copy(trust.command)}>Copy command</button> : null}
        </div>
      ) : null}
      {(data?.notes || []).map((n) => <p key={n} className="cli-memory-note is-warn">{n}</p>)}
      {!shown.length ? (
        <div className="cli-notice" role="status">
          <span>No skill matches this filter.</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setFilter(""); setScope("all"); }}>Show all</button>
        </div>
      ) : (
        <ul className="cli-skills-list">
          {shown.map((row) => {
            const st = skillStatus(row, cli);
            const origin = skillOrigin(row);
            const isOpen = open === row.dir;
            return (
              <li key={row.dir} className={"cli-skills-item" + (row.status === "shadowed" ? " is-dim" : "")}>
                <button type="button" className="cli-skills-head" aria-expanded={isOpen} onClick={() => setOpen(isOpen ? "" : row.dir)}>
                  <span className="cli-skills-line">
                    <span className="cli-skills-name">{row.name}</span>
                    <span className={"cli-skills-status is-" + st.tone}>{st.label}</span>
                  </span>
                  <span className="cli-skills-desc">{row.description || "No description"}</span>
                  <span className="cli-skills-meta">{origin.label} · {row.root}{row.status === "loaded" && row.tokens ? " · " + tokensLabel(row.tokens) : ""}</span>
                </button>
                {isOpen ? (
                  <div className="cli-skills-body">
                    {st.detail ? <p>{st.detail}.</p> : null}
                    {(row.problems || []).length && row.status !== "invalid" ? <p className="is-warn">{row.problems.join("; ")}.</p> : null}
                    <p className="cli-skills-path">{row.dir}</p>
                    {alsoLoadedLine(row) ? <p>{alsoLoadedLine(row)}.</p> : null}
                    {origin.modified ? <p className="is-warn">Changed since it was installed.</p> : null}
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
