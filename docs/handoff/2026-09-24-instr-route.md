# 2026-09-24 — feat/instr-route: Instructions is a page route, not a tab

Owner's request, with two screenshots (the Instructions tab and the Agent CLIs page): Instructions opens like Agent CLIs, as a page over the tabs, and its address names the workspace it reads.

`#/instructions/<workspaceId>` now parses as its own route (`instructions`), and `App` renders `InstructionsSurface` with the other pages. `PageFrame` draws Back and the title, and the context line names the workspace and its folder (a new optional `contextIcon`, here a folder). The `i:<workspace>` tab plumbing is gone from `App`, `AgentTabs` and the inspector. A tab saved before this change is dropped on load (checked on scratch with a seeded `picode-tabs`). A workspace that no longer exists shows "That workspace is gone." with a way back. Opening a file from the page sets the file's address, so the editor tab is what shows.

Verified: `make close`; scratch `instrroute`: the menu opens the page at `#/instructions/<id>`, no Instructions tab appears, Back goes to `#/`, and the gone-workspace state renders. Visual review in a subagent. Nothing deployed.
