---
description: Create, run, transfer, review, and recover a Mission across agents and sessions.
---

# Missions

A Mission keeps the desired result, acceptance criteria, decisions, updates,
and evidence when work moves between agents or sessions. You choose who is
responsible and whether to accept the result. You can create and review a
Mission without Pi or a paid model.

- **Where:** open **… → Missions** for a workspace, or search for **Missions**
  in the browser command palette. On a phone, use the workspace menu or
  **More → Missions**.
- **Before sending:** create or choose an agent in that workspace, then open
  and start it through its normal agent view. Missions does not start agents.
- **What it does not do:** it does not copy working files, infer completion
  from an idle agent, merge code, deploy, or run an unattended workflow.

## First Mission

1. Choose **Create mission**. Select a workspace; enter a title, the result
   you want, and at least one acceptance criterion. Put each criterion on
   its own line. **Next action** and **Context and decisions** are optional.
   For example, an objective could be “Make password recovery work on a
   phone,” with separate criteria for receiving the link and completing the
   flow on a phone.
2. Choose **Assign agent**. Select an existing agent in that workspace and
   confirm that it has no unrelated work or unsent input. An agent can be
   responsible for only one Mission at a time.
3. Open and start that agent normally, then choose **Send mission**. The
   agent receives a context packet with the objective, criteria, decisions,
   latest checkpoint, next action, and evidence references. For a CLI with
   no supported direct send, use **Copy context** and continue in its
   terminal manually.
4. Look for the agent's acknowledgment. If delivery says **unconfirmed**,
   inspect the agent before doing anything else. **Confirm receipt** records
   *your* confirmation only after you see that it received the packet. It
   does not ask the agent or send the packet again. PiCode never retries an
   uncertain send automatically.
5. Follow the Mission's **Next action**, answer questions in Inbox, and
   collect evidence. The agent can report updates and request review; you
   can also add updates and evidence in the Mission view.
6. After every criterion has current passing evidence, choose **Request
   review**. Check the result. Choose **Request changes** if it needs work;
   to accept it, first stop the executor and its child processes, then choose
   **Accept result**. A review request or an idle agent does not complete a
   Mission.

## Four independent signals

The status at the top, the assignment under the objective, the prompt
receipt, and the agent's activity answer different questions. For example,
**Ready for review** and **Assigned** can appear together: the result awaits
your decision while that agent still holds responsibility. **Assigned** does
not prove it is working.

| Signal | What it tells you |
|---|---|
| Mission status | Where the objective is in its lifecycle; see the table below. |
| **Assigned** / **Released** | Whether an agent still holds the Mission's responsibility. Pause, cancellation, and review can leave it **Assigned**. |
| Prepared / unconfirmed / acknowledged receipt | Whether a packet can be sent, may have been received, or was acknowledged. Submission alone is not acknowledgment. |
| Agent activity | Whether the agent appears busy or ready now. A ready or idle badge does not prove its child processes stopped writing. |

### Mission statuses

| Status shown | Meaning | Typical next step |
|---|---|---|
| **Ready** | Created, assigned but awaiting a confirmed receipt, or returned to a state that needs assignment. | Assign, start and send, or confirm a receipt you actually observed. |
| **In progress** | The assignment was acknowledged, or a paused Mission with an acknowledged assignment resumed. | Read updates, answer questions, add evidence, or request review. |
| **Needs you** | An agent reported a blocker or asked a question. | Open its Inbox question or choose **Record decision**. |
| **Ready for review** | Review was requested for the current evidence scope and candidate revision. | Check the criteria and evidence, then accept or request changes. |
| **Paused** | Further Mission sends are blocked; an assigned agent may still be running. | Stop the agent if needed, then resume or release it. |
| **Completed** | You accepted a reviewed result. Responsibility was released and the acceptance stays in history. | Archive it, or reopen it for new work. |
| `Cancelled` | Further Mission sends are blocked. Existing responsibility is retained until released. | Stop and release the agent, then archive or reopen if needed. |

## Read the Missions screen

