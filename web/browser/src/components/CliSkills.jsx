import { Fragment, useCallback, useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import {
  alsoLoadedLine, cliSkillsHash, skillOrigin, skillStatus, skillsEmptyLine, skillsReportPath,
  skillsSummary, tokensLabel, trustLine, visibleSkills,
} from "@picode/shared/domain/cliSkills.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { toast } from "../lib/toast.js";

// What one agent CLI loads as Agent Skills, and from where (ADR-0196). Slice
// 1 reads only: every folder the CLI reads in its own precedence order, which
// copy wins, who installed it, and what it costs at start. Installing and
// switching skills off land in the next slices (docs/plans/skills.md).

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

  // Skill folders change outside PiCode (an installer, an editor): coming back
  // to the tab reads them again rather than showing a stale list.
  useEffect(() => {
    const onFocus = () => setReload((n) => n + 1);
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, []);

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
      <div className="cli-memory-bar" data-align-row>
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
          className="cli-memory-search"
          type="search"
          placeholder="Filter"
          value={filter}
          aria-label="Filter skills"
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>

      <p className="cli-memory-note">
        {summary.loaded} of {summary.total} load in {terminalCliLabel(cli)}
        {summary.tokens ? <> · {tokensLabel(summary.tokens)} at every start</> : null}
        {summary.shadowed ? <> · {summary.shadowed} shadowed by another copy</> : null}
      </p>

      {trust ? (
        <div className="cli-notice" role="status">
          <span>{trust.text}</span>
          {trust.command ? <button type="button" className="btn btn-ghost btn-sm" title={trust.command} onClick={() => copy(trust.command)}>Copy {trust.command}</button> : null}
        </div>
      ) : null}

      {(data?.notes || []).map((n) => <p key={n} className="cli-memory-note is-warn">{n}</p>)}

      {!shown.length ? (
        <div className="cli-notice" role="status">
          <span>No skill matches this filter.</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setFilter(""); setScope("all"); }}>Show all</button>
        </div>
      ) : (
        <div className="cli-memory-scroll">
          <table className="cli-skills-table">
            <thead>
              <tr>
                <th className="cli-skills-c-name">Skill</th>
                <th className="cli-skills-c-status">Status</th>
                <th className="cli-skills-c-origin">Source</th>
                <th className="cli-skills-c-root">Folder</th>
                <th className="cli-skills-c-cost is-num">At start</th>
              </tr>
            </thead>
            <tbody>
              {shown.map((row) => {
                const key = row.dir;
                const st = skillStatus(row, cli);
                const origin = skillOrigin(row);
                const isOpen = open === key;
                return (
                  <Fragment key={key}>
                    <tr
                      className={"cli-skills-head" + (isOpen ? " is-open" : "") + (row.status === "shadowed" ? " is-dim" : "")}
                      onClick={() => setOpen(isOpen ? "" : key)}
                      tabIndex={0}
                      aria-expanded={isOpen}
                      onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); setOpen(isOpen ? "" : key); } }}
                    >
                      <td className="cli-skills-c-name">
                        <span className="cli-skills-name">{row.name}</span>
                        <span className="cli-skills-desc" title={row.description}>{row.description || "No description"}</span>
                      </td>
                      <td className="cli-skills-c-status"><span className={"cli-skills-status is-" + st.tone} title={st.detail || ""}>{st.label}</span></td>
                      <td className="cli-skills-c-origin" title={origin.title}>{origin.label}{origin.modified ? <span className="cli-skills-flag"> · edited</span> : null}</td>
                      <td className="cli-skills-c-root" title={row.dir}>{row.root}{row.linked ? <span className="cli-skills-flag"> · link</span> : null}</td>
                      <td className="cli-skills-c-cost is-num">{row.status === "loaded" ? tokensLabel(row.tokens) : ""}</td>
                    </tr>
                    {isOpen ? (
                      <tr className="cli-skills-detail" ref={(el) => el?.scrollIntoView({ block: "nearest" })}>
                        <td colSpan={5}>
                          {st.detail ? <p>{st.detail}.</p> : null}
                          {(row.problems || []).length && row.status !== "invalid" ? <p className="is-warn">{row.problems.join("; ")}.</p> : null}
                          <p className="cli-skills-path">{row.dir}</p>
                          {alsoLoadedLine(row) ? <p>{alsoLoadedLine(row)}.</p> : null}
                          {row.digest ? <p className="cli-skills-path" title={"sha256 " + row.digest + " — the folder's fingerprint, as the skills tool records it"}>Fingerprint {row.digest.slice(0, 12)}</p> : null}
                        </td>
                      </tr>
                    ) : null}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
