import { useEffect, useId, useState } from "react";
import { previewEmpty, svgDataUrl } from "../lib/filePreview.js";

export default function FilePreview({ kind, text }) {
  if (kind === "svg") return <SvgPreview text={text} />;
  if (kind === "mermaid") return <MermaidPreview text={text} />;
  return null;
}

function SvgPreview({ text }) {
  if (previewEmpty(text)) {
    return <p className="file-pane-msg">Nothing to preview.</p>;
  }
  const src = svgDataUrl(text);
  if (!src) {
    return <p className="file-pane-msg">Can't preview this SVG.</p>;
  }
  return (
    <div className="file-preview">
      <img className="file-preview-svg" src={src} alt="" />
    </div>
  );
}

function MermaidPreview({ text }) {
  const rawId = useId().replace(/[^a-zA-Z0-9]/g, "");
  const id = "fp" + (rawId || "x");
  const [svg, setSvg] = useState("");
  const [err, setErr] = useState("");
  const body = String(text || "").trim();
  const theme = typeof document !== "undefined" && document.documentElement.dataset.theme === "light"
    ? "default"
    : "dark";

  useEffect(() => {
    if (!body) {
      setSvg("");
      setErr("");
      return undefined;
    }
    let stop = false;
    import("mermaid").then(({ default: mermaid }) => {
      if (stop) return;
      mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme });
      return mermaid.render(id, body);
    }).then((out) => {
      if (stop || !out) return;
      setSvg(out.svg);
      setErr("");
    }).catch(() => {
      if (!stop) { setSvg(""); setErr("Can't draw this diagram."); }
    });
    return () => { stop = true; };
  }, [body, theme, id]);

  if (!body) return <p className="file-pane-msg">Nothing to preview.</p>;
  if (err) return <p className="file-pane-msg">{err}</p>;
  if (!svg) {
    return (
      <div className="file-skel" aria-hidden="true">
        <div className="skel-line w-80" />
        <div className="skel-line w-50" />
      </div>
    );
  }
  return <div className="file-preview mermaid-svg" dangerouslySetInnerHTML={{ __html: svg }} />;
}
