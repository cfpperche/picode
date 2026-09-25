# ADR-0166: Activating an account = writing the CLI's own credential file

- **Status**: accepted (owner directive, 2026-09-20)
- **Date**: 2026-09-20
- **Boundary**: persistence — PiCode now **writes** credential files owned by other tools, which it had only ever read; security model — a saved account's plaintext exists in the vendor's file again, and the file PiCode replaces is kept.

## Context

ADR-0165 shipped the vault and stopped there: the panes could hold, import and
verify accounts, but nothing was *used* — each CLI kept reading its own file,
and pi alone had an active slot (ADR-0013's `auth.json` copy).

The plan's step 2 proposed the other mechanism: a per-account directory per CLI
(`CODEX_HOME`, `GROK_HOME`, `CLAUDE_CONFIG_DIR`), so two terminals of one CLI
could run two accounts at once. The owner refused it, and for a practical
reason worth recording: that variable moves the CLI's **whole home** —
settings, sessions, memory, MCP configuration, plugins — so a PiCode-created
directory means the person's own state is no longer where the CLI looks. That
is a big change to a system that currently works, delivered for a benefit
(two accounts in parallel) nobody has asked for.

The field's answer to "use another account" for these CLIs is the one the
study found everywhere (cc-switch, claude-swap, the Codex account switchers,
AI Switcher): **write the other account's credential into the file the CLI
already reads**. Every one of those tools also warns about the same thing —
never swap while the agent is running — which this decision turns into a
refusal instead of a warning.

## Decision

Activation is per CLI, and it is the pi model generalized: the vault holds
every account, and **Use** writes the chosen one into the CLI's own credential
file, in that CLI's own shape, merging and preserving every key PiCode does not
own (`internal/clicreds.RenderLogin`, one renderer per declared format).

- **No HOME change, no config-directory variable, no per-account directory, no
  new environment variable.** The CLI is launched exactly as it is today; only
  the file it reads has different content.
- **The CLI's file stays the truth.** The pane marks a row as in use by
  fingerprinting what that file currently holds, not by remembering a click —
  a login made in the CLI's own TUI therefore shows up as the active row on
  the next load, and a row that was never activated shows as inactive.
- **One account per CLI is live at a time.** That is the cost of not touching
  HOME, and it is the same shape `gh auth switch` has: switching is
  machine-wide and affects terminals opened afterwards. Two Claude Code
  terminals on two accounts at once therefore stays out of scope — it needs
  the directory mechanism this ADR refuses.
- **A write is refused while a terminal of that CLI is live**
  (`liveTerminalsFor`, ADR-0062's runtime presence): the refusal names how
  many are running and what to do. Writing a credential under a running agent
  is the failure every tool in the field warns about.
- **The replaced file is kept before the first write.** The first activation
  for a CLI copies the existing file to `<DataDir>/credfiles/<cli>-<unix>.bak`
  (0600, never overwritten). The confirm names that file, so "put my login
  back" is one copy — not a re-login.
- **Nothing is invented.** A CLI whose store PiCode cannot faithfully write
  refuses the action with a reason in words and the pane hides the control:
  Omp (a live SQLite database), Grok when its file has no session yet (the map
  key is a vendor-issued issuer/client pair PiCode must not fabricate), Hermes
  and Muse API keys (their key paths live in `.env`, not in the file this
  writes), and any kind a renderer does not support.
- Writes are atomic (temp file + rename, 0600) and a file that does not parse
  is never clobbered — the write fails instead.

## Consequences

Easier: switching an account for Claude Code, Codex, Grok, Hermes, OpenCode,
Muse or Antigravity is now one click in the same pane where the account lives,
and pi keeps behaving exactly as it did. The panes also stop looking like two
different products: **Use** exists for every CLI now, which was the visible
asymmetry against pi's roster.

Harder and accepted: a credential now exists in two places — the vault (a
copy, encrypted at rest) and the vendor's file (plaintext, the CLI's own
permissions). That is the same exposure the machine already had before
PiCode; what changed is that PiCode is the one writing it. The vault's
encryption protects backups and copies, not the vendor's file.

Who breaks if we are wrong: someone who activates an account while a CLI runs
— refused, unless they closed it; a CLI that re-reads its file mid-session and
notices the swap (measured risk, which is why the refusal exists); and a
vendor that changes its credential format, which fails visibly at render time
(the pane shows the reason) rather than corrupting the file.

## Alternatives considered

- **Per-account config directories** (the plan's original step 2) — refused by
  the owner: it moves the CLI's entire home, so the person's settings,
  sessions and memory stop being where the tool expects them. Not re-measured
  here; the cost is structural and the benefit (parallel accounts) is not
  needed yet.
- **Environment variables only** (`ANTHROPIC_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`,
  `{PROVIDER}_API_KEY`) — narrower than this decision: it covers API keys and
  a couple of subscription tokens, but a ChatGPT or Claude Max login in Codex
  and Grok has no variable at all. The declarations already carry the variable
  names, so a launch-level injection can be added later without contradicting
  this ADR.
- **A proxy or router holding the pool** (ccflare, claude-code-router) — already
  refused by ADR-0003 and not revisited.
- **Swap the file with no liveness check** (what every comparable tool does) —
  refused: this product starts the processes, so it knows when one is running;
  a refusal it can compute beats a warning nobody reads.

## Amendment 2026-09-24 — a nameless login's row follows the CLI, and PiCode never spends its refresh token

A login whose bytes name no account (Claude Code's subscription) keeps one
vault row per provider, and that row and the CLI's file are the same login.
Two rules, measured on the owner's machine where the in-use Anthropic row had
answered 401 since 09-14 while Claude Code's own login worked:

1. **The row mirrors the file.** `credentials.Mirror` copies the CLI's live
   login into that row whenever they differ — on every roster read and before
   every usage fetch. It is read-only toward the CLI's file (writing there is
   still Use) and never creates a row; keyed and named logins are untouched.
2. **One rotating refresh token, one consumer.** When a CLI's live login holds
   a row's refresh token, PiCode does not refresh that row: an expired or
   rejected access token waits for the CLI, which renews on its next use.
   Refreshing it here would rotate the token out from under the CLI — the
   likely way the row died. pi is excluded: its file is written from the vault
   (ADR-0013).

Boundary: security model (who may spend a credential), no new persistence.

## Amendment 2026-09-25 — one refresh token, one CLI

The same rule between two CLIs: Use refuses (409, `heldBy`, `signInHere`) to
write a row whose refresh token another CLI's own live login already holds —
pi included, on both of its routes — because the first of the two to renew
would sign the other out. The pane offers a separate sign-in for the CLI
instead: the same account can hold two logins, each with its own refresh
token. A share made before this rule is named on the row (`sharedWith`, two or
more holders). Use on the CLI that already holds the login is not a share.
