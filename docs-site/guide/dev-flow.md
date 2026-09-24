---
description: The loop every PiCode change follows — from the first reading to what happens after deploy — and who drives each phase.
---

# Development flow

Every change to this repository follows the same loop: read narrowly, work in
an isolated worktree, close with docs, land on `main`, deploy, clean up after
deploy. Agents drive everything up to the fast-forward; the **owner** alone
lands and deploys. Most of it is not convention — git hooks refuse the
shortcuts and each `make` step checks the previous one.

The diagram is the map; the tables below it are the reference. An example
feature, `feat/cascade-delete`, carries through both.

<div class="devflow">

<section class="devflow-phase">
  <h2 class="devflow-head"><span class="devflow-num">1</span> Elaboration <span class="devflow-owner">agent</span></h2>
  <p class="devflow-sub">Read only what this change touches (ADR-0086) — never the whole handoff archive.</p>
  <div class="devflow-steps">
    <div class="devflow-step"><code>make handoff</code><span class="devflow-note">what is in flight, next up, open debts</span></div>
    <div class="devflow-step"><span class="devflow-kicker">Read by change type</span> UI → the uiux-review skill · handler/store → the <code>docs/architecture/</code> file · protocol or persistence → the ADRs it touches</div>
    <div class="devflow-step"><span class="devflow-kicker">Substantial feature</span> adapt and cite a <code>docs/benchmarks/</code> note (Cursor, t3code, paseo)</div>
    <div class="devflow-decide"><span class="devflow-q">Crosses a boundary</span> — protocol, persistence, security model, process?</div>
    <div class="devflow-branch"><span class="devflow-tag yes">yes</span><code>make adr NAME=…</code> seeds the record before the first edit</div>
    <div class="devflow-branch"><span class="devflow-tag no">no</span>straight on — a UI refinement or a route move needs no record; behavior changes still land in <code>docs/architecture/</code></div>
  </div>
</section>

<div class="devflow-arrow" aria-hidden="true"></div>

<section class="devflow-phase">
  <h2 class="devflow-head"><span class="devflow-num">2</span> Iteration <span class="devflow-owner">agent · isolated worktree</span></h2>
  <div class="devflow-steps">
    <div class="devflow-step"><code>make worktree NAME=cascade-delete</code><span class="devflow-note">isolated tree, hardlinked <code>node_modules</code>; feature commits on <code>main</code> are refused</span></div>
    <div class="devflow-step"><span class="devflow-kicker">Code and docs in the same commit</span> UI work passes the uiux-review checklist and a visual-review screenshot</div>
    <div class="devflow-step"><code>make ci-scoped</code><span class="devflow-note">the gates this diff can break</span></div>
    <div class="devflow-loop">a gate fails → fix, run again</div>
    <div class="devflow-decide"><span class="devflow-q">Interacting conditions</span> — delete / restore / auth / cascade / run mode?</div>
    <div class="devflow-branch"><span class="devflow-tag yes">yes</span>write the decision table; every row gets a test or becomes a named debt in <code>docs/handoff/open/</code></div>
    <div class="devflow-step"><code>make close</code><span class="devflow-note">gates (reusing a green run), regenerated OpenAPI, rendered board, fast-forward check, closing summary</span></div>
  </div>
</section>

<div class="devflow-arrow" aria-hidden="true"></div>

<section class="devflow-phase">
  <h2 class="devflow-head"><span class="devflow-num">3</span> Closing docs <span class="devflow-owner">subagent</span></h2>
  <p class="devflow-sub">Written from <code>make close-summary</code>, never at the working session's peak context.</p>
  <div class="devflow-steps">
    <div class="devflow-step"><code>docs/changelog.d/cascade-delete.md</code><span class="devflow-note">one Keep-a-Changelog fragment; <code>CHANGELOG.md</code> is assembled on <code>main</code></span></div>
    <div class="devflow-step"><code>docs/handoff/2026-09-22-cascade-delete.md</code><span class="devflow-note">session note ≤ 25 lines; next-up and debts reach the board for 7 days</span></div>
  </div>
</section>

<div class="devflow-arrow" aria-hidden="true"></div>

<section class="devflow-phase">
  <h2 class="devflow-head"><span class="devflow-num">4</span> Landing <span class="devflow-owner">owner · from the root</span></h2>
  <div class="devflow-steps">
    <div class="devflow-step"><code>make land BRANCH=feat/cascade-delete</code><span class="devflow-note">fast-forward <code>main</code> to the branch, then <code>make ci</code>; never commits, refuses an overlapping dirty tree</span></div>
    <div class="devflow-decide"><span class="devflow-q">Main moved</span> while the branch cooked?</div>
    <div class="devflow-loop">merge <code>main</code> into the branch, run <code>make close</code> again, land after</div>
    <div class="devflow-step"><code>make worktree-gc</code> · <code>make changelog</code><span class="devflow-note">remove merged worktrees; fold fragments into <code>CHANGELOG.md</code> before a release</span></div>
  </div>
