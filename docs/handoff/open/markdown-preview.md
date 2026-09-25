# Markdown preview — GitHub parity, then an editor

The owner asked (2026-09-25) for a markdown viewer/editor on par with the
reference tools. Researched options, cheapest first: A) GitHub-grade preview,
B) VS Code-style source | preview split with scroll sync, C) Obsidian-style
live preview inside CodeMirror 6, D) WYSIWYG (Milkdown/MDXEditor/Tiptap) —
D refused for repository files because it re-serializes the file and rewrites
tables, lists and emphasis on save. A shipped on `feat/md-preview`
(`docs/architecture/file-preview.md`, "Markdown as a document").

## Next

- B: a Split mode beside Preview/Raw — CodeMirror source and MarkdownDoc side by side, scroll sync both ways from source line positions (`data-line` on blocks, as VS Code does), double-click in the preview jumps to the line.
- C: Live preview in CodeMirror (syntax hidden off the cursor line; references: blueberrycongee/codemirror-live-markdown, kenforthewin/atomic-editor — both young, adapt rather than depend).

## Debts

- [ ] Relative file links are inert text in canvas file panels and chat file cards (no `onOpenPath` at those mounts); tree, file tab and mobile Files open them.
- [ ] Opening `other.md#section` opens the file at its top; the fragment is not scrolled to after load.
- [ ] Raw HTML in markdown now renders after GitHub's sanitizer allow-list (no script, style, event handlers or unprefixed ids). No ADR was written: the owner may want one, since it widens what file content reaches the app DOM.
