import { useEffect, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import { ownerFileURL } from "../lib/fileIO.js";
import { compactCount, prBlockedAction, prChecksLabel, prReviewLabel, prStateLabel } from "../lib/inspector.js";
import { IconExternal } from "./Icons.jsx";

// usePullRequest reads the anchor's pull request through the host's gh
// (ADR-0078, phase 2): one read per owner, folder and branch, answered from
// the server's minute cache; a change of `nonce` (the rail's Refresh) asks
// gh again. Never a timer — GitHub's budget is the user's, not the rail's.
export function usePullRequest({ owner, root, branch, enabled, nonce }) {
  const [page, setPage] = useState(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const key = owner && root ? `${owner.kind}:${owner.id}:${root}:${branch || ""}` : "";
  const lastNonce = useRef(nonce);
  const lastKey = useRef("");
  useEffect(() => {
    if (!enabled || !key) {
      setPage(null);
      setError("");
      return undefined;
    }
    const refresh = nonce !== lastNonce.current && key === lastKey.current;
    lastNonce.current = nonce;
    lastKey.current = key;
    const request = new AbortController();
    setBusy(true);
    const url = ownerFileURL(owner, "pr", "", root);
    api(url + (refresh ? "&refresh=1" : ""), { signal: request.signal })
      .then((next) => { if (!request.signal.aborted) { setPage(next); setError(""); } })
      .catch((e) => { if (!request.signal.aborted) setError(e.message || "Could not read the pull request."); })
      .finally(() => { if (!request.signal.aborted) setBusy(false); });
    return () => request.abort();
  }, [key, enabled, nonce]); // owner and root are inside key
  return { page, error, busy };
}

// The PR tab: gh's answer as a small card, or one line and one action for
// each state PiCode cannot change itself. Creating or logging in happens in
// the owner's terminal with the command pre-typed, never submitted.
export default function InspectorPR({ page, error, busy, branch, onRetry, onTerminal }) {
  if (error) {
    return <p className="insp-msg" role="status"><span>{error}</span><button type="button" className="btn btn-sm" onClick={onRetry} disabled={busy}>Try again</button></p>;
  }
  if (!page) {
    return (
      <div className="ft-skeleton" aria-label="Loading pull request" aria-busy="true">
        <div className="skel-line w-50" /><div className="skel-line w-90" /><div className="skel-line w-70" /><div className="skel-line w-40" />
      </div>
    );
  }
  if (page.status === "blocked") {
    const action = prBlockedAction(page.reason);
    return (
      <p className="insp-msg" role="status">
        <span>{page.message}</span>
        {action === "install" ? <a className="btn btn-sm" href="https://cli.github.com" target="_blank" rel="noreferrer">Get gh</a> : null}
        {action === "login" && onTerminal ? <button type="button" className="btn btn-sm" onClick={() => onTerminal("gh auth login")}>Log in from a terminal</button> : null}
        {action === "retry" ? <button type="button" className="btn btn-sm" onClick={onRetry} disabled={busy}>Try again</button> : null}
      </p>
    );
  }
  if (page.status === "none") {
    return (
      <p className="insp-msg" role="status">
        <span>No pull request for <code className="insp-code">{page.branch || branch || "this branch"}</code>.</span>
        {onTerminal ? <button type="button" className="btn btn-sm" onClick={() => onTerminal("gh pr create --fill")}>Create in terminal</button> : null}
      </p>
    );
  }
  const pr = page.pr || {};
  const checks = pr.checks || {};
  const state = pr.draft ? "draft" : String(pr.state || "open");
  return (
    <section className="insp-pr" aria-label={`Pull request #${pr.number}`}>
      <header className="insp-pr-head" data-align-row>
        <span className="insp-pr-number">#{pr.number}</span>
        <span className={"insp-pr-state insp-pr-state-" + state}>{prStateLabel(pr)}</span>
        <a className="insp-btn insp-pr-open" href={pr.url} target="_blank" rel="noreferrer" title="Open on GitHub" aria-label="Open on GitHub"><IconExternal size={14} /></a>
      </header>
      <h3 className="insp-pr-title" title={pr.title}>{pr.title}</h3>
      <dl className="insp-pr-facts">
        <div><dt>Branch</dt><dd><code className="insp-code">{pr.base}</code> ← <code className="insp-code">{pr.head}</code></dd></div>
        <div><dt>Diff</dt><dd><span className="gg-add">+{compactCount(pr.additions)}</span> <span className="gg-del">−{compactCount(pr.deletions)}</span> · {pr.changedFiles} {pr.changedFiles === 1 ? "file" : "files"}</dd></div>
        <div><dt>Checks</dt><dd className={checks.failed ? "insp-pr-bad" : checks.pending ? "insp-pr-wait" : ""}>{prChecksLabel(checks)}</dd></div>
        <div><dt>Review</dt><dd>{prReviewLabel(pr.reviewDecision)}</dd></div>
        <div><dt>Updated</dt><dd title={pr.updatedAt}>{relTime(pr.updatedAt)}{pr.author ? ` · ${pr.author}` : ""}</dd></div>
      </dl>
      {checks.failing && checks.failing.length ? (
        <ul className="insp-pr-failing" aria-label="Failing checks">
          {checks.failing.map((f) => (
            <li key={f.name + f.url}>{f.url ? <a href={f.url} target="_blank" rel="noreferrer">{f.name}</a> : f.name}</li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
