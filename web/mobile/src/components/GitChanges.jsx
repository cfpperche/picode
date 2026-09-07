import { useState } from "react";
import { fileKind, gitURL } from "../lib/git/model.js";
import { useGitRead } from "../lib/git/useGitRead.js";
import GitReadState from "./GitReadState.jsx";

export function GitFileRow({ file, onSelect }) {
  return <button type="button" className="m-git-file-row" onClick={() => onSelect(file)}>
    <span className="m-git-file-name"><strong>{file.path.split("/").pop()}</strong>{file.path.includes("/") ? <small>{file.path.slice(0, file.path.lastIndexOf("/"))}</small> : null}</span>
    <span className="m-git-file-meta"><span>{fileKind(file.kind || file.status)}</span>{file.binary ? <small>Binary</small> : (file.add || file.del) ? <small><span className="m-git-add">+{file.add || 0}</span> <span className="m-git-del">−{file.del || 0}</span>{file.truncated ? " · partial" : ""}</small> : null}</span>
  </button>;
}

export default function GitChanges({ status, onSelect, onHistory }) {
  const files = status?.changes || [];
  return <section aria-label="Uncommitted changes">
    <div className="m-git-section-head"><h2>{files.length ? `${files.length} changed ${files.length === 1 ? "file" : "files"}` : "Working tree"}</h2>{files.length ? <span><span className="m-git-add">+{status?.totals?.add || 0}</span> <span className="m-git-del">−{status?.totals?.del || 0}</span></span> : null}</div>
    {files.length ? <div className="m-git-files">{files.map(file => <GitFileRow key={file.path} file={file} onSelect={onSelect} />)}</div> : <div className="m-git-state"><p>No uncommitted changes.</p><button type="button" className="btn" onClick={onHistory}>View history</button></div>}
  </section>;
}

export function GitWorktreeChanges({ owner, wt, nonce = 0, blocked, onMoved, onSelect, onBack }) {
  const [retry, setRetry] = useState(0);
  const { data, error, loading } = useGitRead(gitURL(owner, "gitstatus", { root: wt.path, worktree: wt.branch || wt.head }), `${nonce}:${retry}`, onMoved, blocked);
  return <div><p className="m-git-hint">{wt.branch || "Detached worktree"} · {wt.path}</p><GitReadState error={error} loading={!data && loading} onRetry={() => setRetry(n => n + 1)} />{data ? <GitChanges status={data} onSelect={onSelect} onHistory={onBack} /> : null}</div>;
}