| Control or section | What it does |
|---|---|
| **Filter by workspace** | Limits the list to one workspace; **All workspaces** shows the whole list. |
| **Include archived** | Shows completed or canceled Missions you archived. Archiving does not delete history. |
| **Create mission** | Opens the Mission form. **Cancel** in that form closes it without saving. |
| **Load more** | Adds the next page of Missions when the list has more than 100. |
| **Back** / **All missions** | Returns to the previous view / opens the list filtered to this Mission's workspace. |
| **Refresh** / **Reconnect** | Fetches the latest state after a change or connection loss. It never retries a prompt send. |
| Status and agent line | Shows the lifecycle status, workspace, responsible agent, CLI, and whether responsibility is **Assigned** or **Released**. |
| **Next action** / **Needs your attention** | Shows the saved next step or current blocker. The buttons below it are the available actions for this state. |
| **Acceptance criteria** | Shows each criterion, its latest evidence for the current scope and candidate, the reported pass/fail outcome, and the review revision. “Reported passing” is a claim to check, not an independent test by PiCode. |
| **Open Inbox notification** | Opens the latest linked question or review notice. Marking an Inbox item done does not accept the Mission. |
| **Context and decisions** / **Copy context** | Opens the continuation packet and copies it for manual handoff. The packet has bounded excerpts; use **History** for older actions. |
| **History** / **Earlier history** | Shows saved actions, who recorded them, and when. **Earlier history** loads another page; native CLI conversations are not copied into it. |

## Action buttons

Only actions valid for the current status and responsibility appear. Opening
an action shows its form; **Back** in the form leaves the Mission unchanged.
The confirmation boxes for target readiness, prepared files, receipt, or a stopped
source are statements you must verify yourself. They are cleared on a form
reload.

