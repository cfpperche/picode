import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter, drawSelection, dropCursor } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { indentOnInput, bracketMatching, foldGutter, foldKeymap, syntaxHighlighting } from "@codemirror/language";
import { classHighlighter } from "@lezer/highlight";

const chrome = EditorView.theme({
  "&": {
    height: "100%",
    backgroundColor: "var(--bg-base)",
    color: "var(--text-primary)",
    fontSize: "12.5px",
  },
  ".cm-scroller": { fontFamily: "var(--code)", lineHeight: "1.55" },
  ".cm-content": { caretColor: "var(--accent)", padding: "8px 0" },
  ".cm-gutters": {
    backgroundColor: "var(--bg-panel)",
    color: "var(--text-secondary)",
    borderRight: "1px solid var(--border)",
  },
  ".cm-lineNumbers .cm-gutterElement": { minWidth: "2.6em", padding: "0 8px 0 10px" },
  ".cm-activeLine": { backgroundColor: "var(--bg-hover)" },
  ".cm-activeLineGutter": { backgroundColor: "var(--bg-hover)", color: "var(--text-primary)" },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
    backgroundColor: "var(--accent-soft) !important",
  },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--accent)" },
  ".cm-foldPlaceholder": {
    background: "var(--bg-hover)",
    border: "1px solid var(--border)",
    color: "var(--text-secondary)",
  },
});

export function fileEditorExtensions({ lang, dark, onDoc, onSave }) {
  return [
    chrome,
    EditorView.theme({}, { dark: !!dark }),
    lineNumbers(),
    highlightActiveLineGutter(),
    highlightActiveLine(),
    drawSelection(),
    dropCursor(),
    bracketMatching(),
    indentOnInput(),
    foldGutter(),
    history(),
    syntaxHighlighting(classHighlighter),
    keymap.of([
      { key: "Mod-s", run: () => { if (onSave) onSave(); return true; } },
      ...foldKeymap,
      ...defaultKeymap,
      ...historyKeymap,
      indentWithTab,
    ]),
    EditorView.updateListener.of((u) => {
      if (u.docChanged && onDoc) onDoc();
    }),
    EditorView.lineWrapping,
    ...(lang ? [lang] : []),
  ];
}
