import { useState } from "react";
import { IconCopy } from "./Icons.jsx";
import { langOf, highlightSource } from "../lib/highlight.js";

export function mdComponents({ CopyBtn }) {
  return {
    pre({ children }) {
      return <>{children}</>;
    },
    code({ className, children }) {
      const text = String(children).replace(/\n$/, "");
      const lang = langOf(className);
      if (!lang && !String(children).includes("\n")) {
        return <code className="md-inline">{children}</code>;
      }
      return <SourceBlock lang={lang || "text"} text={text} CopyBtn={CopyBtn} />;
    },
  };
}

function SourceBlock({ lang, text, CopyBtn }) {
  const html = highlightSource(lang, text);
  return (
    <div className="source-block">
      <div className="source-head">
        <span className="source-lang">{lang}</span>
        {CopyBtn ? <CopyBtn text={text} /> : <PlainCopy text={text} />}
      </div>
      <pre className="source-body"><code className="hljs" dangerouslySetInnerHTML={{ __html: html }} /></pre>
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
