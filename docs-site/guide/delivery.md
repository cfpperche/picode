# Register deliveries from an agent

An agent can register a proposed change and ask for review using
`picode delivery`. The record names the branch, exact commit and target branch.
A review request does not mean someone approved the change or that checks passed.

Open the agent through PiCode's **Agent CLIs**, in the project's Git repository.
The command requires an updated PiCode binary and daemon, and the identity
inherited from that launch. This first version is a command and optional MCP
tool; the delivery screen and integration/deployment queues are not available.

## Register and request review

Run these commands in the agent's PiCode terminal. Replace the branch and title
with your change. Both the source and target must exist as local branches.

```sh
picode delivery capabilities
picode delivery register --title 'Fix navigation' \
  --branch feat/navigation --revision "$(git rev-parse feat/navigation)" \
  --target main --request-id navigation-create-1
```

The JSON response contains `delivery.id` and `delivery.version`. Use those values
in the next request:

```sh
picode delivery request-review --id DELIVERY_ID \
  --expected-version 1 --request-id navigation-review-1
picode delivery show --id DELIVERY_ID
picode delivery list
```

Each mutation needs a unique request ID. If a response is lost, retry the **same
command with the same request ID**. PiCode returns its original result. Do not
reuse that ID for changed content. Use `show` to obtain the current version before
a later update. Another writer changing the version causes a conflict rather
than overwriting their work.

## Update or withdraw

```sh
picode delivery update --id DELIVERY_ID --expected-version 2 \
  --title 'Fix navigation and focus' --branch feat/navigation \
  --revision "$(git rev-parse feat/navigation)" --target main \
  --request-id navigation-update-1
picode delivery withdraw-review --id DELIVERY_ID \
  --expected-version 3 --request-id navigation-withdraw-1
```

Every update clears the previous review request. A changed source branch appears
as `source.status: changed` in `show`; validation, integration and publication
remain `unknown` in this slice. `list` returns declarations without evaluating
Git evidence; pass its nonzero `nextBefore` value as `--before` for another page.

## Compatibility and limits

The common shell command does not depend on the agent vendor. It requires shell
execution, the inherited PiCode identity and daemon access. It uses the registered
launch folder, even after the shell changes directory. An agent started outside
PiCode has no registered identity and is refused. The identity names a launch,
not a particular conversation inside the vendor's CLI.

`picode mcp delivery` exposes the same actions as an optional MCP tool. Existing
PiCode launch configuration supports MCP injection for Claude Code, Codex and
OpenCode; other clients may use the command. Automated compatibility tests cover
the nine catalog identifiers and a generic terminal, not real authenticated
sessions with all nine vendors. Explicit tool selections must include delivery
through their MCP configuration; the current tool picker has no delivery option.

Only the original launch can change its declaration. Other launches in the same
repository can read it. Deleted launches leave historical records; this version
has no reassignment or deletion command. A repository accepts up to 1,000 records
and each principal up to 10,000 mutation receipts per repository.

`picode delivery --help` lists the actions. Successful calls print JSON; failures
return a nonzero exit status. No action merges code, runs checks or publishes it.
