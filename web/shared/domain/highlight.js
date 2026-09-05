import hljs from "highlight.js/lib/core";
import javascript from "highlight.js/lib/languages/javascript";
import typescript from "highlight.js/lib/languages/typescript";
import xml from "highlight.js/lib/languages/xml";
import css from "highlight.js/lib/languages/css";
import json from "highlight.js/lib/languages/json";
import bash from "highlight.js/lib/languages/bash";
import python from "highlight.js/lib/languages/python";
import go from "highlight.js/lib/languages/go";
import rust from "highlight.js/lib/languages/rust";
import markdown from "highlight.js/lib/languages/markdown";
import sql from "highlight.js/lib/languages/sql";
import yaml from "highlight.js/lib/languages/yaml";
import dockerfile from "highlight.js/lib/languages/dockerfile";

const langs = {
  javascript, js: javascript, jsx: javascript,
  typescript, ts: typescript, tsx: typescript,
  xml, html: xml, svg: xml,
  css,
  json,
  bash, sh: bash, shell: bash, zsh: bash,
  python, py: python,
  go, golang: go,
  rust, rs: rust,
  markdown, md: markdown,
  sql,
  yaml, yml: yaml,
  dockerfile, docker: dockerfile,
};

for (const [name, def] of Object.entries(langs)) {
  if (!hljs.getLanguage(name)) hljs.registerLanguage(name, def);
}

export function langOf(className) {
  const m = /language-([^\s]+)/i.exec(String(className || ""));
  return m ? m[1].toLowerCase() : "";
}

export function highlightSource(lang, text) {
  const src = String(text || "");
  if (src.length > 100000) return escapeHtml(src);
  const id = lang && hljs.getLanguage(lang) ? lang : "";
  if (!id) return escapeHtml(src);
  try {
    return hljs.highlight(src, { language: id, ignoreIllegals: true }).value;
  } catch {
    return escapeHtml(src);
  }
}

export function escapeHtml(s) {
  return String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}
