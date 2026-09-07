import { useState } from "react";
import { hunksFromDiff } from "@picode/shared/domain/diff.js";
import { assetKind, assetSides } from "@picode/shared/domain/gitAsset.js";
import DiffLine from "./DiffLine.jsx";
import FilePreview from "./FilePreview.jsx";
import ImageLightbox from "./ImageLightbox.jsx";
import GitReadState from "./GitReadState.jsx";
import { gitURL, fileKind, isDeleted } from "../lib/git/model.js";
import { useGitRead } from "../lib/git/useGitRead.js";

function Asset({ url, label, kind, onZoom }) {
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  return <figure className="m-git-asset"><figcaption>{label}</figcaption>{failed ? <div className="m-git-state"><p>Could not load this version.</p><button className="btn" type="button" onClick={() => { setFailed(false); setRetry(n => n + 1); }}>Retry</button></div> : kind === "image" ? <button className="m-git-image" type="button" onClick={() => onZoom(url)} aria-label={`Enlarge ${label.toLowerCase()} version`}><img key={retry} src={url} alt={`${label} version`} onError={() => setFailed(true)} /></button> : <FilePreview kind={kind} src={url} />}</figure>;
}

function BinaryDiff({ owner, root, worktree, file, hash, parentHash, revision }) {
  const [zoom, setZoom] = useState("");
  const kind = assetKind(file.path);
  if (!kind) return <p className="m-git-hint">Binary file. No text diff is available.</p>;
  const status = ({ A: "added", D: "deleted", "?": "untracked", "??": "untracked" })[file.kind] || file.status || file.kind;
  const sides = assetSides(status);
  const versionURL = url => hash ? url : `${url}&v=${encodeURIComponent(revision)}`;
  const before = sides.includes("before") && (!hash || parentHash) ? gitURL(owner, "git/blob", { root, worktree, hash: hash ? parentHash : "HEAD", path: file.oldPath || file.path }) : "";
  const after = sides.includes("after") ? gitURL(owner, hash ? "git/blob" : "blob", { root, worktree, hash, path: file.path }) : "";
  return <div className="m-git-assets">
    {before ? <Asset key={versionURL(before)} url={versionURL(before)} label="Before" kind={kind} onZoom={setZoom} /> : null}
    {after ? <Asset key={versionURL(after)} url={versionURL(after)} label="After" kind={kind} onZoom={setZoom} /> : null}
    {!before && !after ? <p className="m-git-hint">No version is available.</p> : null}
    <ImageLightbox src={zoom} onClose={() => setZoom("")} />
  </div>;
}

export default function GitDiff({ owner, root, worktree = "", file, commit, nonce = 0, blocked, onMoved, onOpenFile }) {
  const [retry, setRetry] = useState(0);
  const remote = useGitRead(commit ? "" : gitURL(owner, "gitdiff", { root, worktree, path: file.path }), `${nonce}:${retry}`, onMoved, blocked);
  const data = commit ? file : remote.data;
  return <section className="m-git-detail" aria-label={`Diff for ${file.path}`}>
    <header className="m-git-file-title"><h2>{file.path.split("/").pop()}</h2>{file.path.includes("/") ? <p>{file.path.slice(0, file.path.lastIndexOf("/"))}</p> : null}<span>{fileKind(file.kind || file.status)}</span></header>
    {file.oldPath ? <p className="m-git-hint">Renamed from {file.oldPath}</p> : null}
    {!commit && !worktree && !isDeleted(file) && onOpenFile ? <button type="button" className="btn m-git-open-file" onClick={() => onOpenFile({ owner, root, path: file.path, view: "file" })}>Open file</button> : null}
    <GitReadState error={remote.error} loading={!data && remote.loading} onRetry={() => setRetry(n => n + 1)} />
    {data?.truncated || commit?.truncated ? <p className="m-git-warning">This diff is too large to show in full. Some changes are omitted.</p> : null}
    {data ? data.binary ? <BinaryDiff owner={owner} root={root} worktree={worktree} file={{ ...file, ...data }} hash={commit?.hash} parentHash={commit?.parents?.[0]} revision={`${nonce}:${retry}`} /> : data.patch ? <div className="diff m-git-diff" role="region" tabIndex={0} aria-label={`Changes in ${file.path}`}>
      {hunksFromDiff(data.patch).hunks.map((h, i) => <DiffLine key={i} h={h} />)}
    </div> : <p className="m-git-hint">No text changes.</p> : null}
  </section>;
}