</section>

<div class="devflow-arrow" aria-hidden="true"></div>

<section class="devflow-phase">
  <h2 class="devflow-head"><span class="devflow-num">5</span> Deploy <span class="devflow-owner">owner only</span></h2>
  <div class="devflow-steps">
    <div class="devflow-step"><code>make deploy</code><span class="devflow-note">rebuild, refresh stale captures, restart the service — serialized on <code>/tmp/picode-mutate.lock</code>; refuses while an agent or terminal is mid-turn</span></div>
    <div class="devflow-step"><span class="devflow-kicker">Verified before anything else mutates</span> daemon health checked after the restart</div>
    <div class="devflow-step"><code>var/deploy-log.jsonl</code><span class="devflow-note">deployment history lives in the data dir, not in git</span></div>
  </div>
</section>

<div class="devflow-arrow" aria-hidden="true"></div>

<section class="devflow-phase">
  <h2 class="devflow-head"><span class="devflow-num">6</span> After deploy <span class="devflow-owner">agent · next session</span></h2>
  <div class="devflow-steps">
    <div class="devflow-step"><code>scripts/qa-scratch.sh</code><span class="devflow-note">UI verified on a scratch instance, never on production</span></div>
    <div class="devflow-step"><span class="devflow-kicker">A fact that only exists after the merge</span> — the owner confirming it live, a debt observed paid — is one commit on <code>main</code> amending the note (ADR-0149)</div>
    <div class="devflow-step"><span class="devflow-kicker">Debt paid</span> flip <code>- [ ]</code> to <code>- [x]</code> in <code>docs/handoff/open/</code>; never delete another session's line</div>
  </div>
</section>

</div>

## What enforces the order

| Rule | Enforced by |
|---|---|
| No feature work on `main`, no switching the root checkout off it | the `reference-transaction` and `pre-commit` git hooks (`make hooks`) |
| Code and docs travel together | `make close-summary` flags a missing changelog fragment; the pre-commit hook refuses a direct `CHANGELOG.md` edit (the assembly, a release cut and its preamble are exempt), a fragment written on `main`, and a malformed fragment |
| One branch at a time | ADR-0105 — a branch reaches its fast-forward before the next starts; the owner decides when a session ends |
| Deploy is the owner's call | ADR-0105; `picode deploy` refuses while agents work; `--force` is the owner's deliberate one-off |
| Owner-grade restarts never overlap | `/tmp/picode-mutate.lock` — deploy, desktop restart, one at a time, each verified before the next |

## Guardrails that keep the shared tree alive

- **Stage explicit paths.** `git add -A` / `git add .` is how unrelated files
  ride into a commit; the pre-commit hook refuses conflict markers, but
  wholesale staging is still the failure mode.
- **Kill by exact name or PID only.** Never `pkill -f`, never
  `tmux kill-server` — a scratch instance is stopped with
  `scripts/qa-scratch.sh stop <name>`.
- **Scratch tmux is isolated.** `mkdir -p $dir && TMUX_TMPDIR=$dir tmux -L <unique-name>`;
  `$TMUX` outranks `TMUX_TMPDIR`, and a bare session lands in the owner's
  server.
- Screen evidence lives in `var/screenshots/` (gitignored), not in the repo's
  frozen history under `docs/screenshots/`.

## When the flow bends

Every step above can refuse — and a refusal names the state that makes the
step unsafe, it does not ask to be routed around. Each card quotes the message
the command actually prints; when a script is refactored its quote is what
goes stale first, so re-grep the phrase before trusting a card.

<div class="bend">

<div class="bend-item">
<p class="bend-when"><code>make ci</code> fails after the fast-forward</p>
<p class="bend-says"><code>land: make ci FAILED on main at &lt;sha&gt; — fix it (the branch is already merged; a follow-up branch is the honest fix).</code></p>
<p class="bend-do">Open a follow-up branch. Never rewind <code>main</code> — that guard exists because a stale fast-forward once erased merged work.</p>
</div>

<div class="bend-item">
<p class="bend-when"><code>make land</code> cannot fast-forward</p>
<p class="bend-says"><code>main cannot fast-forward to &lt;branch&gt;: either it is merged already, or it needs make close (merge main into the branch) first.</code></p>
<p class="bend-do">Merge <code>main</code> into the branch, run <code>make close</code> again, land after.</p>
</div>

<div class="bend-item">
<p class="bend-when"><code>make land</code> refuses the root</p>
<p class="bend-says"><code>the root checkout has local changes to N path(s) this branch also changes: …</code></p>
<p class="bend-do">Park or finish that overlapping edit first — <code>git add -A</code> is not a way past it.</p>
</div>

