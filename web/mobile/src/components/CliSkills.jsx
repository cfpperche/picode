import { useCallback, useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import {
  alsoLoadedLine, canRemoveSkill, cliSkillsHash, skillOrigin, skillStatus, skillsEmptyLine, skillsReportPath,
  skillsSummary, skillTargetBody, tokensLabel, trustLine, updatesByKey, updatesSummary, visibleSkills,
} from "@picode/shared/domain/cliSkills.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
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

export default function CliSkills({ route, workspaceId = "", workspaceName = "" }) {
  const cli = route.id;
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const [scope, setScope] = useState(route.scope || "all");
  const [filter, setFilter] = useState("");
  const [open, setOpen] = useState("");
  const [adding, setAdding] = useState(false);
  const [updates, setUpdates] = useState(null);
  const [checking, setChecking] = useState(false);
  const [rowBusy, setRowBusy] = useState("");
  const [ask, setAsk] = useState(null);

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
  const shown = visibleSkills(rows, { scope, text: filter });
  const summary = skillsSummary(rows);
  const trust = trustLine(data, rows);
  const scopes = [
    { id: "all", label: "All" },
    ...(hasWorkspace ? [{ id: "workspace", label: workspaceName || "This workspace" }] : []),
    { id: "machine", label: "This machine" },
  ];

  const updateOf = updates ? updatesByKey(updates) : {};
  const dialog = (
    <AddSkillDialog
      open={adding}
      onClose={() => setAdding(false)}
      workspaceId={workspaceId}
      workspaceName={workspaceName}
      onDone={(res) => {
        toast.ok(res.status === "already" ? res.name + " is already installed." : res.status === "adopted" ? "Recorded where " + res.name + " came from." : res.name + " installed.");
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
    const body = { ...skillTargetBody(row, workspaceId), ...extra };
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

  if (!rows.length) {
    return (
      <>
        <div className="cli-notice" role="status">
          <span>{skillsEmptyLine(cli, { workspaceName, hasWorkspace })}</span>
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAdding(true)}>Add skill</button>
        </div>
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
      <div className="cli-skills-buttons" data-align-row>
        <button type="button" className="btn btn-ghost" disabled={checking} onClick={check}>{checking ? <span className="cred-waiting">Checking</span> : "Check for updates"}</button>
        <button type="button" className="btn btn-primary" onClick={() => setAdding(true)}>Add skill</button>
      </div>
      {updates ? <p className="cli-memory-note">{updatesSummary(updates)}</p> : null}
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
                        <button type="button" className="btn btn-ghost btn-danger btn-sm" disabled={rowBusy === row.dir} onClick={() => setAsk({ key: row.dir, text: "Remove " + row.name + "? Every CLI that reads this folder loses it.", verb: "remove", extra: {} })}>Remove</button>
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
