# Windows clean-machine install

Scope: [ADR-0098](../decisions/0098-windows-clean-install.md). Make
`picode-desktop.exe` finish on a Windows machine that never had PiCode, sign
it, and publish a winget manifest. Phase 3 (imported PiCode distro) is
designed here but starts only after the owner approves it separately.

Do not: bundle a browser, add a WebView, produce MSI/MSIX/Inno packages,
touch `~/.picode`, replace a `picode` or `pi` already at the wanted version,
run `wsl --unregister`, or change `picode provision`'s "pi is on PATH" step
from report to fix.

## Phase 1 — close the chain (exe only)

### 1a. `install-picode` stage

- `internal/desktop/bootstrap.go`: add `StageInstallPicode` after
  `StageCreateUser`; `MachineState` gains `PicodeVersion string` (empty when
  `command -v picode` fails through a login shell as the owner).
  `NextStage` returns the stage when the version is empty or older than the
  exe's stamped version. Unstamped exe (`version.Stamped != "release"`)
  installs only when absent.
- `internal/install/release.go`: `ReleaseFor(tag, asset)` next to
  `LatestReleaseFor`, so the Linux binary comes from the **same tag** as
  the exe. `SHA256SUMS` from the release is fetched and the asset is
  verified before it crosses into the distro.
- Transfer: write to `%TEMP%\picode-linux-amd64`, then
  `wsl.exe -d <distro> -u <user> -- sh -c 'mkdir -p ~/.local/bin && cp
  /mnt/c/<temp path>/picode-linux-amd64 ~/.local/bin/picode && chmod +x
  ~/.local/bin/picode'`. Verify with `sh -lc 'picode --version'`.
  (`/mnt/c` mounts may be off in `wsl.conf`; fall back to streaming through
  stdin: `wsl.exe ... -- sh -c 'cat > ~/.local/bin/picode'` with the file as
  the child's stdin.)
- `Describe(StageInstallPicode)`: "install PiCode inside the distribution".

### 1b. `install-runtime` stage

- `StageInstallRuntime` after `install-picode`. `MachineState` gains
  `Runtime RuntimeState{Tmux, Git, Curl, Mkcert, NodeMajor, Pi bool/int}`
  filled by one `sh -lc` probe as the owner (single wsl.exe round trip).
- Root pass (`-u root`, `DEBIAN_FRONTEND=noninteractive`):
  `apt-get update && apt-get install -y tmux git curl ca-certificates mkcert`.
  Only packages the probe reported missing are named.
- Node: read the wanted major from pi's `engines.node` in the npm registry
  (`https://registry.npmjs.org/@earendil-works/pi-coding-agent/latest`,
  fallback constant `22`). Add the NodeSource keyring and repo for that
  major, `apt-get install -y nodejs`. Skip when the probe's `NodeMajor`
  already satisfies the range.
- Owner pass: `npm install -g @earendil-works/pi-coding-agent@latest`
  (ADR-0093 argv). Skip when pi is on PATH. Verify `pi --version`.
- Adopted distro guard: when the distro was **not** registered by this exe,
  the stage lists the packages it intends to add and asks before the first
  apt call. The same dialog path as `wsl --shutdown` in ADR-0020.
- Non-Ubuntu/Debian distro (`/etc/os-release` `ID_LIKE` lacks `debian`):
  stage reports missing tools and stops; `doctor` names them.

### 1c. Surfaces

- `picode-desktop doctor` prints both new stages with before/after state.
- Tray tooltip during the stages: "Installing PiCode in Ubuntu…" and
  "Installing tmux, Node and pi…", reusing the existing progress path.
- `docs-site/guide/windows-desktop.md`: clean-machine walkthrough, the one reboot,
  disk/network numbers, the adopted-distro confirmation, and the SmartScreen
  paragraph until 2a lands.
- `CHANGELOG.md` `[Unreleased]` + `web/shared/data/whats-new.json` entry.

### Tests

- Table tests for `NextStage` with the two new stages, including "picode
  present but older", "pi present, tmux missing", "non-Debian distro".
- Fake `Runner` scripts asserting the exact argv of every wsl.exe call, in
  the pattern of `drive_test.go`; UTF-16LE decoding of probe output.
- `install/release_test.go`: `ReleaseFor(tag)` and SHA256SUMS verification,
  including a mismatch that must refuse the file.
- `wslreal_test.go`-style guarded test that runs the probe against a real
  distro when one exists, skipped on hosted runners.
- Owner acceptance: a Windows 11 VM without WSL → download exe → reboot →
  PiCode tab. Record the timeline in the handoff note.

## Phase 2 — signing and winget (CI only)

### 2a. Code signing

- `.github/workflows/release.yml`: after the build step, a
  `windows-latest` job signs `dist/picode-desktop-windows-amd64.exe` with
  `azure/trusted-signing-action` when `AZURE_*` secrets exist; skipped with
  a visible notice otherwise. `SHA256SUMS` is regenerated **after** signing.
- `cmd/picode-desktop/update.go`: the self-update verifies the downloaded
  exe's Authenticode signature (`Get-AuthenticodeSignature` via PowerShell,
  or `wintrust` through `golang.org/x/sys/windows`) when the running exe is
  itself signed; unsigned dev builds skip the check.
