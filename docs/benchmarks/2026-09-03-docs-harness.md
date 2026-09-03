# Study: a documentation harness (theme, screenshots, API reference, prose gate, tutorial videos)

- **Date:** 2026-09-03
- **Sources:** browsed live on 2026-09-03 — [diataxis.fr](https://diataxis.fr) (front page, four-mode
  statement), [github.com/scalar/scalar](https://github.com/scalar/scalar) (About sidebar: MIT,
  16,045 stars, "Beautiful API References", 1st-class OpenAPI support), [remotion.pro/license](https://www.remotion.pro/license)
  ("free to use for individuals and companies up to three people; Company License required … for
  4+ people", Automators tier $100/mo minimum), [vale.sh](https://vale.sh) ("Open source · MIT",
  6.1k stars, "a linter for prose … entirely offline"; showcase: **Mintlify ships Vale as a
  built-in CI check**), [github.com/danielgtaylor/huma](https://github.com/danielgtaylor/huma)
  (MIT, 4,367 stars, OpenAPI 3.1 from Go handlers), [mintlify.com](https://mintlify.com)
  (positions as "The Knowledge Platform Built for Agents", ships `llms.txt`),
  [d2lang.com/tour/intro](https://d2lang.com/tour/intro/) ("diagram scripting language that turns
  text to diagrams", CLI render with dark/light themes). Local receipts: `www/.vitepress/config.mjs`
  (stock theme), `www/public/` empty at HEAD, 206 `mux.HandleFunc("METHOD /path", …)` registrations
  under `internal/server/` (Go 1.22 pattern syntax), ~50 internal screenshots in `docs/screenshots/`
  that never reach `www/`, prior study [2026-09-01-openwiki-repo-wiki.md](2026-09-01-openwiki-repo-wiki.md)
  (agent-maintained wiki + ETH Zurich finding), HyperFrames CLI 0.8.27 on PATH,
  `@playwright/cli` + `agent-browser` on PATH.
- **Scope:** a harness that makes the *public* docs (`www/` → VitePress → Pages) complete and
  beautiful, and keeps them cheap to maintain — screenshots, illustrations, API reference, prose
  gate, tutorial videos. Not the internal wiki question (OpenWiki study, still open) and not a
  redesign of the app chrome.

## The problem, in PiCode terms

The docs run but they are stock and thin: default VitePress theme, one flat sidebar group,
`www/public/` has no images at all, guides are prose-only, and the server's 206 API routes have no
published reference. Screenshots exist only as internal evidence. Nothing lints the prose, so
typos and format bugs ship (the 08-29 dead-link freeze; the 09-02 code-span freeze). Videos don't
exist, though the machine already has a full HTML→video stack installed. Every gap is maintained
by hand today; the harness should make the *right* artifact the cheap artifact.

## Benchmarks — who we studied and what we adapt

| Benchmark | What it is | What we adapt | What we refuse |
|---|---|---|---|
| **Diátaxis** | The four modes: tutorials, how-to, reference, explanation — organise around user needs | The site IA: four quadrant sections in the sidebar; every page declares its mode | Treat it as religion; guides may straddle modes with a callout |
| **Stripe** (bar already in `docs/benchmarks.md`) | Docs as a product: every feature ships with its doc, screenshots, example | Feature ↔ doc pairing check in the harness (commands and routes covered vs `www/`) | Stripe's infra (proprietary) |
| **Mintlify / Fern** | Hosted docs platforms: API playground, versioned reference, `llms.txt`, agent-ready positioning | Two ideas: embeddable **API reference** and **`llms.txt`** for agent consumption | The platform itself (hosted, paid, content leaves the repo); VitePress + Pages stays (bar: one generator, not ours) |
| **Scalar** | MIT, OpenAPI-first API reference UI, Vue-based (same ecosystem as VitePress) | The `/api` page: embed Scalar against a generated `openapi.json` | Swagger UI (dated), Redoc (license/runtime friction), Huma (would mean rewriting 206 route registrations to adopt a framework just for docs) |
| **OpenWiki / DeepWiki** (our 09-01 study) | Agent-maintained repo wiki; **verify-then-write**, staleness checks; ETH Zurich: LLM-written context *lowered* task success, human-written raised it | Agents run the *checks* (lint, links, staleness) and *draft* into the inbox; humans edit prose | Agent auto-committing public prose unreviewed |
| **Vale** | MIT prose linter, offline, config in repo; Mintlify ships it as a CI check | `make docs-lint` in `make ci`: house style (short sentences, no unexplained jargon, English) | Second linter; MDX-style bespoke rules |
| **D2 / Mermaid** | Diagrams-as-code with themed CLI render (D2: dark/light themes, elk layout); Mermaid renders in VitePress natively via plugin | Architecture/flow diagrams as committed source with a render target; dark-theme matches site | Hand-drawn PNGs, Figma exports that drift |
| **Remotion** | React-in-video, programmatic rendering | Nothing by default — license: free only ≤3 people, Company License above; a harness distributing rendered videos to users is exactly its paid tier | Adopting it silently; if the owner explicitly chooses it, fine (user-invoked, their license) |
| **HyperFrames** (installed here, CLI 0.8.27) | HTML→video composition with seek-safe timeline, workflows for product tours/explainers, TTS voiceover + captions (media-use skill), registry blocks | Tutorial videos authored as committed HTML compositions, rendered by the user/agent on demand to `www/public/videos/` | Bundling the renderer in the binary (ADR-0003: user-installed tools, detect on PATH) |
| **Playwright / agent-browser** (installed here) | Scriptable browser: screenshots, recordings | `make docs-shots`: drives the real app (or a fixture daemon) into framed, themed PNGs under `www/public/img/` | Manual screenshots that rot |
| **VitePress default theme** | Hero + features home layout, custom theme entry, CSS variables, local search, dark mode | Brand the site with the app's own tokens (`web/src/styles/app.css`), hero page, section landing pages | A bespoke theme engine (bar #1: one generator, not ours) |

## The harness, concretely

New files (all in-repo, make-driven, zero new runtimes):

```
scripts/
  docs-shots.mjs      # agent-browser/Playwright: named routes → framed PNGs → www/public/img/
  gen-openapi.go      # cmd: walks the mux registrations → www/public/openapi.json (+ coverage report)
  docs-llms.mjs       # builds llms.txt from www/ markdown (agent-ready index)
  docs-coverage.sh    # routes vs /api page, slash commands vs commands.md — report, not gate (v1)
www/
  .vitepress/theme/   # custom theme: app tokens, dark-first, section landing pages, home hero
  guide/api.md        # Scalar embed of openapi.json
  public/img/         # committed screenshots + rendered diagrams
  public/videos/      # rendered tutorials (committed MP4s, rendered on demand)
  llms.txt            # generated agent index
.vale.ini             # house prose style (English, short, jargon needs a plain-word line)
videos/               # HyperFrames compositions, one .md storyboard + one HTML per tutorial
```

Make targets: `make docs-shots`, `make docs-api` (CI fails if `openapi.json` is stale),
`make docs-lint` (vale + anchors + link check, joins `make ci`), `make docs-videos` (on demand).

## Phases

1. **IA + theme** (the "feia"): Diátaxis sidebar quadrants, custom theme with app tokens, home
   hero, section landing pages. No content moves lost — redirects kept.
2. **Screenshots + illustrations**: `make docs-shots`; first D2/Mermaid diagrams (architecture
   map, pairing flow, gateway topology); embed into existing guides.
3. **API reference + llms.txt**: route-walking `openapi.json` generator + Scalar page; `llms.txt`;
   `make docs-api` staleness gate in CI.
4. **Prose gate**: `.vale.ini` + `make docs-lint` in `make ci`; fix findings as they surface.
5. **Tutorial videos**: 3 HyperFrames compositions (getting started, pairing/devices, automations),
   storyboard committed next to the composition, rendered to `www/public/videos/`, embedded in
   the matching guides.
6. **(later, owner's call) agent maintenance loop**: an Automations template that runs
   docs-lint + coverage + staleness on a schedule and files inbox items with diffs — the OpenWiki
   pattern applied to public docs, humans still merge.

## Open questions for the ADR

1. Theme depth: pure default-theme CSS-variable pass, or a custom theme package? (Default first.)
2. Screenshots source: a fixture daemon (`picode demo`) vs the owner's real instance — privacy
   of inbox/agent names in shots argues for fixtures.
3. Do rendered videos get committed (binary weight) or built in CI Pages? MP4s of ~1–3 MB × 3
   are acceptable committed; CI rendering adds HyperFrames to Pages build (Node, already there).
4. Vale vocabulary: adopt Microsoft's style package (MIT) as base or hand-roll the ~20 rules we
   actually care about?