<div class="bend-item">
<p class="bend-when"><code>make deploy</code> refuses</p>
<p class="bend-says"><code>a restart would end 1 turn(s): terminal "codex" is working</code> … <code>Wait, or picode deploy --force (PICODE_DEPLOY_FORCE=1 for make deploy).</code></p>
<p class="bend-do">Wait for a quiet window. Force is the owner's call, and never fires while your own jobs are in flight.</p>
</div>

<div class="bend-item">
<p class="bend-when"><code>make handoff</code> names an invisible topic file</p>
<p class="bend-says"><code>handoff: a topic file is invisible — it lists bullets but none under a ## Next or ## Debts heading</code> … <code>Fix: add the heading, or delete the bullets if they are not open items.</code></p>
<p class="bend-do">Put the bullets under a heading the board reads. Deleting another session's line to silence the error is not a fix.</p>
</div>

<div class="bend-item">
<p class="bend-when">The board is over its target</p>
<p class="bend-says">A warning, never a gate (ADR-0145): it still renders, <code>make close</code> still passes.</p>
<p class="bend-do">Prune by paying debts, not by trimming what the team wrote down.</p>
</div>

<div class="bend-item">
<p class="bend-when">A session cannot finish</p>
<p class="bend-says">Nothing prints — the branch just sits there.</p>
<p class="bend-do">Leave the tree compiling and green; write the gap in <code>docs/handoff/open/&lt;topic&gt;.md</code>. <code>make worktree-status</code> marks it <code>stalled: …</code> — <code>no commits yet</code>, <code>nothing committed — empty branch</code>, or idle past a day.</p>
</div>

<div class="bend-item">
<p class="bend-when">A commit is refused over a living doc</p>
<p class="bend-says"><code>CHANGELOG.md no longer starts with # Changelog.</code> … <code>a parallel session likely wrote into this worktree</code></p>
<p class="bend-do"><code>git restore --staged --worktree &lt;file&gt;</code>, re-apply the edit, stage explicit paths.</p>
</div>

<div class="bend-item">
<p class="bend-when">A commit on <code>main</code> is refused</p>
<p class="bend-says"><code>Refusing to commit a new session note directly on main (ADR-0149).</code></p>
<p class="bend-do">A new note belongs to its branch. <code>main</code> accepts an amendment to a note that landed, or a new <code>docs/handoff/open/&lt;topic&gt;.md</code>.</p>
</div>

<div class="bend-item">
<p class="bend-when"><code>make vale</code> flags repo vocabulary</p>
<p class="bend-says"><code>error  Possible typo: '&lt;word&gt;'.  PiCode.Spelling</code></p>
<p class="bend-do">Add the word to <code>styles/config/vocabularies/PiCode/accept.txt</code> in the branch — the gate checks spelling, not style.</p>
</div>

<div class="bend-item">
<p class="bend-when">A docs link is dead</p>
<p class="bend-says">The build fails, so Pages never updates — it froze for days this way once.</p>
<p class="bend-do">Point the link at a real page; local examples are inline code, never bare URLs (<code>ignoreDeadLinks</code> covers only <code>localhost</code> and <code>127.0.0.1</code>).</p>
</div>

<div class="bend-item">
<p class="bend-when"><code>make worktree-gc</code> keeps a tree</p>
<p class="bend-says"><code>keep .worktrees/x (feat/x: written in the last hour; FORCE=1 to remove)</code> — or <code>not merged</code>, <code>dirty</code></p>
<p class="bend-do">Read the reason. Recent is not garbage; <code>dirty</code> means someone still needs it.</p>
</div>

<div class="bend-item">
<p class="bend-when">A debt outlives its note</p>
<p class="bend-says">Nothing prints; a note's sections drop off the board after 7 days.</p>
<p class="bend-do">Promote it to <code>docs/handoff/open/&lt;topic&gt;.md</code> and delete the note's echo in the same commit — one home per item.</p>
</div>

</div>

### Escape hatches

Each knob below disables a guard on purpose, for a caller who accepts what the
guard was preventing. The list is short because overrides are rare events.

| Knob | What it disables | Legitimate use |
|---|---|---|
| `PICODE_ALLOW_SWITCH=1` | the rule that keeps the root checkout on `main` — and, for that command, every `pre-commit` refusal | a deliberate one-off switch, then back to `main` |
| `PICODE_ALLOW_MAIN_REWIND=1` | the refusal to rewind `main` behind its tip | after a measured mistake, never to "undo" a merge casually |
| `picode deploy --force`, `PICODE_DEPLOY_FORCE=1` | the mid-turn refusal on deploy | the owner, when the fleet must restart anyway |
| `FORCE=1` | `make worktree-gc`'s "written in the last hour" keep | a tree that is merged, clean and genuinely abandoned |
| `--no-verify` | `pre-commit` for one commit | never as a shortcut around a refusal you have not read |

