# Study: feedback when an agent is removed — catalog, insights, a loop that improves the agents

**Date**: 2026-09-23 · **Asked by**: the owner ("a feedback system when an agent is
removed: catalog it, develop insights the user can act on to configure the
environment, a system that feeds itself back and makes agents work better over
time"). **Decision**: [ADR-0194](../decisions/0194-agent-exit-records.md).
Closed-source claims are the vendors' own documentation or third-party reverse
engineering, marked as such; vendor numbers are self-reported.

## Why a new study

Removing an agent is the one moment PiCode knows an agent's story is over, and
today it forgets that story in the same click. `DELETE /api/agents/{id}` deletes
the row (`internal/store/agents.go`), `agent.deleted` carries only the id, the
agent's own events go with it (`events.agent_id … ON DELETE CASCADE`), the feed
keeps seven days, and the only copy of the configuration lives in the browser for
the eight-second Undo (`undoRemoveAgent`). Nothing can be learned from an agent
after it is gone.

## Who does what (fetched 2026-09-23)

| Product | Mechanism | What PiCode takes |
|---|---|---|
| [Claude Code `/insights`](https://angelo-lima.fr/en/claude-code-insights-command/) | On demand over local sessions. A small model extracts per-session "facets", cached locally: `outcome` (`fully_achieved` … `not_achieved`, `unclear_from_transcript`), *inferred* satisfaction, 12 friction types (`misunderstood_request`, `wrong_approach`, `user_stopped_early`, `tool_failed`…); skips sessions with < 2 user messages or < 1 minute ([schema, third-party reverse engineering](https://www.zolkos.com/2026/02/03/deep-dive-how-claude-codes-insights-command-works.html)). The report ends in CLAUDE.md rules tied to the sessions that motivated them | A fixed taxonomy; local only; every suggestion cites its sessions; the "worth asking" threshold |
| [Claude Code session survey](https://code.claude.com/docs/en/data-usage) | Inline "How is Claude doing this session? Bad / Fine / Good"; records the rating only; the transcript upload is a separate opt-in; `CLAUDE_CODE_DISABLE_FEEDBACK_SURVEY=1` | One click is enough. [Users ask to turn it off](https://github.com/anthropics/claude-code/issues/44568) — fatigue is real |
| [Devin Session Insights](https://docs.devin.ai/product-guides/session-insights) | Runs by itself **at teardown** for large (L/XL) sessions, on request for small ones, never below one Devin message. Issue timeline with impact, an improved prompt, machine/repo action items, knowledge that helped vs misled; buttons "Start new session" and "Go to machine" | The closest analogue to removal; a size threshold; each finding opens its door |
| [CodeRabbit Learnings](https://docs.coderabbit.ai/knowledge-base/learnings) | Learnings come from human replies; each keeps repo, path pattern, author, PR, usage count, last use. Dashboard with "active in 30 days" and "never used"; approval queue with a 0–30 day delay; credentials redacted | Provenance and usage on every learning |
| [GitHub Copilot Memory](https://github.blog/ai-and-ml/github-copilot/building-an-agentic-memory-system-for-github-copilot/) | The agent stores facts **with citations** (file:line), verified when used; unused facts expire after 28 days. A/B: PR merge 90% vs 83% | Validate at use, expire on disuse, measure by an outcome |
| [Greptile](https://www.greptile.com/docs/how-greptile-works/memory-and-learning) | 👍/👎, replies, and — unasked — whether a comment was addressed in later commits; suggests rules from repeated patterns. Reports addressed comments going from 19% to 55%+ ([self-reported](https://www.zenml.io/llmops-database/improving-ai-code-review-bot-comment-quality-through-vector-embeddings)) | Observed signals outweigh declared ones |
| [Warp](https://www.warp.dev/blog/agents-need-feedback-loops-not-perfect-prompts) | An emoji plus an optional note; a daily learning agent opens a **PR with the diff** to a skill file; a human reviews it. "Principles beat rules, because rules overfit and principles transfer" | Propose a diff, never apply alone |
| [Cursor](https://localskills.sh/blog/cursor-memories-guide) · [Augment](https://www.augmentcode.com/blog/how-we-built-memory-review) · [Windsurf](https://docs.windsurf.com/windsurf/cascade/memories) | Cursor shipped Memories with approval and **removed** them in 2.1 (Rules only; secondary sources). Augment built inline Memory Review because unreviewed memories piled up. Windsurf keeps automatic memories local; durable facts go to rules / AGENTS.md | Automatic learning without curation did not last |
| [Factory Agent Readiness](https://docs.factory.ai/agent-readiness/overview) | Scores a repository on 8 pillars and 5 levels; the 2–3 highest-impact fixes; `/readiness-fix` applies them | The **environment** is an object of insight, not only the agent |
| [Compound Engineering](https://github.com/EveryInc/compound-engineering-plugin) | `/ce-compound` writes what a task taught where the next plan reads it | The same loop, by hand |
| [Zed](https://zed.dev/docs/ai/agent-panel) | 👍/👎 per response; rating uploads the thread, so rating is consent | Explicit consent for anything that leaves |
| t3code (canonical) | Relays Codex's `/feedback` to OpenAI and shows the thread id (`packages/client-runtime/src/state/threadFeedback.ts`, `apps/web/src/components/chat/ComposerFeedback.tsx`) | Nothing local; Paseo and Cursor ask nothing on removal either |

No agent product studied asks anything when an agent is removed. The question
at removal is PiCode's own, which is also why there is no public response-rate
benchmark for it: PiCode measures its own from the first day.

## Research

- [ACE](https://arxiv.org/abs/2510.04618) (ICLR 2026): generator → reflector →
  curator, +10.6% on agents. Context grows as small deduplicated items, because
  rewriting the whole prompt each cycle erases detail and over-summarizes.
- [ReasoningBank](https://arxiv.org/abs/2509.25140): the gain comes from learning
  from failures too, not only from successes.
- [LangMem](https://langchain-ai.github.io/langmem/guides/optimize_memory_prompt/),
  [Letta](https://www.letta.com/blog/sleep-time-compute/): (trajectory, feedback)
  pairs become a prompt proposal; consolidation happens off the hot path.
- ["Failure as a Process"](https://arxiv.org/html/2607.09510v1) (1,794 CLI-agent
  trajectories on Terminal-Bench): 57.9% of failures are epistemic (false
  premise, ignored specification), 32.8% competence, 9.4% environment; the
  decisive error lands around step 7; 26% of agents claim success after locking
  in. **The reason a person gives at removal is a symptom, not a root cause.**
- ["Agent Skills Can Be Harmful"](https://arxiv.org/html/2608.11888v1): skills
  lowered pass rates on 16 of 84 tasks and cost up to +451% tokens. **A loop
  without measurement can make agents worse.**

## Asking at the exit

- Cancel flows ([Churnkey](https://churnkey.co/resources/customer-exit-survey/)):
  one multiple-choice question, optional text, and each reason routes to its
  own answer. In PiCode the "answer" is a configuration suggestion, never a
  barrier to the removal.
- Browser extensions ([`runtime.setUninstallURL`](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/API/runtime/setUninstallURL)):
  the survey opens **after** the removal, never before it.

## Eight patterns that converged

1. Removing is not failing: keep **outcome** apart from **reason**.
2. Ask little, never block; observed signals carry most of the weight.
3. One taxonomy for the human answer and for any machine analysis.
4. Every suggestion cites the evidence that produced it.
5. A person approves before behaviour changes.
6. Principles over rules, deduplication, expiry on disuse.
7. Measure the effect of every change.
8. Local by default; anything that leaves needs explicit consent.

## What PiCode adapts — the options put to the owner

| Option | What it is | Status |
|---|---|---|
| **A — exit record** | At a person's removal: a frozen snapshot of the agent's configuration, observed signals and an optional outcome/reason answer, written with the delete; a catalog with export; outcome numbers on the dashboard. No model calls | **Built first** (ADR-0194) |
| **B — computed insights** | Rules over the catalog with a minimum sample and provenance; each reason opens the door that already exists (CLI Settings, Model roles, Connectors, Snippets, Memory, checklist `always`, delivery review); one-click apply only for configuration PiCode owns; the insight shows in the Create dialog; before/after with n | Next |
| **C — review by a model at removal** | Devin's teardown analysis for large or unresolved agents, run through a CLI the person picks, returned over the picode MCP; inferred labels never overwrite the human one | Later; needs a CLI-neutral unattended runner (Automations `start` is Pi-only) |
| **D — periodic learning** | `/insights` + Warp + ACE: weekly diffs to instruction/memory files and presets, as principles, with evidence, approval, expiry and measurement | Later; security-model ADR (a reviewer reads every agent's transcript and writes repository files) |
| **E — cross-user comparison** | "Agents like yours resolve 80% with X" | Parked: telemetry and a server conflict with local-first and with no paid services now |

The owner took every recommendation on 2026-09-23: ask in the removal dialog;
ask only when the agent worked (at least one turn, at least one minute) and
offer "stop asking"; the taxonomy below; the loop closes first at creation
(presets, option B); no model calls now; warn that deleting sessions deletes
what the exit could be reviewed from; insights scoped per workspace with a
global view. Asking only at removal leaves out good agents that are never
removed — accepted for now and named in ADR-0194; B revisits it with observed
signals.

### The taxonomy (version 1)

| Outcome | Shown as |
|---|---|
| `resolved` | Resolved |
| `partial` | Partly |
| `unresolved` | Didn't resolve |
| `trial` | Just trying |

Reasons, asked only after `partial` or `unresolved`, several allowed:

| Reason | Shown as | Closest `/insights` friction · failure-process category |
|---|---|---|
| `setup` | Wrong setup | `wrong_file_or_location`, `tool_failed` · environment |
| `misunderstood` | Misunderstood the task | `misunderstood_request`, `wrong_approach` · epistemic |
| `stuck` | Got stuck or looped | `claude_got_blocked` · competence |
| `false_done` | Said done when it wasn't | `buggy_code` · epistemic (success claimed after lock-in) |
| `slow_costly` | Too slow or too costly | `slow_or_verbose` · — |
| `switched` | Switched to another agent | — (a revealed preference) |

## What PiCode explicitly does not copy

| Pattern | Why not |
|---|---|
| Memories or rules written without approval | Cursor removed them; Augment had to add review; a rule nobody approved is an invisible system prompt |
| Retention offers in the exit flow | PiCode has nothing to save; the removal never waits on the question |
| Uploading a transcript to anyone | PiCode has no server of its own; everything stays in the data dir |
| One rule per correction | Warp measured the overfitting |
| Inferred satisfaction treated as truth | `/insights` labels it an estimate; PiCode keeps the human label apart |

## Open questions

- Response rate: unknown until measured (`asked` vs answered is recorded from day one).
- Cost per agent: `internal/climetrics` measures by CLI and session, not by agent
  (ADR-0127); the exit keeps session pointers so it can be priced later.
