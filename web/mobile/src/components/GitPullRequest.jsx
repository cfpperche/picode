import { useEffect, useRef, useState } from "react";
import { relTime } from "@picode/shared/domain/relTime.js";
import { gitURL } from "../lib/git/model.js";
import { useGitRead } from "../lib/git/useGitRead.js";
import GitReadState from "./GitReadState.jsx";

function checksLabel(checks = {}) {
  const groups = ["failed", "pending", "passed", "skipped"].filter(k => checks[k]).map(k => `${checks[k]} ${k}`);
  return groups.join(" · ") || "No checks";
}

export default function GitPullRequest({ owner, root, nonce, onMoved, onOpenTerminal, blocked, read = null }) {
  const [retry, setRetry] = useState(0);
  const [sending, setSending] = useState(false);
  const submitting = useRef(false);
  const [actionError, setActionError] = useState("");
  // A host that already reads the PR (the Inspector, for its tab label)
  // passes its read; the component then skips its own fetch.
  const self = useGitRead(read ? "" : gitURL(owner, "pr", { root, refresh: !!nonce || !!retry }), `${nonce}:${retry}`, onMoved, blocked);
  const { data, error, loading } = read || self;
  useEffect(() => { setActionError(""); }, [data?.status, data?.reason]);
  const retryRead = read ? (read.onRetry || (() => {})) : () => setRetry(n => n + 1);
  async function prepare(command) {
    if (submitting.current || blocked) return;
    submitting.current = true;
    setSending(true); setActionError("");
    try { await onOpenTerminal(owner, root, command, { run: false }); }
    catch (e) { setActionError(e?.message || "Could not prepare the terminal."); }
    finally { submitting.current = false; setSending(false); }
  }
  const pr = data?.pr;
  return <section className="m-git-pr" aria-label="Pull request">
    <GitReadState error={error} loading={!data && loading} onRetry={retryRead} />
    {data?.status === "blocked" ? <div className="m-git-state"><p>{data.message}</p>
      {data.reason === "gh-missing" ? <a className="btn" href="https://cli.github.com" target="_blank" rel="noreferrer">Get GitHub CLI</a> : data.reason === "gh-unauth" ? <button type="button" className="btn" disabled={sending || blocked} onClick={() => prepare("gh auth login")}>{sending ? "Preparing…" : "Log in from terminal"}</button> : <button type="button" className="btn" onClick={retryRead}>Retry</button>}
    </div> : null}
    {data?.status === "none" ? <div className="m-git-state"><p>No pull request for {data.branch || "this branch"}.</p><button type="button" className="btn" disabled={sending || blocked} onClick={() => prepare("gh pr create --fill")}>{sending ? "Preparing…" : "Create in terminal"}</button></div> : null}
    {pr ? <><p className="m-git-hint">#{pr.number} · {pr.draft ? "Draft" : String(pr.state || "open").toLowerCase()}</p><h2>{pr.title}</h2><a className="btn" href={pr.url} target="_blank" rel="noreferrer">Open on GitHub</a>
      <dl className="m-git-pr-facts"><dt>Branch</dt><dd>{pr.head} → {pr.base}</dd><dt>Changes</dt><dd><span className="m-git-add">+{pr.additions || 0}</span> <span className="m-git-del">−{pr.deletions || 0}</span> · {pr.changedFiles || 0} files</dd><dt>Checks</dt><dd>{checksLabel(pr.checks)}</dd><dt>Review</dt><dd>{({ APPROVED: "Approved", CHANGES_REQUESTED: "Changes requested", REVIEW_REQUIRED: "Review required" })[pr.reviewDecision] || "No review yet"}</dd><dt>Updated</dt><dd>{relTime(pr.updatedAt)}{pr.author ? ` · ${pr.author}` : ""}</dd></dl>
      {pr.checks?.failing?.length ? <ul className="m-git-failed-checks">{pr.checks.failing.map(check => <li key={check.name + check.url}>{check.url ? <a href={check.url} target="_blank" rel="noreferrer">{check.name}</a> : check.name}</li>)}</ul> : null}
    </> : null}
    {sending ? <p className="m-git-pending" role="status"><span aria-hidden="true" />Preparing the terminal…</p> : null}
    {actionError ? <p className="form-error" role="alert">{actionError}</p> : null}
  </section>;
}
