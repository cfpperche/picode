# Benchmarks — the bars we hold ourselves to

> Quality is not a vibe; it's a checklist backed by companies that set the
> standard. This file is the reference used by `/skill:quality-gate` and
> `/skill:uiux-review`. Update it when a benchmark stops serving us —
> explicitly, with rationale (it's a decision, see ADR process).

Delivery-flow research: [observing and governing agent delivery](benchmarks/2026-09-21-delivery-governance.md) compares review readiness, integration queues and deployment evidence; [execution baseline](plans/delivery-flow.md).

## Engineering benchmarks

Inspired by: **Google** (code review culture, small CLs), **Stripe**
(documentation as a product), **SQLite** (testing discipline), **Go team**
(stdlib-first minimalism), **Keep a Changelog / SemVer** (release hygiene),
**ADRs** (Michael Nygard) for decision records.

### The bar

1. **Build must never break.** CI is green on `main`, always. Work-in-progress
   that compiles + passes tests beats "finished" code that doesn't.
2. **Small, reviewable changes.** One logical change per commit/PR. If a diff
   needs a meeting to explain, split it.
3. **Tests travel with code.** New endpoint → handler test. New parser →
   table-driven test with edge cases. Bug fix → regression test first.
4. **Stdlib first.** Each non-stdlib dependency needs justification. "It's
   popular" is not justification; "mature, small, does what we can't" is.
5. **Docs change with code** (see AGENTS.md — non-negotiable #1).
6. **Visual gate on UI.** Screenshot read + overlay geometry audit +
   visual-card. Clipped menus are FAIL (see `/skill:visual-review`).
7. **Changelog discipline.** User-visible change ⇒ `[Unreleased]` entry,
   Keep a Changelog format, honest verbs (no "various improvements").
8. **Decisions are recorded.** Architectural choice ⇒ ADR. Superseded ADRs
   stay; history is evidence, not clutter.

## Documentation benchmarks

Inspired by: **Stripe** (docs as a product), **Diátaxis** (tutorials /
how-to / reference / explanation), **LibreChat** (self-hosted agent-product
IA and feature-page rhythm — see
[benchmarks/2026-09-14-librechat-docs.md](benchmarks/2026-09-14-librechat-docs.md)),
**VitePress** (Markdown → static HTML, heading anchors, local search —
Vite/Vue/Vitest), **pi** (`packages/coding-agent/docs`: command tables, no
duplicate source of truth).

Public user docs are Markdown in `docs-site/`, built by VitePress, hosted on
GitHub Pages. The app never hosts a docs viewer. See
[guidelines.md](guidelines.md).

### The bar

1. **One generator, not ours.** VitePress only. No handmade HTML site, no
   in-app iframe, no second copy in the React bundle.
2. **Reference has anchors.** Slash hints open `/commands#{id}` in a new tab.
3. **Pi correlation.** When a heading exists in pi, link the canonical doc
   and a same / changed / TUI-only table. Do not paste pi.
4. **Diátaxis-ish IA.** Getting started ≠ command reference ≠ internal ADRs.
5. **Short.** Tables over prose. Status of debts said plainly.
6. **Feature page rhythm (LibreChat).** A user-visible capability gets
   one public page: what it is, the UI path, how to enable it, what it is
   not. A how-to that can brick a deploy opens with one sentence that,
   remembered alone, does not.
7. **First run is not from source.** Getting started does not lead with
   the contributor toolchain. `make build` / Go / Node live on a from-source
   page.

## UI/UX benchmarks

Inspired by: **Cursor** (product + aesthetic north star — see
[benchmark-cursor.md](benchmark-cursor.md)), **t3code** and **paseo**
(architecture + composer-depth — see [benchmarks/](benchmarks/)),
**Linear** (speed, keyboard-first, dark-first, density), **shadcn/ui**
(Radix + Tailwind recipes), **Vercel/Geist**
(typography, minimal chrome, user menu, theme), **Stripe Dashboard**
(progressive disclosure, empty states that teach), **Apple HIG**
(clarity, deference to content), **xterm.js/ttyd** (terminal honesty).

**UI copy rule (owner directive, 2026-08-23):** documentation does not live
in chrome. Explanatory prose, setup steps and product storytelling belong
in README/docs — UI surfaces carry state and actions only (minimal empty
state = one line + one action; statusbar = live state, not hints).

### The bar — every UI change must satisfy

**One page width (the Agent CLIs standard, 2026-09-09)**
- [ ] Every desktop route renders through `PageFrame`, whose `.settings-wrap`
      is fluid up to **1240px**, centred, with shared gutters — the geometry
      ADR-0103 gave Agent CLIs. A page never picks its own width; embedded
      panes (inside Agent CLIs or dialogs) size to their container.
- [ ] Route map at the standard's adoption (before → 1240px):

  | Route | View | Width before |
  |---|---|---|
  | `#/clis/*` (CLIs, Settings, Packages, Messages) | `AgentClisFrame` | 1240px (reference) |
  | `#/system` | `System` | 680px |
  | `#/integrations` | `Integrations` | 680px |
  | `#/mcps` | `Mcps` | 680px |
  | `#/devices` | `Devices` | 680px |
  | `#/preferences` (+ sections) | `Settings` | 680px |
  | `#/pins`, `#/pins/new`, `#/pins/:id` | `PinStudio` | 680px |
  | `#/automations` | `Automations` | 1080px |
  | `#/llama/*` | `LlamaPanel` | 1080px |
  | `#/termset` (+ `/:id`) | `TermSettingsPage` | 1080px |

  Workspace surfaces (`#/`, `#/term/*`, `#/file/*`, `#/git/*`, `#/tree/*`) are
  canvases, not page frames — they keep their own layout. `#/app/*` is the
  exception since 2026-09-14: a primitives app's surface is a **page frame**
  (`settings-wrap` + `settings-head`, the view's tabs as an underline nav
  inside the card, the filter in the card toolbar, one line + one action for
  empty/blocked/error), so an app and a system route read as one product —
  `docs/plans/app-surface-parity.md`; native app surfaces (Canvas,
  ADR-0109) are still canvases, and own their container — `.native-surface`
  keeps the flex column the page frame dropped, without which `.cv-stage`
  measures 0px). Mobile is full-width by design
  (ADR-0072/0103).

**Control rhythm (shadcn `h-9` / HIG)**
- [ ] Adjacent controls share `--ctl-h` (36px): input + button in a row are
      the same height. Mismatched heights in one axis are FAIL.
- [ ] Buttons are `inline-flex` + `nowrap`; icon and label stay one line.

**Feel (Linear)**
- [ ] Interactions respond in <100ms; never a frozen frame without feedback.
- [ ] Keyboard-first: every primary action reachable without the mouse.
- [ ] Dark-first design; light mode is a derivative, not the default.
- [ ] Density with breathing room — power tool, not toy.
- [ ] **Optimistic UI:** in-flight fetches show layout skeletons or keep
      last good results. A blank content well while loading is FAIL.

**Clarity (Stripe/HIG)**
- [ ] Progressive disclosure: advanced options hidden behind a deliberate
      reveal, core flows naked and obvious.
- [ ] Empty states teach: first-run screens show *what this is* and *the
      one action to take next*.
- [ ] Destructive actions are confirmed; reversible where possible.
- [ ] Language: no unexplained jargon for our terminal-averse audience.
      PTY → "terminal integration". RPC → "control channel". If a real term
      must appear (tmux), add a one-line tooltip.

**Deference (HIG)**
- [ ] The agent's output is the hero; chrome recedes. No gratuitous
      animation competing with content.
- [ ] Scrollbars are overlay chrome: ≤8px, no arrow buttons, transparent
      track (VS Code / Cursor / Linear). Native Windows 17px+arrows is FAIL.
- [ ] Status is always truthful: streaming shows streaming, stuck shows
      why, unknown says "unknown" — never fake progress.

**Terminal honesty (ttyd)**
- [ ] The embedded terminal behaves like a terminal: selection, copy/paste,
      scrollback, resize. Users of the real Pi TUI must feel at home.

### Anti-benchmarks (things we refuse)

- Homemade widgets when Radix/cmdk/native already cover it (AGENTS.md Style).
- Empty list with no placeholder, or a "0" count on a collapsed empty group.
- Setup essays, architecture, or npm specs in chrome (those live in `docs-site/`).
- Disabled segmented controls with only a `title` tooltip; hide or explain.
- Tall empty `settings-card` (hug content).
- AI-slop UI: generic dashboard shells, 12-card grids of nothing, gradients
  for depth, spinner-only loading states, **blank wells while fetching**.
- Hiding the terminal to "protect" users (see philosophy: door, not cage).
- Modals for flows longer than 2 fields — wizards use full pages.

### Desktop app bar (ADR-0122)

The shell's windows carry a shell-owned 40px app bar (brand, drag region,
Windows caption buttons) above their content webview. Views never draw
window controls and never need to reserve space for them — the bar is the
shell's, and a browser sees none of it. In-page headers stay app chrome.
