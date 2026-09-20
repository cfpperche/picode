# Study: design mode / annotations — how the products do it, and what PiCode adapts

- **Date:** 2026-09-18
- **Sources (fetched live during the study):** Orca docs
  (`stablyai/orca @ main`: `docs/site/content/docs/browser/design-mode.mdx` and
  `browser/overview.mdx`, raw.githubusercontent.com — open repo, MIT), Lovable
  docs (`docs.lovable.dev/features/preview-toolbar`), bolt.diy source
  (`stackblitz-labs/bolt.diy @ main: public/inspector-script.js`). Plus the
  in-repo studies this one extends:
  [2026-09-10-browser-surface](2026-09-10-browser-surface.md) (Lovable, Replit,
  bolt, Orca — the preview→design-mode convergence) and
  [2026-09-06-cli-terminal-attach](2026-09-06-cli-terminal-attach.md) (paths,
  never bytes).
- **Owner requirement being served:** `docs/plans/desktop-v2.md` — *"Annotation
  screenshots — when the agent comments on a page, include the screenshot
  (Always include / ask / never) ... pairs with visual comments on DOM elements
  — ChatGPT's annotation mode"* (owner screenshots, 2026-09-12/13).
- **Not verified:** ChatGPT Work's own docs. Its release-notes page returns an
  empty document to an automated browser (JS/auth), DuckDuckGo and Mojeek served
  bot challenges, and the owner's history in our store has no `chatgpt` visit.
  Everything about ChatGPT below is the owner's recorded requirement, not a
  reading of the product — marked *owner-only*.

## The grammar four products converged on

| Product | How you point | What is captured | How it reaches the agent | The loop |
|---|---|---|---|---|
| **Orca** (live) | **Design Mode** toggle in the toolbar; cursor becomes a picker; hover highlights | element `outerHTML` + a small neighborhood, **computed CSS**, a **cropped screenshot**, and **file/line** when a dev source map exists | **"one attachment" into the active agent terminal** — the human then types what to change | agent edits → hot-reload → click again to verify |
| **Lovable** (live) | toolbar **modes**: Select elements (`S`), Edit text, Draw annotation, Comments; **multi-select** with Ctrl/Cmd-click | the chosen element(s) | **"attaches to the project chat input as a reference"**, the prompt applies to all of them | same, preview reloads |
| **bolt.diy** (source) | injected `inspector-script.js`: hover → highlight, click → message | `ElementInfo {tag, id, classes, computed styles, rect}` + a readable selector | chat attachment via `postMessage` | same |
| **ChatGPT Work** (*owner-only*) | visual comments on DOM elements | screenshot rides with the comment (the parity spec's wording) | not recorded | not recorded |
| Replit | no click-to-context; "Replit Design" carries design systems + Figma as context | — | text/image in chat | — |
| Cursor / Devin / Browserbase / browser-use | live view / take-over; element context comes from the **agent's own tooling** (*inference*) | — | agent tool | agent acts, human watches |

**Convergence:** (1) one mode switched on in the browser's own toolbar; (2) hover
highlight, click to pick (Lovable adds multi); (3) the same capture pack — DOM +
computed CSS + a cropped image (+ source file/line when available); (4) **the
result is one attachment in the conversation**, never a side panel; (5) the loop
is edit → reload → point again.

**Divergence that matters:** Lovable/bolt draw over an **iframe of the project's
own dev server** (zero latency, own project only); Orca uses a **real Chromium**
(any URL) and is therefore the reference we follow — WebView2, any site.

## What PiCode adapts

| Benchmark pattern | PiCode adaptation |
|---|---|
| One mode in the browser toolbar (Orca toggle, Lovable mode buttons) | An **Annotate** toggle in the tab's toolbar, picker over the **frozen page** (HTML can never paint over WebView2 — the still is already the mechanic the ⋮ menu uses) |
| Capture pack: DOM + computed CSS + crop (+ file/line) | Same three via the engine (`elementFromPoint` + computed styles + CDP `Page.captureScreenshot` with `clip`); **file/line deferred** — we have no source-map bridge |
| One attachment into the agent's input (all three) | **A staged file + its path, delivered through the existing prompt door** with the human's caption — not a new input path, and never auto-sent (ADR-0078) |
| Paths, never bytes (our own 2026-09-06 study) | The crop and a small markdown note land in `<terminal cwd>/.picode/drop/`; the row keeps the pointer |
| Screenshot policy (owner requirement) | The setting row **Always include / Ask / Never**; `Ask` is asked at creation time and an unanswered ask **proceeds without the image** (the inverse of the permission watchdog, which must deny) |
| Single vs multi select | **Single first** — Orca's shape; the file format holds a list so multi is additive later |

## Refusals

| Temptation | Why not |
|---|---|
| Auto-sending the annotation as a prompt | The element is context; the instruction is the human's sentence (Orca: "you type what you want changed") — and ADR-0078 refuses typing into a CLI's input |
| Image bytes in SQLite | The attach study's rule; the file system holds the bytes |
| Drawing mode in this slice | It is the natural follow-up (rectangle over the still) but the owner's ChatGPT requirement names **DOM elements** — element picking first |
| A separate annotations panel as the delivery | No product studied delivers there; the conversation is where the agent already is |

## Open

1. **Multi-select** — flip only if the owner's ChatGPT screenshots show it.
2. **Source file/line** — needs a dev-server/source-map bridge (Orca has one).
3. **Drawing mode** — rectangle over the frozen page; same substrate, second mode.
