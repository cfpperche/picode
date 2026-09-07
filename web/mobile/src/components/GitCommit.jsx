import { useState } from "react";
import { gitURL, shortHash, historyDate } from "../lib/git/model.js";
import { useGitRead } from "../lib/git/useGitRead.js";
import GitReadState from "./GitReadState.jsx";
import { GitFileRow } from "./GitChanges.jsx";

export default function GitCommit({ owner, root, hash, onMoved, onCommit, onFile, onBack }) {
  const [nonce, setNonce] = useState(0);
  const { data, error, loading } = useGitRead(gitURL(owner, "git/commit", { root, hash }), nonce, onMoved);
  return <section className="m-git-detail" aria-label={`Commit ${shortHash(hash)}`}>
    <GitReadState error={error} loading={!data && loading} onRetry={() => setNonce(n => n + 1)} />
    {data ? <><h2 className="m-git-commit-title">{data.subject}</h2><p className="m-git-hint">{data.author} · {historyDate(data.at)} · <code>{shortHash(data.hash)}</code></p>
      {data.parents?.length ? <div className="m-git-parents"><span>{data.parents.length > 1 ? "Parents" : "Parent"}</span>{data.parents.map(parent => <button type="button" className="btn" key={parent} onClick={() => onCommit(parent)}>{shortHash(parent)}</button>)}</div> : null}
      {data.body ? <pre className="m-git-commit-body">{data.body}</pre> : null}
      {data.truncated ? <p className="m-git-warning">This diff is too large to show in full. Some changes are omitted.</p> : null}
      {data.files?.length ? <div className="m-git-files">{data.files.map(file => <GitFileRow key={file.path} file={file} onSelect={() => onFile(file, data)} />)}</div> : <div className="m-git-state"><p>This commit changes no files.</p><button className="btn" type="button" onClick={onBack}>Back to history</button></div>}
    </> : null}
  </section>;
}
