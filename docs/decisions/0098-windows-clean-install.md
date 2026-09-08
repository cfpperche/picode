# ADR-0098: Windows clean-machine install — the desktop exe finishes the job

- **Status**: accepted (owner approved 2026-09-08; extends 0020, amends 0003's first-run boundary, extends 0093); amended 2026-09-08 — no paid signing, `install.ps1` is the install line (see Amendment)
- **Date**: 2026-09-08

## Context

ADR-0020 promised "install one thing on Windows, open the browser, PiCode is
there", and built most of it: `picode-desktop.exe` enables WSL, survives the
reboot, registers Ubuntu, creates the Linux account, drives `picode
provision` as root and as the owner, imports the mkcert root into the Windows
store, registers the logon task (ADR-0071), lives in the tray and updates
itself from the GitHub release. The release workflow already publishes the
exe next to `picode-linux-amd64`.

The chain still breaks on a machine that never had PiCode. Two facts, read
from the code on 2026-09-08:

| Gap | Where |
|---|---|
| The Linux `picode` binary never reaches the distro. `PicodePath` fails with "picode is not installed in the distro — install it in the distro first". | `internal/desktop/drive.go` |
| tmux, git, Node and `pi` are not installed. The provision step "pi is on PATH" is *blocked* with no fix, by design of ADR-0003. A fresh Ubuntu has none of them. | `internal/provision/steps.go` |

So today the exe works for the owner, whose distro already carries
everything, and stops one step short for anyone else. The owner asked
whether PiCode could ship as a Windows installer that bundles "the browser,
WSL and whatever else is needed" so a Windows user skips the setup.

Forces that shape the answer:

- **WSL is a Windows feature, not a payload.** It cannot be bundled; it can
  only be enabled (`wsl --install --no-distribution`, already done), and the
  first enablement needs a reboot. A distro, however, *can* be shipped:
  `wsl --import` registers one from a rootfs tarball without the Microsoft
  Store.
- **Every Windows 10/11 already has a Chromium browser** (Edge). ADR-0001
  refused Electron and Tauri; ADR-0072 gives the app an installed PWA
  identity. Bundling Chromium adds ~150 MB and a second update cycle for a
  window the OS already provides.
- **An unsigned exe is the loudest failure.** SmartScreen shows a red
  "Windows protected your PC" wall for an unsigned, low-reputation
  executable. No packaging format fixes that; a code-signing certificate
  does. Azure Trusted Signing is priced at roughly US$ 10 per month and
  runs in GitHub Actions.
- **ADR-0093 already accepted running `npm install -g <pkg>@latest` for pi
  on the user's behalf**, from the Agent CLIs surface. Doing the same on
  first run is the same argv in a different moment, not a new class of
  action. ADR-0003's objection was to *vendoring* pi, not to installing it
  with npm.
- **`internal/provision/container.go` already knows how to build a minimal
  root filesystem** with `debootstrap --include=ca-certificates,curl,git,
  tmux,nodejs,npm,procps` plus `npm install -g` of pi under
  `systemd-nspawn`. A CI job can produce that tarball once per release.
- The desktop is a Windows-only surface maintained from WSL; ADR-0020
  refused MSI/WiX for needing a Windows build host. GitHub's
  `windows-latest` runners remove that reason, but the exe already
  performs every installer duty (task, cert store, tray, update). What an
  installer would add is a Start-menu entry, an "Apps" uninstall row and a
  signed envelope.

## Decision

`picode-desktop.exe` stays the single Windows deliverable and completes the
clean-machine chain itself. Two stages join `internal/desktop.NextStage`
between `create-user` and `provision`:

1. **`install-picode`** — download `picode-linux-<arch>` from the release
   that matches the exe's own version (falling back to latest when the exe
   is an unstamped build), verify it against `SHA256SUMS`, write it to
   `~/.local/bin/picode` inside the distro as the owner, `chmod +x`, and
   check `picode --version` through a login shell. A binary already present
   at the same version is left alone.
2. **`install-runtime`** — as root: `apt-get install -y tmux git curl
   ca-certificates mkcert` (mkcert from Ubuntu's archive; if absent, the
   provision cert step keeps its existing blocked message), and Node.js LTS
   from the NodeSource apt repository pinned to the major that pi's package
   declares in `engines`. As the owner: `npm install -g
   @earendil-works/pi-coding-agent@latest`, the exact argv ADR-0093 runs.
   Each package is check → fix → verify; a tool already on PATH is not
   touched.

`provision` keeps ADR-0003's contract: its "pi is on PATH" step still only
*reports*, so a native Linux user who chooses their own pi is never
overridden. The desktop stage is what installs, and it runs only on a distro
the desktop itself registered or on an adopted distro **after the owner
confirms** in the same dialog that today confirms `wsl --shutdown`.

Distribution changes in two ways, both outside the exe: the release
workflow signs `picode-desktop-windows-amd64.exe` with Azure Trusted Signing
when the secret exists and publishes unsigned otherwise, and a winget
manifest (`cfpperche.PiCode`) is submitted per Stable release so `winget
install picode` is the documented install line. No MSI, MSIX or Inno Setup
is produced.

A **dedicated distro imported from a CI-built rootfs** (`picode-rootfs-
amd64.tar.gz`, produced by the same debootstrap recipe as `container.go`,
registered with `wsl --import picode`) is accepted as the *third* phase and
as an amendment to ADR-0020's "adopted, never recreated": adopt an existing
WSL 2 distro when there is one, import the PiCode distro when there is none
or when the user asks for isolation. It is not part of the first delivery.

