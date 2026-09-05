import { useEffect, useId, useState } from "react";

export default function MermaidBlock({ text, CopyBtn }) {
  const rawId = useId().replace(/[^a-zA-Z0-9]/g, "");
  const id = "mmd" + (rawId || "x");
  const [svg, setSvg] = useState("");
  const [err, setErr] = useState("");
  const theme = typeof document !== "undefined" && document.documentElement.dataset.theme === "light"
    ? "default"
    : "dark";

  useEffect(() => {
    let stop = false;
    import("mermaid").then(({ default: mermaid }) => {
      if (stop) return;
      mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme });
      return mermaid.render(id, String(text || "").trim() || "graph TD; A[empty]");
    }).then((out) => {
      if (stop || !out) return;
      setSvg(out.svg);
      setErr("");
    }).catch((e) => {
      if (!stop) { setSvg(""); setErr((e && e.message) || "Invalid diagram"); }
    });
    return () => { stop = true; };
  }, [text, theme, id]);

  return (
    <div className="source-block mermaid-block">
      <div className="source-head">
        <span className="source-lang">mermaid</span>
        {CopyBtn ? <CopyBtn text={text} /> : null}
      </div>
      {err ? <p className="source-out err">{err}</p> : null}
      {svg ? <div className="mermaid-svg" dangerouslySetInnerHTML={{ __html: svg }} /> : null}
    </div>
  );
}
