import PageFrame from "./PageFrame.jsx";
import { commandDocUrl } from "../lib/commandDocs.js";

export default function Docs({ hidden, slug }) {
  const url = commandDocUrl(slug);
  const title = slug ? "/" + slug : "Docs";
  return (
    <PageFrame id="docs-view" title={title} hidden={hidden} wide>
      <p className="settings-desc">
        <a href={url} target="_blank" rel="noreferrer">Open in browser</a>
      </p>
      <iframe className="docs-frame" src={url} title={title} />
    </PageFrame>
  );
}
