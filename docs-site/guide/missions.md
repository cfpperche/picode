# Missions

A mission keeps an objective, acceptance criteria, decisions and evidence when
work moves between agents or sessions. You choose the responsible agent and
accept the result. Creating and reviewing missions does not require Pi or a
paid model.

## Create and assign

Open **… → Missions** in a workspace, or search **Missions** in the command
palette. On a phone, use the workspace menu or **More → Missions**.

1. Choose **Create mission**, describe the result and add one criterion per line.
2. Choose **Assign agent**. Check that the target is ready for this work.
3. Open and start the agent through its normal view, then choose **Send mission**.
4. Wait for the agent's acknowledgment. If receipt is uncertain, inspect the
   agent; use **Confirm receipt** only after checking. This records your own
   confirmation, not a provider acknowledgment.

A mission's state is separate from the agent's activity. Idle does not mean
completed. An uncertain send is never retried automatically.

## Questions, updates and transfer

Agents can record checkpoints and ask a mission question. The question appears
in Inbox. **Record decision** keeps your answer in mission context; the agent
reads it with `mission show`. An answer to an older assignment stays in history
for explicit review before reuse. Inbox Done does not accept a mission.

**Transfer mission** shows the context that will carry forward. Stop the source
and its child processes before confirming the transfer. For another working
folder, first transfer the changes yourself and open the same committed
revision there. Mission transfer does not copy working files.

**Pause mission** and **Cancel mission** stop future mission sends. Managed Pi
receives an interruption request; for other agents, open their terminal and
stop their current work. Responsibility remains reserved until you confirm
that writing has stopped. Files and history are preserved.

## Review the result

Add evidence for each criterion: an observation/test result, a file in the
working folder, or a Delivery ID for the same repository and candidate commit. Commit changes and keep the folder
clean before recording evidence.
Evidence is labeled as reported by the agent or owner. Commit Git changes
and keep the candidate folder clean before requesting review.

Choose **Request changes** to record findings and require fresh evidence, or
**Accept result** after checking the criteria, evidence and stopped executor.
A changed candidate or missing evidence blocks acceptance. Acceptance keeps
its historical snapshot; use **Reopen mission** for new work. Merge and deploy
remain separate actions. Linking Delivery never changes its author.

## Agent command

Run inside the assigned PiCode agent. `show` returns the current version,
assignment generation and criterion IDs:

```sh
picode mission show --id MISSION_ID
picode mission acknowledge --id MISSION_ID --generation 1 --expected-version 2 --request-id ack-1
picode mission report --id MISSION_ID --generation 1 --expected-version 3 --request-id update-1 --note 'Inspected the existing flow' --next-action 'Implement the selected option'
```

Use `picode mission --help` for evidence, blockers and review requests. After
a timeout, keep the same request ID and exact payload. Refresh after a version
conflict. Agent commands cannot assign, transfer or accept a mission.

The optional **Missions** tool family in Agent CLIs exposes the same reporting
contract through `picode mcp mission`. If a CLI sandbox blocks local networking, use this MCP tool or approve the
specific native command; keep the same request ID when retrying.

It uses the launch identity; it is not a
new security sandbox or permission to read other agents' conversations.

## Limits and recovery

A restart preserves state and history and never starts new work. A missing
workspace or agent leaves the mission readable; restore the folder or release
the stopped assignment and choose an available agent. **Restore workspace**
connects a mission to an existing workspace in the original repository; confirm
prepared files when restoring a plain folder. Archive keeps history.
Copy context for continuation; use PiCode's database backup for full history.

There are up to 32 criteria, 128 evidence entries and 2,000 versions per mission.
Files used as evidence must be regular files inside the working folder and at
most 4 MiB. Missions does not run unattended workflows or assign a separate
reviewer; those are later capabilities.
