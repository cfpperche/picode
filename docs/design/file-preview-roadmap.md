# File open + preview — roadmap

- **Date:** 2026-08-29
- **Status:** open. **Closes when tracks 1, 2 and 3 are shipped** (including 3D).
- **Why:** Ctrl+click in the terminal already opens a text tab (ADR-0019).
  SVG / mermaid have Preview | Raw | Save. Chat still uses the split pane.
  Relative paths use the terminal's *start* cwd. That is not the VS Code
  host behavior the owner asked for.

Shipped (not this plan): ADR-0019 text tabs, V1.1 `.svg` / `.mmd`.

## Sequence

| Order | Track | Start when |
|---|---|---|
| 1 | **Preview kinds** (png, pdf, md, audio, video, **3D**) | now |
| 2 | **Chat paths** open the same file tab | after 1, or in parallel if 1 is in review |
| 3 | **Live cwd** on Ctrl+click | after 1 (detector must know cwd) |

The plan is **done** only when all three rows are shipped. No fourth track.

## Chrome (every preview)

Same file tab, same chip group: **Preview | Raw | Save**. Default Preview
when we can render; Raw is always the text (or “Can't show this file.”).
Empty = one line + Raw. Error = one line + Raw. Save stays in the group.

Ctrl+click is host-only: it must **not** send SGR mouse (`<16;…m`) into the
TUI composer. Eat mousedown/mouseup on a link before xterm encodes them.

## Track 1 — Preview kinds

| Kind | Ext | How |
|---|---|---|
| Image | `.png` `.jpg` `.jpeg` `.gif` `.webp` | `<img>` via existing file bytes (cwd only) |
| PDF | `.pdf` | native `<iframe>` / `<object>` first |
| Markdown | `.md` `.mdx` | `react-markdown` already in the app |
| Audio | `.mp3` `.wav` `.ogg` `.m4a` | `<audio controls>` |
| Video | `.mp4` `.webm` `.mkv` | `<video controls>` |
| **3D** | `.glb` `.gltf` | **`@google/model-viewer`** — one web component, same idea as Excalidraw (popular primitive, not a homemade WebGL canvas). `three` only if model-viewer cannot load a file we already offer |

3D is **preview**, not an editor. No gizmos, no scene graph.

### Track 1 refuse

| Temptation | Why not |
|---|---|
| pdf.js until iframe fails | extra weight; browser already paints PDF |
| three.js scene from scratch | model-viewer is the Excalidraw analog |
| CAD / edit mesh | not a DCC |
| Preview for `.js` / `.jsx` | they stay the text editor |

## Track 2 — Chat paths (card + Open in tab)

Click a path in the conversation (`read` / `edit` / turn file names)
opens a **closable card in the thread** (inline preview or excerpt).
No Expand. **Open in tab** pops the same `#/file/a/<agentId>/<path>`
as Ctrl+click in the terminal (Preview | Raw | Save).

The chat split FilePane is gone — card or tab, not a third hatch.

Agent cwd. No Ctrl (already a button).

## Track 3 — Live cwd

Ctrl+click resolves against **the shell's current folder**, not the
cwd stored when the terminal was created.

**How:** on activate, ask tmux `#{pane_current_path}` for that session
(host query — no `PROMPT_COMMAND`, no OSC 7 in V1 of this track).
Absolute and `~/` stay as today. Path outside that live cwd: not a link
/ “outside this project.”

OSC 7 later only if tmux is wrong (rare).

## Done when

- [x] png / pdf / md / audio / video / glb|gltf Preview | Raw | Save
- [x] empty + error one-liners screenshot-read (visual-review)
- [x] chat tool path opens a closable in-thread file card; Open in tab uses the terminal file tab
- [ ] `cd` then Ctrl+click a relative path opens the file in the new folder
- [x] 3D uses model-viewer (or a written FAIL + three.js if it cannot)

Then mark this file **closed** and point Next up elsewhere.
