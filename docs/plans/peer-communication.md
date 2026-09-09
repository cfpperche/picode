# Peer communication

Owner-approved implementation, 2026-09-09. Direct messages between existing
sessions; no task scheduling, recipient startup or automatic model turns.

## Delivery

1. Store authorized connections and durable messages. Scope contacts to the
   same workspace; an explicit connection represents one recorded conversation.
2. Expose four tools through an embedded, stateless Streamable HTTP MCP server
   using the official Go SDK. Authenticate every request with a connection token.
3. Add an owner-facing Messages tab to Agent CLIs, including managed Pi sessions,
   opt-in, revoke, connection setup and history. Preserve independent app views.
4. Exercise real HTTP MCP clients, persistence/retry/revocation, browser states
   on an isolated instance, scoped gates, close and main CI. No deploy.

## Decision table / acceptance

| Conditions | Action / evidence |
|---|---|
| No opt-in or invalid token (including auth off/loopback) | MCP refuses; HTTP tests |
| Scoped token used on owner API | Refuse, including auth off/loopback; auth test |
| Connection enabled for a recorded native conversation | Return secret once; store hash only |
| Recorded conversation changes, connection revoked, owner removed | Old credential cannot operate; store tests |
| Two opted-in connections in same workspace | Contact discovery and durable send |
| Different workspace, self-send or disabled recipient | Refuse without writing |
| Same sender and request ID, identical content | Return original receipt, no duplicate event |
| Same request ID, different content | Conflict, do not silently replace |
| Read repeated or daemon restarts | Same unacknowledged messages remain visible |
| Ack own received IDs | Record acknowledgement once, never implies task completion |
| Ack batch includes another inbox or unknown ID | Refuse whole batch, no partial acknowledgement |
| Reply references a message outside this pair | Refuse |
| Invalid/oversized input, inbox capacity exhausted | Refuse with bounded error |
| UI refresh fails | Retain history; block mutation until successful reload |
| UI owner switches while request in flight | Ignore stale response, discard one-time secret |

## Boundaries

A bearer capability identifies its enrolled conversation; MCP does not attest
which native CLI process holds it. Install it only for that conversation, never
in shared/global CLI configuration. PiCode refuses it once its recorded session
pointer changes; native CLI session discovery remains best effort (ADR-0084).
This is not proof of a live recipient. Setup is explicit; no native config files
are silently rewritten and no running process is restarted.

Communication remains pull-based. ADR-0106 adds launch-time credential
injection; native automatic notification and adapter installation remain outside
this increment.
Interoperability claims name the tested clients; six terminal integrations do
not imply six working MCP adapters.

## Launch integration (ADR-0106)

| Conditions | Action / evidence |
|---|---|
| Supported CLI, recorded conversation, automatic opt-in | Save private setup; return no bearer to browser |
| Pi adapter missing / unsupported automatic CLI | Reject before replacing the current connection |
| Setup file cannot be written | Revoke new connection; report retry; old connection stays revoked |
| Managed Pi restarts the same recorded conversation | Attach installed adapter and private registration extension |
| Pi changes native session or forks inside its process | Dispose registration; never register for the other identity |
| Terminal resumes exact recorded recipe | Inject only into the launched command |
| Fresh start, another session, fork or implicit continue | No conversation credential injected |
| Owner/workspace/session changed, token revoked | Do not attach stale setup; existing store checks refuse access |
| OpenCode already has inline JSON configuration | Preserve unrelated keys and MCP servers; resolved CLI override takes precedence over inherited environment; invalid/non-object JSON refuses launch (JSONC inline merging is not supported) |
| Private setup unreadable or token belongs to another owner | Fail launch visibly; never fall back to another credential |
| CLI exits to shell | Credential env is confined to exited child; shell does not inherit it |
| Disable / replace | Old credential fails immediately; history remains |

Native validation records distinguish config parsing, MCP handshake and model
turns. No untested vendor is described as runtime-compatible.

### Measured runtime evidence — 2026-09-09

| Runtime | Evidence | Limits |
|---|---|---|
| Pi 0.85.1 + pi-mcp-adapter 2.32.1 | Real managed Pi → tmux Pi → managed Pi; send/read/reply/ack in both directions; automatic setup through owner API and launcher | Explicit prompts initiated each turn; no wake |
| Claude Code 2.1.266 | Native model called list_contacts through the generated private MCP file | Scoped fixture capability; no full recorded Claude resume conversation roundtrip |
| Codex 0.153.4 | Native model called list_contacts through URL/env overrides | Fixture explicitly preapproved only list_contacts; launcher leaves native approval policy unchanged; upstream WebSocket 503 fell back to HTTPS |
| OpenCode 1.18.29 | Native mcp list reported the HTTP server connected | Handshake only, no model turn |
| Grok 1.0.24 | Installed help inspected: no per-launch MCP config override | Automatic wiring unavailable |
| Hermes Agent 0.21.1 | Installed hermes_constants.get_config_path resolves HERMES_HOME/config.yaml | Automatic wiring unavailable; no home overlay introduced |

Owned fixture artifacts: `var/screenshots/peer-launch/` (roundtrip receipts,
native client logs, browser matrix). Anthropic rejected model turns because
extra usage was exhausted; OpenAI's retired 5.4 variants were unavailable to the
account. The successful Pi roundtrip used openai-codex/gpt-5.6-luna.

Config references: [Codex](https://developers.openai.com/codex/config-reference/),
[OpenCode](https://opencode.ai/docs/mcp-servers/),
[Hermes](https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp/).
Pi runtime-registration contract verified from installed adapter 2.32.1 README
and source. Current tests cover the launch decision table, including private
file modes, atomic setup, native Pi switch/fork disposal, exact resume vs fresh
launch, stale credentials, rollback on setup failure and no secret in diagnostics.

HTTPS acceptance: a task-owned mkcert certificate served the scratch endpoint.
Managed Pi connected and called list_contacts using only its generated child CA
bundle. An unaided Node connection to the installed server failed certificate
validation, demonstrating why the explicit local trust attachment is necessary.
Unit coverage includes self-signed trust, hostname mismatch, remote endpoint
refusal and preservation/failure of existing CA bundles. CA lookup failures
happen before token replacement. Certificate/CA changes require replacing setup.
