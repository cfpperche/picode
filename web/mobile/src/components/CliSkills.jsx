import { useCallback, useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import ScopeIcon from "./ScopeIcon.jsx";
import { api } from "@picode/shared/client/api.js";
import {
  agentScopeLine, alsoLoadedLine, canRemoveSkill, skillScopes, cliSkillsHash, skillOrigin, skillStatus, skillsEmptyLine, skillsReportPath,
  skillsSummary, skillTargetBody, installedLine, removeQuestion, askLabel, canPromoteSkill, promoteQuestion, promoteRetry, promotedLine, trialLine, noAgentScopeLine, tokensLabel, trustLine, updatesByKey, updatesSummary, visibleSkills,
  hasSkillSwitch, noSwitchLine, skillToggleBody, toggledLine, withSkillEnabled,
} from "@picode/shared/domain/cliSkills.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import AddSkillDialog from "./AddSkillDialog.jsx";
import SkillsMarketplace from "./SkillsMarketplace.jsx";
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
  const [flipping, setFlipping] = useState("");
  const [open, setOpen] = useState("");
  const [adding, setAdding] = useState(false);
  // Installed | Marketplace (slice 5); a card opens the dialog on its source.
  const [tab, setTab] = useState("installed");
  const [market, setMarket] = useState(null);
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
      onClose={() => { setAdding(false); setMarket(null); }}
      initialSource={market?.install || ""}
      pickName={market?.name || ""}
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

  // The CLI's own per-skill switch (slice 3), flipped at once and read back.
  const flip = async (row, enabled) => {
    setFlipping(row.dir);
    setData((d) => withSkillEnabled(d, row.dir, row.scope, enabled));
    try {
      const res = await api("/api/skills/toggle", { method: "POST", body: JSON.stringify(skillToggleBody(cli, row, workspaceId, enabled)) });
      (res.note ? toast.info : toast.ok)(toggledLine(res));
    } catch (x) {
      toast.error(x.message);
    } finally {
      setFlipping("");
      setReload((n) => n + 1);
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
      } else if (verb === "promote") {
        // Slice 6: the copy the agent ran with moves into its workspace.
        const res = await api("/api/skills/promote", { method: "POST", body: JSON.stringify({ agent: agentId, name: row.name, ...extra }) });
        toast.ok(promotedLine(res, workspaceName));
        (res.notes || []).forEach((n) => toast.info(n));
        setOpen("");
      } else {
        const res = await api("/api/skills/update", { method: "POST", body: JSON.stringify(body) });
        toast.ok(res.status === "current" ? row.name + " is already current." : row.name + " updated.");
        setUpdates((u) => (u ? u.map((r) => (r.scope === row.scope && r.name === row.name ? { ...r, status: "current" } : r)) : u));
      }
      setReload((n) => n + 1);
    } catch (x) {
      const code = x.body?.code || "";
      if (verb === "promote" && promoteRetry(code)) setAsk({ key: row.dir, text: x.message, verb, extra: { ...extra, ...promoteRetry(code) } });
      else if (code === "modified" || code === "unlocked") setAsk({ key: row.dir, text: x.message, verb, extra: verb === "remove" ? { confirm: true } : { force: true } });
      else toast.error(x.message);
    } finally {
      setRowBusy("");
    }
  };

  const tabs = (
    <div className="pkg-tabs" role="tablist" aria-label="Skills">
      <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "installed"} onClick={() => setTab("installed")}>
        Installed{rows.length ? <span className="pkg-tab-count">{rows.length}</span> : null}
      </button>
      <button type="button" role="tab" className="pkg-tab" aria-selected={tab === "marketplace"} onClick={() => setTab("marketplace")}>Marketplace</button>
    </div>
  );
  if (tab === "marketplace") {
    return (
      <div className="cli-skills">
        {tabs}
        <SkillsMarketplace report={data} onInstall={(item) => { setMarket(item); setAdding(true); }} />
        {dialog}
      </div>
    );
  }

  // With an agent chip, the pane stays up even with no skill anywhere: that
  // chip is where the agent's own list starts.
  if (!rows.length && !data?.agent) {
    return (
      <>
        {tabs}
        <div className="cli-notice" role="status">
          <span>{skillsEmptyLine(cli, { workspaceName, hasWorkspace })}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setTab("marketplace")}>Browse the Marketplace</button>
        </div>
        {noAgentScopeLine(data, cli, agentId) ? <p className="cli-memory-note">{noAgentScopeLine(data, cli, agentId)}</p> : null}
        {dialog}
      </>
    );
  }

  return (
    <div className="cli-skills">
      {tabs}
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
        {summary.off ? <> · {summary.off} switched off</> : null}
      </p> : null}
      {noSwitchLine(data, cli) ? <p className="cli-memory-note">{noSwitchLine(data, cli)}</p> : null}
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
                <div className="cli-skills-row">
                <button type="button" className="cli-skills-head" aria-expanded={isOpen} onClick={() => setOpen(isOpen ? "" : row.dir)}>
                  <span className="cli-skills-line">
                    <span className="cli-skills-name">{row.name}</span>
                    <span className={"cli-skills-status is-" + st.tone}>{updateOf[row.scope + ":" + row.name]?.status === "behind" && canRemoveSkill(row) ? "Update" : st.label}</span>
                  </span>
                  <span className="cli-skills-desc">{row.description || "No description"}</span>
                  <span className="cli-skills-meta">{origin.label} · {row.root}{row.status === "loaded" && row.tokens ? " · " + tokensLabel(row.tokens) : ""}</span>
                </button>
                {data?.toggle && current !== "agent" && hasSkillSwitch(row) ? (
                  <Switch.Root
                    className={"rx-switch cli-skills-switch" + (flipping === row.dir ? " is-busy" : "")}
                    checked={row.enabled}
                    disabled={flipping === row.dir}
                    onCheckedChange={(v) => flip(row, v)}
                    aria-label={(row.enabled ? "Turn off " : "Turn on ") + row.name}
                  >
                    <Switch.Thumb className="rx-switch-thumb" />
                  </Switch.Root>
                ) : null}
                </div>
                {isOpen ? (
                  <div className="cli-skills-body">
                    {st.detail ? <p>{st.detail}.</p> : null}
                    {data?.switchNote && hasSkillSwitch(row) && row.scope === "workspace" ? <p>{data.switchNote}</p> : null}
                    {(row.problems || []).length && row.status !== "invalid" ? <p className="is-warn">{row.problems.join("; ")}.</p> : null}
                    <p className="cli-skills-path">{row.dir}</p>
                    {alsoLoadedLine(row) ? <p>{alsoLoadedLine(row)}.</p> : null}
                    {origin.modified ? <p className="is-warn">Changed since it was installed.</p> : null}
                    {trialLine(row, { agentName: data?.agent?.name, workspaceName }) && data?.workspacePath ? <p>{trialLine(row, { agentName: data?.agent?.name, workspaceName })}</p> : null}
                          {canRemoveSkill(row) ? (ask && ask.key === row.dir ? null : (
                      <div className="cli-skills-actions" data-align-row>
                        {updateOf[row.scope + ":" + row.name]?.status === "behind"
                          ? <button type="button" className="btn btn-primary btn-sm" disabled={rowBusy === row.dir} onClick={() => act(row, "update")}>Update</button> : null}
                        {canPromoteSkill(row, data)
                                ? <button type="button" className="btn btn-primary btn-sm" disabled={rowBusy === row.dir} onClick={() => setAsk({ key: row.dir, text: promoteQuestion(row, { workspaceName, agentName: data?.agent?.name }), verb: "promote", extra: {} })}>Promote to {workspaceName || "the workspace"}</button> : null}
                              <button type="button" className="btn btn-ghost btn-danger btn-sm" disabled={rowBusy === row.dir} onClick={() => setAsk({ key: row.dir, text: removeQuestion(row, { workspaceName, agentName: data?.agent?.name }), verb: "remove", extra: {} })}>Remove</button>
                      </div>
                    ))
 : <p>{terminalCliLabel(cli)} manages this folder itself; PiCode removes only what lives in .agents/skills.</p>}
                    {ask && ask.key === row.dir ? (
                      <div className="cli-notice" role="alert">
                        <span>{ask.text}</span>
                        <span className="cli-skills-ask">
                          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAsk(null)}>Cancel</button>
                          <button type="button" className={"btn btn-sm " + (ask.verb === "remove" ? "btn-danger" : "btn-primary")} disabled={rowBusy === row.dir} onClick={() => act(row, ask.verb, ask.extra)}>{askLabel(ask)}</button>
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
