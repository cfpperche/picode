# Study: the providers roster as a table — one row per account, aligned columns

- **Date:** 2026-09-11
- **Sources:** owner screenshot of `#/clis/providers/pi` (2026-09-11 12:41,
  2559×1599 px, ~150 % scale) measured by pixel scan; three live reads the
  same day — [Carbon Design System, Data table](https://carbondesignsystem.com/components/data-table/usage/),
  [MDN, `repeat()`](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Values/repeat),
  [Apple HIG, Settings](https://developer.apple.com/design/human-interface-guidelines/settings).
  In-house bars: [benchmarks.md](../benchmarks.md) (one page width, density,
  anti-benchmarks), [ADR-0031](../decisions/0031-provider-usage-dialog.md)
  (never a number we did not fetch), [ADR-0072](../decisions/0072-independent-web-applications.md)
  (two apps), and the roster this replaces:
  [2026-09-03-providers-view-v2.md](2026-09-03-providers-view-v2.md).
- **Scope:** the roster's geometry only — `#/clis/providers/pi` in both apps.
  The vault, the Add wizard, the Usage dialog, quota semantics and every
  endpoint stay as they are.

## The problem, measured

The v2 roster put identity and quota on one band and the controls on another,
with `justify-content: space-between` and a `flex: 1` gap. On the owner's
1706 CSS px window the card is 1180 px wide, so the two halves of every band
sat at opposite edges of it:

| Measured in the screenshot | Worse by |
|---|---|
| Largest empty run inside an account band | **847 px** |
| Identity line to its own quota gauges | **671 px** |
| Search field to Add provider | **805 px** |
| Height per provider (one account each) | **101–126 px** |

Ten signed-in providers is therefore ~1.5 screens of a page whose rows are
mostly air, and the eye has to cross the page to connect a login with its
own numbers.

## What the field says

| Source | Bar | PiCode's version |
|---|---|---|
| **Carbon — Data table** | A data table is for "displaying all of a user's resources", in one of **five row densities**, with the toolbar holding search *and* the primary action, and save-space variants (expandable rows) for the detail | The roster *is* "all of a user's resources": rows at 45 px, search and Add provider as one cluster, per-account rows instead of one card per provider |
| **MDN — `repeat()`** | `auto-fill` "repeats to fill a space … the largest number of repetitions that does not cause overflow"; `auto-fit` also collapses empty tracks | `minmax()` columns plus `container-type: inline-size`: "as many facts as fit", answered per container, not per viewport |
| **Apple HIG — Settings** | "Prefer letting people modify task-specific options without going to your settings area"; moving them away "disconnects it from its context" | Quota and its actions stay on the row — v2's decision, kept; the table only gives the reading a fixed place |
| **2026-09-03 roster study** | Quota belongs on the roster row (cc-switch, CodexBar, claude-swap); three honest states `live` / `stale` / `—`; traffic light at 70/90 | Unchanged: `QuotaStrip` and `providers-domain` helpers are the same code, moved into a column |
| **benchmarks.md** | One page width (1240 px, centred); density with breathing room; empty states are one line + one action; blank wells while fetching are FAIL; "card grids of nothing" are an anti-benchmark | The page width is untouched — the waste was *inside* the row. The roster keeps its loading, empty and error states |

## What PiCode adapts

1. **A row is an account, not a provider.** The provider is what its rows
   share, and each row stays readable on its own: the name (with its mark)
   is repeated on a second account row, dimmed and indented. One grid
   template is the layout's source of truth —
   `--prov-cols: minmax(120px,.8fr) minmax(190px,1.1fr) minmax(110px,1fr) 252px 84px 128px` —
   used by the heading, every row and the collapsed card.
   **Every track is fixed or `fr`, never `auto`**: an
   `auto` track resolves against that row's own content, which walks the fixed
   columns (the gauges) to a different x on every row. The first
   implementation did exactly that and the QA pass caught it — the ⋯ sat
   38 px further right on the two-account provider than on the rest.
2. **The quota reading gets a column, not an edge.** Two windows plus the
   `live`/`4m old` word fit a 252 px column; the percentage and the state
   word never shrink, the bar itself may (it has a printed number next to
   it), and a vendor reason truncates on one line with its text in `title`.
   Wrapping a long note under the buttons was the v2 defect: two lines, a
   ragged right column, 14 px of extra row height.
3. **Search and the primary action are one cluster** (Carbon's toolbar),
   left-aligned with the table they act on.
4. **Recently used becomes a line of chips**, not a second list of rows —
   one chip is `provider · Sign in · ×`.
5. **Column headings disappear with the rows**, and the empty roster carries
   exactly one Add provider button.

## Measurements after the change

Read from the scratch instance (1706×1000, ten providers, `#/clis/providers/pi`):

| | Before | After |
|---|---|---|
| Row height | 101–126 px | **45 px** |
| Largest empty run inside a row | 847 px | **10 px** (the column gap) |
| Columns identical on every row | no (buttons drifted) | **yes, header included** |
| Page scroll with ten providers | ~1.5 screens | **none** (list + llama.cpp + Recently used in one window) |
| Quota wrapped to a second line | yes (`Rate limited.`) | **no** |

## What PiCode refuses

| Temptation | Why not |
|---|---|
| A KPI row of totals over the roster | The anti-benchmark ("12-card grids of nothing"): the page has one question and the table already answers it |
| Card grid of providers, 2-up | Each card reproduces the void (`space-between` inside 590 px) and takes the gauges out of a comparable column |
| A third column (list on the left, detail on the right) | The shell already has the workspace rail; a settings tab inside Agent CLIs does not need a third level, and mobile degrades to one column anyway |
| Virtualization | Ten to forty rows; the page scroll is free |
| Hiding the usage column behind the Usage dialog again | v2's study (cc-switch, CodexBar) settled this: quota belongs on the row |
| A percentage for a provider with no endpoint | ADR-0031. The cell says "—" with the reason in `title`; the dialog keeps the detail |

## Debt this study leaves

1. **The mobile app duplicates `Providers.jsx` byte for byte** (one import
   differs: `ResponsiveDialog` vs `MobileSheet`). The CSS is shared now
   (`web/shared/styles/providers.css`, both `index.css` import it last), but
   the component still changes twice. Extracting it would cross ADR-0072's
   "each app owns its editor" line and needs the owner's call.
2. **The `.prov-row` list idiom is still used by `LlamaPanel`** (model rows)
   and by the llama.cpp pointer. It was not exercised in the scratch QA:
   no local router was running, so no model rows rendered. The rules moved
   byte-identical and `llama.css`'s `gap: 12px` override still wins, but the
   surface deserves one live look the next time a router is up.
3. **The 7-day spend column is empty on a machine with no recorded session
   cost** (`formatSpend` hides sub-cent noise). It is a real column on the
   owner's machine (deepseek, openai-codex); the scratch instance could not
   show it.
4. **`Verify with pi` answers from credential presence, not a real call.**
   During the QA pass a bogus `GROQ_API_KEY` still produced a green
   `pi can use this` (`pi auth check` reports what pi would *read*, and the
   env var is readable). Cursor, Raycast and Vercel verify with a real cheap
   request; making PiCode's badge mean that would change server semantics
   and needs the owner's call. The badge text is honest about what it checks
   once you know this — today nobody does.
