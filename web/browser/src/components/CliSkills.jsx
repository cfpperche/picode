import { Fragment, useCallback, useEffect, useState } from "react";
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

// What one agent CLI loads as Agent Skills, and from where (ADR-0196). Slice
// 1 reads: every folder the CLI reads in its own precedence order, which copy
// wins, who installed it, and what it costs at start. Slice 2 installs,
// removes, checks and updates — PiCode's own writes into .agents/skills and
// the skills CLI's locks (docs/plans/skills.md).

// Brings a row's detail into view the first time it renders.
function scrollOnce(el) {
  if (el && !el.dataset.seen) {
    el.dataset.seen = "1";
    el.scrollIntoView({ block: "nearest" });
  }
}

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
  const [ask, setAsk] = useState(null); // { key, text, action }

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

  // PiCode's own writes announce themselves (skills.changed).
  useEffect(() => subscribeFeed((event) => {
    if (event.type === "skills.changed") setReload((n) => n + 1);
  }), []);

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

  // Remove and update: the first call carries no confirmation; a refusal the
  // person can answer (409) becomes a question under the row.
  const act = async (row, verb, extra = {}) => {
    const key = row.dir;
    setRowBusy(key);
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
      if (code === "modified" || code === "unlocked") {
        setAsk({ key, text: x.message, verb, extra: verb === "remove" ? { confirm: true } : { force: true } });
      } else {
        toast.error(x.message);
      }
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
      <div className="cli-memory-bar" data-align-row>
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
          className="cli-memory-search"
          type="search"
          placeholder="Filter"
          value={filter}
          aria-label="Filter skills"
          onChange={(e) => setFilter(e.target.value)}
        />
        <span className="cli-skills-bar-gap" />
        <button type="button" className="btn btn-ghost" disabled={checking} onClick={check}>{checking ? <span className="cred-waiting">Checking</span> : "Check for updates"}</button>
        <button type="button" className="btn btn-primary" onClick={() => setAdding(true)}>Add skill</button>
      </div>
      {updates ? <p className="cli-memory-note">{updatesSummary(updates)}</p> : null}

      {summary.total ? <p className="cli-memory-note">
        {summary.loaded} of {summary.total} load in {terminalCliLabel(cli)}
        {summary.tokens ? <> · {tokensLabel(summary.tokens)} at every start</> : null}
        {summary.shadowed ? <> · {summary.shadowed} shadowed by another copy</> : null}
      </p> : null}

      {trust ? (
        <div className="cli-notice" role="status">
          <span>{trust.text}</span>
          {trust.command ? <button type="button" className="btn btn-ghost btn-sm" title={trust.command} onClick={() => copy(trust.command)}>Copy {trust.command}</button> : null}
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
                      <td className="cli-skills-c-status">
                        <span className={"cli-skills-status is-" + st.tone} title={st.detail || ""}>{st.label}</span>
                        {updateOf[row.scope + ":" + row.name]?.status === "behind" && canRemoveSkill(row) ? <span className="cli-skills-flag is-update"> · update</span> : null}
                      </td>
                      <td className="cli-skills-c-origin" title={origin.title}>{origin.label}{origin.modified ? <span className="cli-skills-flag"> · edited</span> : null}</td>
                      <td className="cli-skills-c-root" title={row.dir}>{row.root}{row.linked ? <span className="cli-skills-flag"> · link</span> : null}</td>
                      <td className="cli-skills-c-cost is-num">{row.status === "loaded" ? tokensLabel(row.tokens) : ""}</td>
                    </tr>
                    {isOpen ? (
                      <tr className="cli-skills-detail" ref={scrollOnce}>
                        <td colSpan={5}>
                          {st.detail ? <p>{st.detail}.</p> : null}
                          {(row.problems || []).length && row.status !== "invalid" ? <p className="is-warn">{row.problems.join("; ")}.</p> : null}
                          <p className="cli-skills-path">{row.dir}</p>
                          {alsoLoadedLine(row) ? <p>{alsoLoadedLine(row)}.</p> : null}
                          {row.digest ? <p className="cli-skills-path" title={"sha256 " + row.digest + " — the folder's fingerprint, as the skills tool records it"}>Fingerprint {row.digest.slice(0, 12)}</p> : null}
                          {updateOf[row.scope + ":" + row.name] && updateOf[row.scope + ":" + row.name].status !== "current" && updateOf[row.scope + ":" + row.name].reason
                            ? <p className="is-warn">{updateOf[row.scope + ":" + row.name].reason}.</p> : null}
                          {canRemoveSkill(row) ? (ask && ask.key === row.dir ? null : (
                            <div className="cli-skills-actions" data-align-row>
                              {updateOf[row.scope + ":" + row.name]?.status === "behind"
                                ? <button type="button" className="btn btn-primary btn-sm" disabled={rowBusy === row.dir} onClick={() => act(row, "update")}>Update</button> : null}
                              <button type="button" className="btn btn-ghost btn-danger btn-sm" disabled={rowBusy === row.dir} onClick={() => setAsk({ key: row.dir, text: removeQuestion(row, { workspaceName, agentName: data?.agent?.name }), verb: "remove", extra: {} })}>Remove</button>
                            </div>
                          ))
 : <p>{terminalCliLabel(cli)} manages this folder itself; PiCode removes only what lives in .agents/skills.</p>}
                          {ask && ask.key === row.dir ? (
                            <div className="cli-notice" role="alert" ref={scrollOnce}>
                              <span>{ask.text}</span>
                              <span className="cli-skills-ask">
                                <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAsk(null)}>Cancel</button>
                                <button type="button" className={"btn btn-sm " + (ask.verb === "remove" ? "btn-danger" : "btn-primary")} disabled={rowBusy === row.dir} onClick={() => act(row, ask.verb, ask.extra)}>{ask.verb === "remove" ? (ask.extra.confirm ? "Remove anyway" : "Remove") : "Update anyway"}</button>
                              </span>
                            </div>
                          ) : null}
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
      {dialog}
    </div>
  );
}
