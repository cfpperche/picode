import { useEffect, useMemo, useState } from "react";
import * as Sheet from "./MobileSheet.jsx";
import GitReadState from "./GitReadState.jsx";
import GitAncestry from "./GitAncestry.jsx";
import { gitURL } from "../lib/git/model.js";
import { groupBranches, matchCommits, walkParams } from "../lib/git/history.js";
import { useGitRead } from "../lib/git/useGitRead.js";

export default function GitHistory({ owner, root, nonce, blocked, onMoved, onCommit, onWorktree, onActions, onGraph }) {
  const [limit, setLimit] = useState(100);
  const [retry, setRetry] = useState(0);
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useState({ branches: [], remotes: true });
  const [draft, setDraft] = useState(filters);
  const [settings, setSettings] = useState(false);
  const walk = walkParams(filters.branches, filters.remotes);
  const { data, error, loading } = useGitRead(gitURL(owner, "git", { root, limit, ...walk }), `${nonce}:${retry}`, onMoved, blocked);
  const commits = data?.commits || [];
  // The screen keeps the latest graph so a commit opened from here, or the
  // header's own sheet, can offer the refs and worktrees drawn on it.
  useEffect(() => { if (data) onGraph?.(data); }, [data, onGraph]);
  const searching = query.trim().length >= 2;
  const matches = useMemo(() => matchCommits(commits, query), [commits, query]);
  const matchList = commits.filter(c => matches.has(c.hash));
  const [matchIndex, setMatchIndex] = useState(0);
  const activeMatch = searching && matchList.length ? matchList[matchIndex % matchList.length]?.hash : "";
  const groups = groupBranches(data?.refs, draft.remotes);
  const refs = (data?.refs || []).filter(r => filters.remotes || r.kind !== "remote");
  const toggle = name => setDraft(d => ({ ...d, branches: d.branches.includes(name) ? d.branches.filter(b => b !== name) : [...d.branches, name] }));

  return <section className="m-git-history" aria-label="Git history" aria-busy={loading}>
    <div className="m-git-history-tools" data-align-row><input type="search" aria-label="Search loaded history" placeholder="Search history…" value={query} onChange={e => { setQuery(e.target.value); setMatchIndex(0); }} onKeyDown={e => { if (e.key === "Enter" && matchList.length) { e.preventDefault(); setMatchIndex(i => (i + (e.shiftKey ? -1 : 1) + matchList.length) % matchList.length); } }} /><button type="button" className="btn" onClick={() => { setDraft(filters); setSettings(true); }}>Branches</button></div>
    {filters.branches.length ? <p className="m-git-hint">{filters.branches.join(" · ")}</p> : null}
    <GitReadState error={error} loading={!data && loading} onRetry={() => setRetry(n => n + 1)} />
    {data ? <>
      {searching ? <p className="m-git-hint" role="status">{matchList.length ? `${matchList.length} matches in loaded history` : "No matches in loaded history."} <button type="button" className="m-git-text-button" onClick={() => setQuery("")}>Clear search</button></p> : null}
      {commits.length ? <GitAncestry graph={data} refs={refs} query={searching ? query : ""} matches={matches} activeMatch={activeMatch} onCommit={onCommit} onWorktree={onWorktree} onActions={onActions} /> : <div className="m-git-state"><p>No commits in this history.</p><button type="button" className="btn" onClick={() => { setFilters({ branches: [], remotes: true }); setRetry(n => n + 1); }}>Show all branches</button></div>}
      {data.more ? <button type="button" className="btn m-git-load-more" disabled={loading || limit >= 10000} onClick={() => setLimit(n => Math.min(10000, n * 2))}>{loading ? "Loading…" : limit >= 10000 ? "History limit reached" : "Load earlier commits"}</button> : null}
    </> : null}
    <Sheet.Root open={settings} onOpenChange={setSettings}><Sheet.Portal><Sheet.Overlay className="dlg-overlay" /><Sheet.Content className="dlg m-git-sheet">
      <Sheet.Title className="dlg-title">History branches</Sheet.Title><Sheet.Description className="dlg-body">Choose which branches appear in history.</Sheet.Description>
      <label className="m-git-choice"><input type="checkbox" checked={draft.remotes} onChange={e => setDraft(d => ({ ...d, remotes: e.target.checked, branches: e.target.checked ? d.branches : d.branches.filter(b => groups.local.includes(b)) }))} />Include remote branches</label>
      {[["Local", groups.local], ["Remote", groups.remote]].map(([label, names]) => names.length ? <fieldset className="m-git-branch-group" key={label}><legend>{label}</legend>{names.map(name => <label className="m-git-choice" key={name}><input type="checkbox" checked={draft.branches.includes(name)} onChange={() => toggle(name)} /><span>{name}</span></label>)}</fieldset> : null)}
      {!groups.local.length && !groups.remote.length ? <p className="m-git-hint">No branches are available yet.</p> : null}
      <div className="dlg-actions" data-align-row><button type="button" className="btn btn-ghost" onClick={() => setDraft({ branches: [], remotes: true })}>Select all</button><button type="button" className="btn btn-primary" onClick={() => { setFilters(draft); setLimit(100); setSettings(false); }}>Apply</button></div>
    </Sheet.Content></Sheet.Portal></Sheet.Root>
  </section>;
}
