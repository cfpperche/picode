import PageFrame from "./PageFrame.jsx";
import { COMMAND_DOCS } from "../lib/commandDocs.js";

export default function Docs({ hidden, slug }) {
  const doc = slug && COMMAND_DOCS[slug];
  if (!doc) {
    return (
      <PageFrame id="docs-view" title="404" hidden={hidden}>
        <p className="settings-desc">
          {slug ? "No documentation for /" + slug + " yet." : "Not found."}
        </p>
      </PageFrame>
    );
  }
  return (
    <PageFrame id="docs-view" title={"/" + slug} hidden={hidden}>
      <section className="settings-section">
        <h3>{doc.title || "/" + slug}</h3>
        <p className="settings-desc">{doc.body}</p>
      </section>
    </PageFrame>
  );
}
