import { useCallback, useEffect, useState } from "react";
import ScopeIcon from "./ScopeIcon.jsx";
import { api } from "@picode/shared/client/api.js";
import {
  agentScopeLine, alsoLoadedLine, canRemoveSkill, skillScopes, cliSkillsHash, skillOrigin, skillStatus, skillsEmptyLine, skillsReportPath,
  skillsSummary, skillTargetBody, installedLine, removeQuestion, noAgentScopeLine, tokensLabel, trustLine, updatesByKey, updatesSummary, visibleSkills,
} from "@picode/shared/domain/cliSkills.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import AddSkillDialog from "./AddSkillDialog.jsx";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { toast } from "../lib/toast.js";

// The phone's Skills pane (ADR-0196): the desktop's rows as a list, the same
// words from cliSkills.js; slice 2 adds install, remove, check and update
// (docs/plans/skills.md).

function copy(command) {
  navigator.clipboard.writeText(command)
    .then(() => toast.ok("Command copied."))
    .catch(() => toast.error("Clipboard blocked — copy it by hand."));
}

export default function CliSkills({ route, workspaceId = "", agentId = "", workspaceName = "" }) {
  const cli = route.id;
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const [scope, setScope] = useState(route.scope || "machine");
  const [filter, setFilter] = useState("");
  const [open, setOpen] = useState("");
  const [adding, setAdding] = useState(false);
  const [updates, setUpdates] = useState(null);
  const [checking, setChecking] = useState(false);
  const [rowBusy, setRowBusy] = useState("");
  const [ask, setAsk] = useState(null);

  const load = useCallback(() => api(skillsReportPath(cli, workspaceId, agentId)), [cli, workspaceId, agentId]);

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

  useEffect(() => subscribeFeed((event) => {
    if (event.type === "skills.changed") setReload((n) => n + 1);
  }), []);

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
  const scopes = skillScopes(data, workspaceName);
  const agentLine = scope === "agent" ? agentScopeLine(data, cli) : "";
  const shown = visibleSkills(rows, { scope: scopes.some((x) => x.id === scope) ? scope : "machine", text: filter, agent: data?.agent });
  const current = scopes.some((x) => x.id === scope) ? scope : "machine";
  // The count follows the chip, like the table under it.
  const trust = current === "machine" ? null : trustLine(data, rows);
  const summary = skillsSummary(visibleSkills(rows, { scope: current, agent: data?.agent }));


  const updateOf = updates ? updatesByKey(updates) : {};
  const dialog = (
    <AddSkillDialog
      open={adding}
      onClose={() => setAdding(false)}
      workspaceId={workspaceId}
      workspaceName={workspaceName}
      agentId={data?.agent ? agentId : ""}
      agentName={data?.agent?.name || ""}
      initialScope={current}
      onDone={(res, where) => {
        toast.ok(installedLine(res, where, data?.agent?.name));
        (res.notes || []).forEach((n) => toast.info(n));
        setReload((n) => n + 1);
      }}
    />
  );

  const check = async () => {
    setChecking(true);
    try {
      const q = new URLSearchParams();
      if (workspaceId) q.set("workspace", workspaceId);
      const res = await api("/api/skills/updates?" + q);
      setUpdates(res.rows || []);
    } catch (x) {
      toast.error(x.message);
    } finally {
      setChecking(false);
    }
  };

  const act = async (row, verb, extra = {}) => {
    setRowBusy(row.dir);
    setAsk(null);
    const body = { ...skillTargetBody(row, workspaceId, agentId), ...extra };
    try {
      if (verb === "remove") {
        await api("/api/skills", { method: "DELETE", body: JSON.stringify(body) });
        toast.ok(row.name + " removed.");
        setOpen("");
      } else {
        const res = await api("/api/skills/update", { method: "POST", body: JSON.stringify(body) });
        toast.ok(res.status === "current" ? row.name + " is already current." : row.name + " updated.");
        setUpdates((u) => (u ? u.map((r) => (r.scope === row.scope && r.name === row.name ? { ...r, status: "current" } : r)) : u));
      }
      setReload((n) => n + 1);
    } catch (x) {
      const code = x.body?.code || "";
      if (code === "modified" || code === "unlocked") setAsk({ key: row.dir, text: x.message, verb, extra: verb === "remove" ? { confirm: true } : { force: true } });
      else toast.error(x.message);
    } finally {
      setRowBusy("");
    }
  };

  // With an agent chip, the pane stays up even with no skill anywhere: that
  // chip is where the agent's own list starts.
  if (!rows.length && !data?.agent) {
    return (
      <>
        <div className="cli-notice" role="status">
          <span>{skillsEmptyLine(cli, { workspaceName, hasWorkspace })}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAdding(true)}>Add skill</button>
        </div>
        {noAgentScopeLine(data, cli, agentId) ? <p className="cli-memory-note">{noAgentScopeLine(data, cli, agentId)}</p> : null}
        {dialog}
      </>
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
            aria-checked={(scopes.some((x) => x.id === scope) ? scope : "machine") === s.id}
            href={cliSkillsHash(cli, { workspaceId, agentId, scope: s.id })}
            onClick={() => { setScope(s.id); setOpen(""); }}
          ><ScopeIcon scope={s.id} />{s.label}</a>
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
      <div className="cli-skills-buttons" data-align-row>
        <button type="button" className="btn btn-ghost" disabled={checking} onClick={check}>{checking ? <span className="cred-waiting">Checking</span> : "Check for updates"}</button>
        <button type="button" className="btn btn-primary" onClick={() => setAdding(true)}>Add skill</button>
      </div>
      {updates ? <p className="cli-memory-note">{updatesSummary(updates)}</p> : null}
      {summary.total ? <p className="cli-memory-note">
        {summary.loaded} of {summary.total} load in {terminalCliLabel(cli)}
        {summary.tokens ? <> · {tokensLabel(summary.tokens)} at every start</> : null}
      </p> : null}
      {trust ? (
        <div className="cli-notice" role="status">
          <span>{trust.text}</span>
          {trust.command ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => copy(trust.command)}>Copy command</button> : null}
        </div>
      ) : null}
      {agentLine ? (
        <div className="cli-notice" role="status">
          <span>{agentLine}</span>
          {cli === "pi" || cli === "omp" ? <a className="btn btn-ghost btn-sm" href={cliPackagesHash(cli, { workspaceId, agentId, scope: "agent" })}>Open its packages</a> : null}
        </div>
      ) : null}
      {(data?.notes || []).map((n) => <p key={n} className="cli-memory-note is-warn">{n}</p>)}
      {noAgentScopeLine(data, cli, agentId) ? <p className="cli-memory-note">{noAgentScopeLine(data, cli, agentId)}</p> : null}
      {!shown.length ? (
        <div className="cli-notice" role="status">
          <span>{filter.trim() ? "No skill matches this filter." : scope === "agent" ? "Nothing here loads for this agent." : "No skills in this scope."}</span>
          {filter.trim()
            ? <button type="button" className="btn btn-ghost btn-sm" onClick={() => setFilter("")}>Clear the filter</button>
            : <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAdding(true)}>Add skill</button>}
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
                    <span className={"cli-skills-status is-" + st.tone}>{updateOf[row.scope + ":" + row.name]?.status === "behind" && canRemoveSkill(row) ? "Update" : st.label}</span>
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
                    {canRemoveSkill(row) ? (ask && ask.key === row.dir ? null : (
                      <div className="cli-skills-actions" data-align-row>
                        {updateOf[row.scope + ":" + row.name]?.status === "behind"
                          ? <button type="button" className="btn btn-primary btn-sm" disabled={rowBusy === row.dir} onClick={() => act(row, "update")}>Update</button> : null}
                        <button type="button" className="btn btn-ghost btn-danger btn-sm" disabled={rowBusy === row.dir} onClick={() => setAsk({ key: row.dir, text: removeQuestion(row, { workspaceName, agentName: data?.agent?.name }), verb: "remove", extra: {} })}>Remove</button>
                      </div>
                    ))
 : <p>{terminalCliLabel(cli)} manages this folder itself; PiCode removes only what lives in .agents/skills.</p>}
                    {ask && ask.key === row.dir ? (
                      <div className="cli-notice" role="alert">
                        <span>{ask.text}</span>
                        <span className="cli-skills-ask">
                          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAsk(null)}>Cancel</button>
                          <button type="button" className={"btn btn-sm " + (ask.verb === "remove" ? "btn-danger" : "btn-primary")} disabled={rowBusy === row.dir} onClick={() => act(row, ask.verb, ask.extra)}>{ask.verb === "remove" ? (ask.extra.confirm ? "Remove anyway" : "Remove") : "Update anyway"}</button>
                        </span>
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
      {dialog}
    </div>
  );
}
