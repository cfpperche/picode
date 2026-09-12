# WSL control — seeing and reclaiming the disk

Status: **P0 and P1 landed** (2026-09-11, `feat/wsl-control`); P2–P4 are designed
here and not built. Owner approved the direction on 2026-09-11.

## The problem, measured

A Windows+WSL user watches disk space by hand because nothing tells them the
truth. On the machine this was written against:

| Fact | Value |
|---|---|
| `C:` free | 27 GB of 476 GB (94% used) |
| `<distro>/ext4.vhdx` | 218 GB on the volume |
| Used inside the distro (`df /`) | 126 GB |
| **Space Windows holds that the distro freed** | **≈92 GB** |
| The file is sparse? | **No** |
| `.wslconfig` | already `sparseVhd=true` |

Two things make that gap invisible. `sparseVhd=true` in `.wslconfig` applies to
**newly created** VHDs, so it never touched the existing file; and a VHDX is
only compacted or converted while the distro is stopped — which PiCode's own
keepalive child prevents, since it holds WSL open by design (ADR-0020). The
tray is therefore part of the cause, and the right place to fix it.

## Decisions taken (owner-approved)

| Question | Decision |
|---|---|
| Where do Windows facts come from? | **The tray reports them.** The Linux half stays ignorant of Windows paths and Win32 tools; `internal/desktop` keeps owning the Windows boundary, and the same process later runs the actions that need elevation. Interop (`/mnt/c/Windows/...` from inside the distro) stays a development convenience, not the contract. |
| v1 scope | See the two sides and reclaim what is safe (P0–P2). Compacting or moving the VHDX (P3) costs the tmux sessions and comes after. |
| Automation | Review first. A weekly safe prune can follow once the reviewed flow has run for real; nothing deletes without a confirmation before then. |
| Units | Binary units under the familiar label (`218 GB` = 218 GiB), because that is what File Explorer shows for the same file. Mixing SI and binary made the same disk read as 218 GB in one column and 203 GB in the next. |

## What landed (P0 + P1)

One deviation from the approved write-up, recorded here on purpose: P0 shipped
as a dedicated `disk` command on **both** binaries instead of a `doctor --disk`
flag — the tray and the later app consume the measurement as JSON, and a
report is not a provisioning check for `doctor` to own.

**`picode disk [--json]` — the Linux half, inside the distro**
(`internal/hostfs`, `cmd/picode/disk.go`). One `df` and two `du` calls: the
filesystem numbers, the reclaim table (what each cache costs and the exact
command that gives it back), and the top-level breakdown of home. Every
consumer carries a kind — `safe` (only time), `redownload` (comes back from the
network), `data` (the person's own; shown, never offered as a command) — and
the report names what it could *not* measure (root-owned paths such as
`/var/lib/docker`) instead of quietly folding it into a category. Read-only:
it prints and never deletes.

Two `du` calls, not one, because `du` skips a path it has already visited:
asking for `~/.cache` and `~/.cache/go-build` together reported the parent with
its biggest child removed (29 GB understated on the reference machine).

**`picode-desktop disk [--json]` — both halves** (`internal/desktop/disk.go`,
`cmd/picode-desktop/disk.go`). Windows facts never guess a folder: the VHDX
path comes from WSL's own registry (`HKCU\…\Lxss`, including `VhdFileName`),
the volume numbers from one PowerShell call that answers JSON (so no locale
decides what `.` means), and the sparse question from `fsutil sparse
queryrange`, whose parse reads the last two hex numbers on a line because every
word around them is translated on a non-English Windows. A failure on either
side keeps the other side's numbers and says which half failed — including the
common one, an older `picode` inside the distro.

**The tray line** (`cmd/picode-desktop/tray_windows.go`), refreshed every five
minutes:

```
WSL 218 GB · ≈92 GB held by Windows · C: 27 GB free
```

and `C: 9.0 GB free — low` under 20 GB, with the tooltip carrying the same
sentence plus the pointer to the full report. `fyne.io/systray` v1.12 has no
balloon API left, so the tooltip is the alert surface.

## Not built yet

### P2 — the app, and the facts the browser can see

A first-party app (`internal/apps/storage.go`, primitives — no shipped JSX) with
**Overview** (both sides in one screen) and **Reclaim** (the plan, reviewed and
confirmed). The tray posts its facts to `POST /api/host/windows` with the
install token read from the distro (`wsl.exe … cat ~/.picode/token`), and the
server serves `GET /api/host/usage` merging live distro facts with the last
reported Windows ones, stamped so the UI can say how old they are. **This is
the boundary that needs its own ADR** (a new route carrying another machine
half's facts, authenticated by the existing token): write it with P2, not
before.

Blocked state, which is the point of the feature: with no tray running the app
says so in one line and offers the action ("Start PiCode Desktop"), never a
zero.

Reclaim items are mutations, so they land with the store pattern (ADR-0048):
plan → confirm → job → **verified delta** measured by the same code that
reported the estimate, an event in the same transaction, and a decision table
for every row that can fail.

### P3 — the actions that need Windows

**Core shipped 2026-09-11** (`feat/wsl-actions`): the tray item **Give back ≈N
GB…** and `picode-desktop disk-compact`. Flow: readiness interlock (`GET
/api/deploy/readiness`, the deploy one) → confirmation naming the cost →
`wsl --terminate` → sparse conversion (`wsl --manage --set-sparse true`, no
elevation) → distro restarted → keepalive re-armed → before/after measured on
the file. `--dry-run`, `--yes`, `--force`, `--method optimize-vhd` (elevated
terminal; no Hyper-V on Windows Home). Decision table as implemented:

| Conditions | Action |
|---|---|
| Any agent mid-turn, or a terminal busy | Refuse, name them (tray dialog; CLI text; `--force` overrides on the CLI only) |
| Server not answering (no interlock possible) | Refuse — a missing answer is not consent |
| Held below the tray threshold, or already sparse and tight | Item greyed with the reason / "nothing is held" |
| WSL build without `--set-sparse` | Tray item points to the admin-terminal CLI; CLI demands elevation for Optimize-VHD |
| Distro running, everything idle, confirmed | Terminate → convert → restart → keepalive re-armed → measured delta shown |
| Conversion fails | Error reported; **the distro restarts anyway** (pinned by test) |
| Verified delta below what was promised | The measured after-number is the answer; the estimate is never quoted as the result |

Still open in P3: offering Optimize-VHD from the tray itself (needs an
elevation design that does not relaunch the tray as admin, since `elevate()`
re-uses the tray's own arguments), and cache prunes as reviewed actions (P2).

### P4 — history and the automatic path

One row per day (so "what grew this week" is answerable), a threshold alert, and
a coordinated *stop → compact → start* on a monthly cadence once P3 has run for
real a few times.

## Debts from P0/P1

- The merged report (`picode-desktop disk` with a live distro) is verified
  against a fake runner and against the real JSON contract run by hand; the
  end-to-end path needs the new `picode` **deployed** inside the distro.
- The tray line is a Windows notification-area surface: it cannot be
  screenshotted from WSL. Its wording is unit-tested; its rendering is the
  owner's `make desktop-restart`.
- The tray warns in words only. An alert icon needs a second generated asset
  (`mkicon.go` renders `icon.ico` today) and a visual pass that cannot happen
  from here.
- Docker's own storage is inside the same VHDX but is not measured here: the
  Docker app already owns prune with review (ADR-0065/0067), and duplicating it
  would be two doors to one delete.
