import { javascript } from "@codemirror/lang-javascript";
import { python } from "@codemirror/lang-python";
import { json } from "@codemirror/lang-json";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { html } from "@codemirror/lang-html";
import { css } from "@codemirror/lang-css";
import { go } from "@codemirror/lang-go";

export function languageFor(path) {
  const n = String(path || "").toLowerCase();
  const i = n.lastIndexOf(".");
  const ext = i >= 0 ? n.slice(i) : "";
  switch (ext) {
    case ".js":
    case ".jsx":
    case ".ts":
    case ".tsx":
    case ".mjs":
    case ".cjs":
      return javascript({ typescript: ext === ".ts" || ext === ".tsx", jsx: ext === ".jsx" || ext === ".tsx" });
    case ".py":
      return python();
    case ".json":
      return json();
    case ".md":
    case ".mdx":
      // GFM: tables, task lists and strikethrough are part of the grammar.
      return markdown({ base: markdownLanguage });
    case ".html":
    case ".htm":
      return html();
    case ".css":
      return css();
    case ".go":
      return go();
    default:
      return null;
  }
}
