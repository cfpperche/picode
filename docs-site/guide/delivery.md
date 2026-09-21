# Follow changes in Delivery

Open a project's **Git** view and choose **Delivery**. Choose the target branch
(`main` is the common choice), then read each change using two separate facts:

| Label | Meaning |
|---|---|
| Integrated | The target branch already contains this revision. Whether the running version does is the Deployment lens's fact, shown as Published or Not published. |
| Not integrated | The target branch does not contain this revision yet. |
| Update needed | The change and target moved independently; inspect the history before integrating. |
| Checks passed | A clean full-project check covers the selected revision. |
| Relevant checks passed | The recorded checks cover only part of the project. |
| Checks not recorded | PiCode found no check evidence; it does not prove that no one ran checks. |
| Checked X seconds ago | When PiCode read Git and local evidence, not when tests ran. |

`Needs attention` filters the list; it does not create or order a work queue.
`View details` shows the revision and recorded evidence. `Open history` opens
the commit history for inspection. **Integrated** and **Checks passed** describe
the source and validation state; neither status confirms publication. What is
running is a third fact, read in the [Deployment](#deployment) lens and only for
the instance connected there: PiCode observes the local instance, and a remote
environment stays unknown.

An **Observed branch** was found in Git automatically. A **Registered delivery**
was declared by an agent through the command below. **Association not recorded**
means PiCode could not connect the current checkout to an active agent; it does
not identify the author.

## Deployment

The **Deployment** lens answers three questions about one environment: which
revision your PiCode instance is running, which integrated changes that revision
does not contain, and what the last recorded deploy attempt did. Open it from
**Git → Delivery** and choose **Deployment** beside **Integration**.

| Fact | Meaning |
|---|---|
| Running 0.3.1+abc1234 | The version and short revision of the artifact that answered. |
| Responding | The instance answered the health read. `Not responding` and `Health unknown` are different states. |
| checked 12 seconds ago | When PiCode read the instance and its receipts, not when anything was deployed. After 30 seconds the lane says the check is out of date. |
| Integrated, not published: 2 changes | The target contains them; the running revision does not. |
| Last attempt: Passed | The newest receipt recorded a finished deploy of this revision. |
| Last attempt: Failed · previous version is still responding | The attempt failed and an older revision answered; the two facts stay separate. |
| Last attempt: Unknown outcome | The attempt started and never recorded a result. It is not running and not a failure. |

An integrated change also carries a publication label in the **Integration**
lens.

| Label | Meaning |
|---|---|
| Published | The running revision contains this change's revision. |
| Not published | The target contains it; the running revision does not. |
| Publication unconfirmed | PiCode could not establish whether the running revision contains it. |

### Connect the instance

Set **Environment** to **This PiCode instance**. That is the whole
configuration: it binds this project's repository to the instance answering the
read. **Not connected** removes the binding, and the lane then says **Deployment
is not connected for this project.** with a link back to this page. The project
must be a Git repository; PiCode refuses to connect a folder that is not one.

One instance is one environment. PiCode observes the local instance only; remote
environments remain unknown. The lane reads the viewed project's own connection,
never another project's. One project may observe the local instance; when two
claim it the lane reports **Two projects claim this environment; disconnect
one.** instead of picking a side. Disconnect the other project's **Environment**
selector (Git → Delivery → Deployment → Environment → Not connected), or
disconnect this one.

**View evidence** names the running revision, the instance boot, whether the
artifact was built from a clean checkout, the attempt's own revision, and any
record the read could not accept.

### What it never claims

- No **safe**, **ready** or **approved** verdict. The lane reports what an artifact contains and what a producer recorded.
- No queue and no deploy button: it never starts, retries, cancels or rolls back a deployment.
- An unfinished attempt is **Unknown outcome**, never running and never failed.
- Receipts and health answers are local evidence. They are not tamper-proof and they authorize nothing.

### States you will see

| State | The lane says | Action |
|---|---|---|
| Not connected | Deployment is not connected for this project. | View setup |
| Identity unconfirmed | Running version found; included changes are unconfirmed. | View evidence |
| Connection points elsewhere | The connected environment points at another repository. | View setup |
| Two projects claim it | Two projects claim this environment; disconnect one. | View setup |
| No target chosen | Choose the branch changes will join. | Choose a target in the Integration lens |
| Agents still working | Agents are still working. | View activity |

## Register a delivery from an agent

An agent can register a proposed change and ask for review using
`picode delivery`. The record names the branch, exact commit and target branch.
A review request does not mean someone approved the change or that checks passed.

Open the agent through PiCode's **Agent CLIs**, in the project's Git repository.
The command requires an updated PiCode binary and daemon, and the identity
inherited from that launch. Open the project’s **Git → Delivery** view on desktop or mobile to follow
integration and deployment. Integration and deployment execution queues are not
available.

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
as `source.status: changed` in `show`; `show` also reports observed integration and recorded check evidence. `show` reads no
environment, so its `publication` field stays `unknown`; only the
[Deployment](#deployment) lens compares the running revision with the changes.
`list` returns declarations without evaluating
Git evidence; pass its nonzero `nextBefore` value as `--before` for another page.

## Compatibility and limits

The common shell command does not depend on the agent vendor. It requires shell
execution, the inherited PiCode identity and daemon access. It uses the registered
launch folder, even after the shell changes directory. An agent started outside
PiCode has no registered identity and is refused. The identity names a launch,
not a particular conversation inside the vendor's CLI.

`picode mcp delivery` exposes the same actions as an optional MCP tool. Existing
PiCode managed launches with default tool selection include this family for Claude
Code, Codex and OpenCode. Explicit tool selections remain unchanged; other clients
may use the command. Automated compatibility tests cover
the nine catalog identifiers and a generic terminal, not real authenticated
sessions with all nine vendors. Explicit tool selections must include delivery
through their MCP configuration; the current tool picker has no delivery option.

Only the original launch can change its declaration. Other launches in the same
repository can read it. Deleted launches leave historical records; this version
has no reassignment or deletion command. A repository accepts up to 1,000 records
and each principal up to 10,000 mutation receipts per repository.

`picode delivery --help` lists the actions. Successful calls print JSON; failures
return a nonzero exit status. No action merges code, runs checks or publishes it.

## Refresh and limits

Each row keeps integration separate from checks: a change may already be
integrated while the project checks have failed. Registered deliveries keep their
ID; other local branches appear as observed candidates.

The view refreshes while visible and keeps the last result if a read fails.
**Follow folder** explicitly adopts a moved project; a failed read never means the
queue is empty. **Needs attention** filters the observation list, not an execution
queue. Agent associations indicate who currently uses a checkout, not authorship.

For PiCode's own repository, `make ci-scoped`, `make ci` and `make land` record
future check/integration attempts automatically. Existing history is not recreated.
Other projects show Git facts and unknown checks until a supported evidence source
exists. The observer never runs a project's checks or deployment commands.