The mutation lock is not on this list: `flock` on `/tmp/picode-mutate.lock`
serializes deploy and desktop restart, force or not.

## What this page is not

Not the command reference — the build targets live in
[From source](/guide/from-source) and in
[AGENTS.md](https://github.com/cfpperche/picode/blob/main/AGENTS.md) in the
repository. Not the record of what shipped — that is the
[changelog](/changelog) and `git log`. And not negotiable prose: every rule
above traces to an ADR under
[`docs/decisions/`](https://github.com/cfpperche/picode/tree/main/docs/decisions),
and any decision can be re-argued with the owner — in a new ADR, never by a
commit that contradicts one silently.

<style>
.vp-doc .bend { display: flex; flex-direction: column; gap: 10px; margin: 16px 0 4px; }
.vp-doc .bend-item {
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  background: var(--vp-c-bg-soft);
  padding: 10px 14px;
}
.vp-doc .bend-when { margin: 0; font-size: 14px; font-weight: 600; line-height: 1.4; color: var(--vp-c-text-1); }
.vp-doc .bend-says { margin: 6px 0 0; font-size: 13px; line-height: 1.55; color: var(--vp-c-text-2); }
.vp-doc .bend-says code { font-size: 12.5px; }
.vp-doc .bend-do { margin: 8px 0 0; font-size: 13.5px; line-height: 1.55; color: var(--vp-c-text-1); }

.vp-doc .devflow { margin: 20px 0 12px; }
.vp-doc .devflow-phase {
  border: 1px solid var(--vp-c-divider);
  border-radius: var(--pi-radius-panel, 12px);
  background: var(--vp-c-bg-soft);
  padding: 14px 16px 16px;
}
.vp-doc .devflow-head {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  margin: 0; padding: 0; border: 0;
  font-size: 16px; line-height: 1.3;
}
.vp-doc .devflow-num {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; border-radius: 999px; flex: none;
  background: var(--vp-c-brand-1); color: var(--vp-c-bg);
  font-size: 13px; font-weight: 600;
}
.vp-doc .devflow-owner {
  margin-left: auto;
  font-size: 12px; font-weight: 500; letter-spacing: 0.02em;
  color: var(--vp-c-brand-1);
  border: 1px solid var(--vp-c-brand-soft);
  background: var(--vp-c-brand-soft);
  border-radius: 999px; padding: 2px 10px; white-space: nowrap;
}
.vp-doc .devflow-sub { margin: 4px 0 0; font-size: 13px; color: var(--vp-c-text-2); }
.vp-doc .devflow-steps { display: flex; flex-direction: column; gap: 6px; margin-top: 12px; }
.vp-doc .devflow-step {
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  padding: 7px 12px;
  font-size: 13.5px; line-height: 1.55;
  color: var(--vp-c-text-1);
}
.vp-doc .devflow-note { display: block; font-size: 12.5px; color: var(--vp-c-text-2); margin-top: 2px; }
.vp-doc .devflow-kicker {
  display: block; font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.05em;
  color: var(--vp-c-text-3); margin-bottom: 2px;
}
.vp-doc .devflow-decide {
  font-size: 13.5px; line-height: 1.55;
  padding: 0 2px 0 2px; color: var(--vp-c-text-1);
}
.vp-doc .devflow-decide .devflow-q {
  font-weight: 600; color: var(--vp-c-brand-1);
}
.vp-doc .devflow-branch {
  font-size: 13px; line-height: 1.55;
  color: var(--vp-c-text-1);
  border-left: 2px solid var(--vp-c-brand-1);
  padding: 2px 0 2px 12px; margin-left: 4px;
}
.vp-doc .devflow-tag {
  display: inline-block; margin-right: 8px; padding: 0 8px;
  border-radius: 999px; font-size: 11px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.05em;
  color: var(--vp-c-brand-1); background: var(--vp-c-brand-soft);
}
.vp-doc .devflow-tag.no {
  color: var(--vp-c-text-2);
  background: transparent;
  border: 1px solid var(--vp-c-divider);
}
.vp-doc .devflow-loop {
  font-size: 12.5px; font-style: italic;
  color: var(--vp-c-text-2);
  padding: 0 2px 0 12px; margin-left: 4px;
  border-left: 2px dashed var(--vp-c-divider);
}
.vp-doc .devflow-arrow {
  display: flex; flex-direction: column; align-items: center;
  margin: 2px 0;
}
.vp-doc .devflow-arrow::before {
  content: ""; width: 2px; height: 12px; background: var(--vp-c-divider);
}
.vp-doc .devflow-arrow::after {
  content: ""; width: 0; height: 0;
  border: 5px solid transparent; border-top-color: var(--vp-c-divider);
}
.vp-doc .devflow code { font-size: 0.92em; }
</style>
