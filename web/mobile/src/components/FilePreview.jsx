import { useEffect, useId, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { previewEmpty, svgDataUrl } from "@picode/shared/domain/filePreview.js";
import { fileMessage } from "../lib/fileIO.js";

export default function FilePreview({ kind, text, src, html }) {
  if (kind === "html") return <HtmlPreview html={html} />;
  if (kind === "svg") return <SvgPreview text={text} />;
  if (kind === "mermaid") return <MermaidPreview text={text} />;
  if (kind === "markdown") return <MarkdownPreview text={text} />;
  if (kind === "image") return <Media src={src} tag="img" label="image" />;
  if (kind === "pdf") return <PdfPreview src={src} />;
  if (kind === "audio") return <Media src={src} tag="audio" label="audio" />;
  if (kind === "video") return <Media src={src} tag="video" label="video" />;
  if (kind === "model3d") return <ModelPreview src={src} />;
  return null;
}

// The one preview that is a page. The sandbox flags mirror the response's
// CSP sandbox (ADR-0136) — two locks on the same door. Never add
// allow-same-origin: the page must not see PiCode's origin or session, and a
// nested navigation must not move the app.
const PREVIEW_SANDBOX = "allow-scripts allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-pointer-lock allow-downloads";

function HtmlPreview({ html }) {
  if (!html || html.status === "idle" || html.status === "loading") {
    return (
      <div className="file-skel" aria-hidden="true">
        <div className="skel-line w-80" />
        <div className="skel-line w-50" />
      </div>
    );
  }
  if (html.status === "error") {
    return (
      <p className="file-pane-msg file-pane-msg-actions">
        <span>{fileMessage(html.error)}</span>
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => { void html.reload(); }}>Retry</button>
      </p>
    );
  }
  return (
    <div className="file-preview file-preview-html">
      <iframe
        className="file-preview-frame"
        title="HTML preview"
        src={html.frameUrl || html.url}
        sandbox={PREVIEW_SANDBOX}
        allow="fullscreen; clipboard-write"
        allowFullScreen
      />
    </div>
  );
}

function SvgPreview({ text }) {
  if (previewEmpty(text)) return <p className="file-pane-msg">Nothing to preview.</p>;
  const url = svgDataUrl(text);
  if (!url) return <p className="file-pane-msg">Can't preview this SVG.</p>;
  return (
    <div className="file-preview">
      <img className="file-preview-svg" src={url} alt="" />
    </div>
  );
}

function MarkdownPreview({ text }) {
  if (previewEmpty(text)) return <p className="file-pane-msg">Nothing to preview.</p>;
  return (
    <div className="file-preview file-preview-md">
      <Markdown remarkPlugins={[remarkGfm]}>{text}</Markdown>
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

function Media({ src, tag, label }) {
  const [err, setErr] = useState("");
  if (!src) return <p className="file-pane-msg">Can't preview this {label}.</p>;
  if (err) return <p className="file-pane-msg">{err}</p>;
  if (tag === "img") {
    return (
      <div className="file-preview file-preview-fit">
        <img className="file-preview-svg" src={src} alt="" onError={() => setErr("Can't preview this image.")} />
      </div>
    );
  }
  if (tag === "audio") {
    return (
      <div className="file-preview">
        <audio className="file-preview-audio" src={src} controls onError={() => setErr("Can't preview this audio.")} />
      </div>
    );
  }
  return (
    <div className="file-preview file-preview-fit">
      <video className="file-preview-video" src={src} controls onError={() => setErr("Can't preview this video.")} />
    </div>
  );
}

function PdfPreview({ src }) {
  if (!src) return <p className="file-pane-msg">Can't preview this PDF.</p>;
  return (
    <iframe className="file-preview-frame" title="PDF" src={src} />
  );
}

function ModelPreview({ src }) {
  const [ready, setReady] = useState(false);
  const [err, setErr] = useState("");
  useEffect(() => {
    let stop = false;
    import("@google/model-viewer").then(() => {
      if (!stop) setReady(true);
    }).catch(() => {
      if (!stop) setErr("Can't preview this model.");
    });
    return () => { stop = true; };
  }, []);
  if (err) return <p className="file-pane-msg">{err}</p>;
  if (!src) return <p className="file-pane-msg">Can't preview this model.</p>;
  if (!ready) {
    return (
      <div className="file-skel" aria-hidden="true">
        <div className="skel-line w-80" />
        <div className="skel-line w-50" />
      </div>
    );
  }
  return (
    <div className="file-preview file-preview-3d-wrap">
      <model-viewer
        className="file-preview-3d"
        src={src}
        camera-controls
        shadow-intensity="1"
        alt=""
      />
    </div>
  );
}
