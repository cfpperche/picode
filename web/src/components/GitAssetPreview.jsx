import { useState } from "react";
import FilePreview from "./FilePreview.jsx";
import ImageLightbox from "./ImageLightbox.jsx";
import { assetKind, assetSides, gitRevBlobUrl, gitWorkBlobUrl } from "../lib/gitAsset.js";

// One binary asset's visual preview inside a git diff card — the answer to
// "Binary file — no text diff." for screenshots, renders and recordings.
// Images show before|after side by side, click to zoom (GitHub's pattern);
// video, audio, pdf and 3D reuse the file pane's player on the changed side.
//
// With `hash` set the card reads a commit (before = parentHash, after =
// hash); without it, uncommitted changes (before = HEAD, after = the working
// tree). `fallback` renders when the extension has no visual answer.
export default function GitAssetPreview({ base, ownerId, path, oldPath, status, hash, parentHash, fallback }) {
  const [zoom, setZoom] = useState("");
  const kind = assetKind(path);
  if (!kind) return fallback || null;

  const sides = assetSides(status);
  const beforePath = oldPath || path;
  // A deleted asset's "before" lives at its old name; a renamed one's too.
  const beforeUrl = sides.includes("before")
    ? hash
      ? parentHash
        ? gitRevBlobUrl(base, ownerId, parentHash, beforePath)
        : ""
      : gitRevBlobUrl(base, ownerId, "HEAD", beforePath)
    : "";
  const afterUrl = sides.includes("after")
    ? hash
      ? gitRevBlobUrl(base, ownerId, hash, path)
      : gitWorkBlobUrl(base, ownerId, path)
    : "";

  if (kind === "image") {
    if (sides.includes("before") && !beforeUrl) {
      // A deleted asset in the repository's first commit: neither side exists.
      return <p className="diff-empty">No earlier version to compare — first commit.</p>;
    }
    return (
      <div className="gg-asset">
        <div className="gg-asset-pair">
          {beforeUrl ? (
            <AssetImage label="Before" src={beforeUrl} onZoom={setZoom} />
          ) : null}
          {afterUrl ? <AssetImage label="After" src={afterUrl} onZoom={setZoom} /> : null}
        </div>
        <ImageLightbox src={zoom} onClose={() => setZoom("")} />
      </div>
    );
  }

  // video / audio / pdf / model3d: one player on the side that exists going
  // forward — the new version, or the old one when the asset was deleted.
  const src = afterUrl || beforeUrl;
  if (!src) {
    return <p className="diff-empty">No earlier version to compare — first commit.</p>;
  }
  return (
    <div className="gg-asset">
      <FilePreview kind={kind} src={src} />
    </div>
  );
}

function AssetImage({ label, src, onZoom }) {
  const [err, setErr] = useState(false);
  return (
    <figure className="gg-asset-side">
      <figcaption>{label}</figcaption>
      {err ? (
        <p className="diff-empty">Can't load this image.</p>
      ) : (
        <button type="button" className="gg-asset-zoom" onClick={() => onZoom(src)} title="Zoom">
          <img src={src} alt={`${label} version`} onError={() => setErr(true)} />
        </button>
      )}
    </figure>
  );
}