- Owner action: create the Azure Trusted Signing account and identity;
  store secrets in the repo. Not scriptable from here.

### 2b. winget

- `scripts/winget-manifest.mjs <version>` writes the three manifest files
  (`version`, `installer`, `defaultLocale`) for `cfpperche.PiCode`,
  installer type `portable`, with the release asset URL and SHA256.
- Release workflow opens the PR against `microsoft/winget-pkgs` with
  `wingetcreate` (token in secrets) once per Stable tag; patch tags too.
- `docs/release-process.md`: two new steps and the manual fallback.

## Phase 3 — imported PiCode distro (after separate approval)

- CI job: `debootstrap --variant=minbase --include=…` (recipe shared with
  `internal/provision/container.go`, extracted to `scripts/rootfs.sh`),
  plus `systemd`, `mkcert`, NodeSource Node, `npm install -g pi`, and the
  release's `picode` at `/usr/local/bin/picode`. Output
  `picode-rootfs-amd64.tar.gz` + checksum on the release.
- Desktop: `--distro picode --import` (and the default when no WSL 2 distro
  exists): download, verify, `wsl --import picode %LOCALAPPDATA%\PiCode\wsl
  <tarball> --version 2`, then create-user → provision as today. The
  `install-*` stages find everything present and no-op.
- ADR-0020 amendment row: "Existing distro: adopted; none: PiCode distro
  imported; `--import` opts in on a machine that has one."

## Decision table

| Conditions | Action | Required evidence |
|---|---|---|
| Distro has no `picode` | Download same-tag Linux asset, verify SHA256, install to `~/.local/bin`, verify version | Runner argv tests; checksum mismatch test |
| `picode` present, older than exe | Replace; unit restart is provision's job | Version-compare table test |
| `picode` present, same or newer | No-op | Table test |
| Unstamped exe, `picode` present | No-op regardless of version | Table test |
| tmux/git/curl/mkcert missing on Debian-like distro registered by the exe | apt install as root, only the missing names | Runner argv tests |
| Same, distro adopted | Show the package list, act only after confirmation | Dialog path test (fake runner + confirm stub) |
| Node absent or below pi's `engines` | NodeSource repo for the wanted major, apt install | Table test on `engines` parsing; argv test |
| pi absent | `npm install -g @earendil-works/pi-coding-agent@latest` as owner | Argv equals ADR-0093 plan |
| pi present | No-op, whatever the version (ADR-0003) | Table test |
| Non-Debian distro | Report missing tools, stop; doctor names them | Table test |
| `/mnt/c` unavailable | Stream the binary through stdin | Runner test with automount off |
| Download fails | Stage fails with URL and reason; rerun resumes | Error path test |
| Release has secrets | Signed exe, SHA256SUMS regenerated after signing | Workflow run on a tag; `Get-AuthenticodeSignature` Valid |
| Release lacks secrets | Unsigned exe, notice in job summary | Workflow run |
| Stable tag | winget PR opened with correct SHA | Dry-run of `winget-manifest.mjs` in `make ci` |
| Self-update, running exe signed | Refuse an unsigned or mismatched download | Update path test with a fake signature check |

## Validation

To be filled when the phases land: `make ci`, `go test -race
./internal/desktop ./cmd/picode-desktop ./internal/install`, the owner's
clean-VM timeline, and a signed-release check.
