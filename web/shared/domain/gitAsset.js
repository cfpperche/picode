// Pure logic behind the git graph's binary asset previews: which extension
// kinds get a visual answer, which sides of the change to show, and the URLs
// to fetch them from. Kept out of the component so node:test can reach it.

import { previewKind, isBlobKind } from "./filePreview.js";

// assetKind: the preview family for one changed path, or "" when the diff
// card should keep its plain binary line. SVG never reaches here — git diffs
// it as text, so the text diff is already the better answer.
export function assetKind(path) {
  const kind = previewKind(path);
  return isBlobKind(kind) ? kind : "";
}

// assetSides: which versions of the asset the card shows. An added or
// untracked file has no earlier side to show; a deleted one no later side.
export function assetSides(status) {
  if (status === "added" || status === "untracked") return ["after"];
  if (status === "deleted") return ["before"];
  return ["before", "after"];
}

// gitRevBlobUrl: one asset at one revision. hash is a full object name or
// the literal "HEAD" — the server refuses anything else.
export function gitRevBlobUrl(base, ownerId, hash, path) {
  return (
    base +
    encodeURIComponent(ownerId) +
    "/git/blob?hash=" +
    encodeURIComponent(hash) +
    "&path=" +
    encodeURIComponent(path)
  );
}

// gitWorkBlobUrl: one asset as the working tree has it right now.
export function gitWorkBlobUrl(base, ownerId, path) {
  return base + encodeURIComponent(ownerId) + "/blob?path=" + encodeURIComponent(path);
}