Not done: bundling a browser or WebView. "Open PiCode" keeps opening the
default browser; a later, separate decision may add an `msedge --app=`
window mode if the PWA proves insufficient.

## Consequences

- **Easier**: a clean Windows 11 goes from download to the PiCode tab with
  one exe, one reboot (only when WSL was off) and no terminal. `winget
  install picode` is a one-liner support answer. A signed exe removes the
  SmartScreen wall, which is the biggest single drop-off for a non-developer.
- **Harder**: the desktop now carries package-manager knowledge (apt,
  NodeSource, npm) for one distro family. Ubuntu is the only supported
  auto-installed distro; on an adopted Debian/Fedora the runtime stage
  reports what is missing and stops, exactly like today. Downloads add a
  network dependency to a stage that was local; every download is
  checksum-verified and resumable by re-running the stage.
- **Cost accepted**: a yearly-recurring signing bill and a winget manifest
  to bump per release (`scripts/release-notes.mjs` gains a sibling). The
  release train in `docs/release-process.md` grows two steps.
- **Preservation contract (ADR-0020) holds**: `~/.picode` is never written,
  tmux sessions survive (apt and npm never touch the tmux server), an
  existing `picode` at the same version is not replaced, an existing pi is
  not reinstalled, and no stage ever runs `wsl --unregister`.
- **If we're wrong**: each stage is additive and idempotent. Removing the
  two stages returns the exe to today's behavior. Signing and winget are
  external and can stop without touching the product. Phase 3 is a separate
  approval.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Bundle Chromium / Electron / Tauri window | ADR-0001 stands; Edge is present on every supported Windows and the PWA (ADR-0072) already gives an app identity |
| Embed WebView2 in the exe for a native window | Needs cgo or a binding on a `CGO_ENABLED=0` cross-compiled binary; a tray icon and a browser tab still do not justify it (ADR-0020) |
| MSI / WiX / Inno Setup / MSIX | The exe already does the installer's work; a package would add a Start-menu row and an "Apps" entry at the price of a second packaging pipeline. winget gives discoverability without it. Revisit if the Store is ever a goal |
| Vendor pi, Node and tmux inside the exe | ADR-0003's objections (drift, user's `~/.pi` diverging) are unchanged; the exe would also swell by hundreds of MB per release |
| Prebaked rootfs as the *only* path (no adoption) | Breaks the owner's own machine and everyone who already runs PiCode in their distro; ADR-0020's adoption rule exists for them. Kept as phase 3, opt-in |
| Ubuntu's `nodejs` package instead of NodeSource | Ubuntu LTS ships a Node major that pi's `engines` may not accept; a pinned NodeSource major is the version the package declares |
| Leave `pi` to the Agent CLIs "Install" button (ADR-0093) | The button needs npm, which a clean Ubuntu does not have (the job fails with "npm was not found"), and it is a manual step inside the UI — the promise here is no manual step. First run has to close the loop itself |
| Ship unsigned and document the SmartScreen "More info → Run anyway" | Works for developers; the request is for users who should not need to know what SmartScreen is |

## Amendment 2026-09-08 — no paid signing; the install line is a script

The owner ruled out spending on signing for now, and two facts checked the
same day close the free alternatives: Azure Trusted Signing (now Artifact
Signing) validates organizations only in the USA, Canada, the EU and the UK,
so a Brazilian company cannot use it; SignPath Foundation requires an
OSI-approved license without commercial dual-licensing, which PolyForm
Noncommercial plus a commercial license is not.

SmartScreen's wall appears when Explorer runs a file carrying the Mark of
the Web, which browsers stamp on downloads. For an unsigned file the
reputation is per hash, so every release starts over. A file fetched by a
script and unblocked carries no mark; the logon task and the exe's own
self-update never carry one either. The wall is therefore a first-run
problem only, and the fix is to not hand the first run to Explorer.

Decision, replacing the distribution paragraph above:

- The documented install line is a PowerShell one-liner,
  `irm https://cfpperche.github.io/picode/install.ps1 | iex`. The script
  lives in the repo, is short enough to read, pins the release tag, verifies
  the exe against the release's `SHA256SUMS`, writes it to
  `%LOCALAPPDATA%\PiCode`, runs `Unblock-File` and calls
  `picode-desktop.exe install`. Nothing else is downloaded or executed.
- A winget manifest (`cfpperche.PiCode`, type `portable`) is still
  submitted, as a second door. winget stamps the mark on installers before
  running them and an unsigned exe has been seen to stop there; the
  manifest becomes the documented line only after a real `winget install`
  on a clean machine passes without a prompt.
- The GitHub release page keeps the exe with the sentence "More info → Run
  anyway" documented once, in the Windows guide.
- No signing job in CI, no signature check in the self-update. The
  `SHA256SUMS` check is the integrity boundary on both paths.
- Paid signing returns to the table only with a legal entity in a
  supported region or a budget for an OV certificate (around US$ 200 a
  year). This amendment does not change phases 1 and 3.

Relation to ADR-0093: that ADR refuses to *execute vendors'* `curl | bash`
installers on the user's behalf. Publishing PiCode's own script, which the
user chooses to run, is a different act; the constraints above (repo-hosted,
tag-pinned, checksum-verified, one binary) are what keep it defensible.
The alternatives row "Ship unsigned and document the SmartScreen wall" is
superseded: the script removes the wall for the documented path.

