import { useState } from "react";
import { IconCopy, IconPlay } from "./Icons.jsx";
import { langOf, highlightSource } from "../lib/highlight.js";
import { safeImgSrc } from "../lib/mdSafe.js";
import MermaidBlock from "./MermaidBlock.jsx";

const RUN = new Set(["bash", "sh", "shell", "python", "py", "javascript", "js", "go", "golang"]);

export function mdComponents({ CopyBtn, onRun }) {
  return {
    pre({ children }) {
      return <>{children}</>;
    },
    img({ src, alt }) {
      const url = safeImgSrc(src);
      if (!url) return null;
      return <img className="md-img" src={url} alt={alt || ""} loading="lazy" />;
    },
    code({ className, children }) {
      const text = String(children).replace(/\n$/, "");
      const lang = langOf(className);
      if (!lang && !String(children).includes("\n")) {
        return <code className="md-inline">{children}</code>;
      }
      if (lang === "mermaid") {
        return <MermaidBlock text={text} CopyBtn={CopyBtn} />;
      }
      return <SourceBlock lang={lang || "text"} text={text} CopyBtn={CopyBtn} onRun={onRun} />;
    },
  };
}

function SourceBlock({ lang, text, CopyBtn, onRun }) {
  const html = highlightSource(lang, text);
  const can = typeof onRun === "function" && RUN.has(lang);
  const [busy, setBusy] = useState(false);
  const [out, setOut] = useState(null);

  async function play(e) {
    e.preventDefault();
    e.stopPropagation();
    if (!can || busy) return;
    setBusy(true);
    try {
      setOut(await onRun(lang, text));
    } catch (err) {
      setOut({ ok: false, exit: 1, stdout: "", stderr: (err && err.message) || String(err) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="source-block">
      <div className="source-head">
        <span className="source-lang">{lang}</span>
        {can ? (
          <button type="button" className="copy-btn" title="Run in this agent" disabled={busy} onClick={play}>
            <IconPlay />
            {busy ? "Running…" : "Run"}
          </button>
        ) : null}
        {CopyBtn ? <CopyBtn text={text} /> : <PlainCopy text={text} />}
      </div>
      <pre className="source-body"><code className="hljs" dangerouslySetInnerHTML={{ __html: html }} /></pre>
      {out ? (
        <pre className={"source-out" + (out.ok ? " ok" : " err")}>
          {out.timedOut ? "timed out (15s)\n" : ""}
          {out.stdout || ""}
          {out.stderr ? (out.stdout ? "\n" : "") + out.stderr : ""}
          {!out.stdout && !out.stderr && !out.timedOut ? (out.ok ? "(no output)" : "exit " + out.exit) : ""}
        </pre>
      ) : null}
    </div>
  );
}

function PlainCopy({ text }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      className="copy-btn"
      title="Copy"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        } catch { /* ignore */ }
      }}
    >
      <IconCopy />
      {done ? "Copied" : "Copy"}
    </button>
  );
}