| Button | When and what it does |
|---|---|
| **Edit objective** | Available without a reserved agent. Changes the title, objective, criteria, context, or next action; starts a new evidence scope and returns the Mission to **Ready**. Review the latest saved objective before applying a recovered draft after a conflict. |
| **Assign agent** | Reserves one available agent in the Mission's workspace after you confirm it is ready and has no unrelated draft. The Mission stays **Ready** until receipt is acknowledged. |
| **Send mission** | Submits the prepared packet once to a supported, started agent. The receipt becomes **unconfirmed** before submission; inspect the agent if the outcome is uncertain. |
| **Confirm receipt** | Records your confirmation that the assigned agent received the packet. It changes the Mission to **In progress** and names you as the person who confirmed it. Use it only after checking the agent. |
| **Add update** | Saves a checkpoint in the Mission and optionally changes **Next action**. It does not send a new prompt to the agent. |
| **Record decision** | Saves your answer in **Context and decisions** and clears a blocker. An agent reads the updated Mission; the answer is not pasted into its terminal automatically. |
| **Add evidence** | Attaches a reported pass or fail to one criterion. Choose an observation/test result, a file path, or a Delivery ID; see [Evidence and review](#evidence-and-review). Adding evidence during review returns the Mission to **In progress** for another review. |
| **Request review** | Available during work or when blocked. Requires current passing evidence for every criterion. Records the candidate revision and makes the Mission **Ready for review**. |
| **Accept result** | Available in review. Rechecks current evidence and the candidate, requires your confirmation that the executor and its children stopped writing, releases responsibility, and completes the Mission. It does not merge or deploy. |
| **Request changes** | Records what needs fixing in context, returns the Mission to **In progress**, and starts a new evidence scope. Older evidence stays in history but cannot satisfy the next review. |
| **Transfer mission** | Reassigns a reserved Mission to another agent after you confirm the source and its children stopped, the target is ready, and required files are in its working folder. Preview the context and send it to the new agent separately. |
| **Pause mission** | Blocks further Mission sends and requests interruption for managed Pi. For other CLIs, stop work in the agent's normal terminal. Responsibility remains assigned until released. |
| **Resume mission** | Returns a paused Mission to **Ready**, or to **In progress** if its reserved assignment was already acknowledged. It does not start an agent or replay a prompt. |
| **Cancel mission** | Cancels future Mission sends and preserves files and history. Stop the executor separately; release its responsibility before archiving or reopening. |
| **Release agent** | Requires confirmation that the executor and its children stopped writing. Frees that agent for another Mission. An active Mission returns to **Ready** for a new assignment. |
| **Reopen mission** | Starts new work from a completed or canceled Mission after any reservation is released. Its history remains, while the current status becomes **Ready**. |
| **Archive mission** / **Restore from archive** | Hides a completed or canceled Mission from the default list, or shows it there again. The executor must be released first; history is retained. |
| **Restore workspace** | Reconnects a Mission to an existing workspace after releasing the executor. The repository must match the original one; for a plain folder, confirm the required files are prepared. Review must be requested again. |

## Evidence and review

Use **Add evidence** once or more for each acceptance criterion. The form
asks for the criterion, evidence type, value, and a reported **Pass** or
**Fail**. The most recent evidence for each criterion must pass in the
current scope and, for Git workspaces, the current commit.

| Evidence type | Enter | What PiCode checks |
|---|---|---|
| **Observation or test result** | What was observed, and how. | Records the reporter and text; it does not run the test for you. |
| **File in working folder** | A path relative to the assigned working folder, such as `reports/result.md`. | Confines the path to that folder, requires a regular file of at most 4 MiB, and stores its digest. |
| **Delivery ID** | The ID of an existing Delivery. | Checks that it belongs to the same repository and candidate commit. The link never changes its author. |

For a Git Mission, commit changes and keep the working folder clean before
adding evidence or requesting review. PiCode checks the current commit and
file/Delivery references again when you accept. If the commit changes after
review, add fresh evidence for the new commit and request review again.
**Request changes** also requires fresh evidence because it starts a new
scope. The owner decides whether reported results meet the objective.

A Mission without Git can keep research or documentation results and their
evidence without a commit. The evidence scope and owner acceptance still
apply. Acceptance is a saved historical result; reopening starts new work.

## Questions, transfer, and recovery

An agent can report a blocker that makes the Mission **Needs you**. Its
question appears in Inbox. **Record decision** there keeps the answer with
the Mission when it still belongs to the current assignment. An older
question can still be answered as history, but it does not reactivate a new
agent or paste the answer into a terminal. [Inbox tools](./inbox-tools)
explains the Inbox itself.

Before **Transfer mission**, stop the source and any child processes it
started. A ready or idle badge alone is not enough. Check that the new
agent has no unrelated work or input draft. If the working folders differ,
move or commit the needed changes yourself, open the same clean candidate
commit in the target folder, and confirm the files are prepared. Missions
never copies those files. The new agent receives a new assignment generation;
the previous session cannot report for it.

| If this happens | Do this |
|---|---|
| The send is unconfirmed, times out, or the daemon restarts | Inspect the assigned agent and its terminal. Confirm receipt only after observing it; do not send the same Mission again blindly. State and history survive a restart, but work does not start on its own. |
| The objective editor says the Mission changed | Review the latest saved objective shown there. Use **Apply draft to latest version** only after checking your preserved draft against it. For another action conflict, refresh the Mission and review its latest state before retrying. |
| A review says the candidate changed | Commit and clean the folder, add current evidence, then choose **Request review** again. |
| The working folder or agent is missing | Open the Mission history; stop and **Release agent** if still reserved. Use **Restore workspace** with an existing workspace in the original repository, or assign an available agent. |
| The assigned agent started a new native session | Stop the old writer and explicitly **Transfer mission** to that agent again. The new generation binds the new session; old reports are refused. |
| Direct sending is unavailable for this CLI | Use **Copy context** and continue manually in the agent's terminal. Have the assigned agent acknowledge the Mission before reporting. |

## Reporting from an agent

Pi launched by PiCode has a native `mission` tool in managed and interactive
sessions. It needs no MCP adapter or workspace extension install. Ask the Pi
agent to use `mission` with the Mission ID: its `show` action gives the current
version, assignment generation, and criterion IDs. Pi can then acknowledge,
report a checkpoint or blocker, attach evidence, and request review. The owner
still assigns, transfers, and accepts.

Other agent CLIs can use the optional MCP tool family enabled under **Agent
CLIs → Missions**, where their launch supports it, or the `picode mission`
command in their PiCode session. The command also works as a fallback for Pi:

```sh
picode mission show --id MISSION_ID
picode mission acknowledge --id MISSION_ID --generation 1 --expected-version 2 --request-id ack-1
picode mission report --id MISSION_ID --generation 1 --expected-version 3 --request-id update-1 --note 'Inspected the existing flow' --next-action 'Implement the selected option'
```

Use `picode mission --help` for the evidence, blocker, and review commands.
Replace the example version and generation with the values returned by `show`.
After a timeout, retry only with the *same request ID and exact payload*;
refresh the Mission after a version conflict. If a CLI sandbox blocks local
networking, use the configured Mission MCP tool or approve that specific
native command. The launch identity limits reporting to the current
assignment; it does not grant access to another agent's conversations.

## Limits and scope

A Mission supports up to 32 criteria, 128 evidence entries, and 2,000 work
updates. A full database backup preserves the ledger; **Copy context** is a
bounded continuation packet, not a backup. Separate reviewer assignment and
unattended dependent stages are later work. Missions does not turn other
CLIs into Pi managed mode.
