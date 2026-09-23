# ADR-0198: The Management window runs a closed list of root commands in the distro through `wsl.exe -u root`

- **Status**: accepted
- **Date**: 2026-09-23
- **Boundary**: security model — the Windows shell starts processes as the distro's root account, which no PiCode surface did before

## Context

The desktop Management window measures and prunes the person's own caches
and edits the global `.wslconfig`, all as the person's Linux account. Two
things the owner asked for need root inside the distro: the distro's own
settings file, `/etc/wsl.conf` (systemd, the default user, whether the
Windows PATH is appended, automount, the generated `resolv.conf`), and the
system caches (`/var/cache/apt`, the systemd journal).

On this machine `sudo` asks for a password. Three ways in were weighed with
the owner: `wsl.exe -u root` (no password: WSL lets the Windows user enter
any account of their own distro), a password typed in the window and handed
to `sudo -S`, or read-only views. The owner chose `wsl.exe -u root`
(2026-09-23).

`wsl.exe -u root` grants nothing new to the Windows account: any terminal it
opens can already run it. What is new is that a PiCode surface does it, so
the surface must not become a way to run *arbitrary* root commands.

## Decision

The shell's Go tool (`picode-desktop`) runs, as `wsl.exe -d <distro> -u root
--exec <argv>`, only commands from a closed list compiled into it, each an
argv (`--exec` hands argv over as argv; the `--` form goes through a shell
that eats `$`, measured 2026-09-23). Three of them are `sh -c` with a fixed
script compiled in, whose only inputs are positional arguments: reading
`/etc/wsl.conf` when it exists; the write (its one argument is the new
content as base64 — data, never syntax — decoded into a temporary file; the
file as it was before PiCode's first save is kept once as
`/etc/wsl.conf.picode-orig`, the previous save as `/etc/wsl.conf.bak`; a
symlinked file is refused; then a rename and a `sync`); and `du -sB1` over
the system-cache table's paths that exist. The rest are plain argv: `getent
passwd <name>` to check a default account exists, and each system cache's own
prune (`apt-get clean`, `journalctl --rotate --vacuum-size=100M`). The window names caches and settings by id and key;
the tool maps them to the table. `wsl.conf` edits are limited to a key table
of Microsoft's documented settings with typed values; `[boot] command` —
which would run a command as root at every boot — is shown and never
written; `[user] default=root` and accounts that do not exist are refused;
mount options and the mount root take a character allowlist. Every write
asks the person first in the window (naming the settings that would stop
PiCode itself working: systemd off, interop off, no generated
`resolv.conf`), and the tool refuses to write without `--yes`.

## Consequences

- The window can fix what caused real problems here (the bare PATH behind
  "go is not installed" came from `appendWindowsPath=false`) without a
  terminal, and prune system caches the person cannot reach otherwise.
- The trust is the Windows account's, exactly as before; the page gains no
  command a terminal on the same account did not already have. A compromised
  page reaches the closed list, never a root shell.
- A new root action is a code change to the list, reviewed like this ADR's
  scope — never a string from the page.
- `wsl.conf` changes apply when the distro restarts; the window offers the
  existing Restart WSL flow (interlock and confirmation included).
- Cost: a distro whose root account is disabled or whose `/etc` is read-only
  gets an error from the tool, shown as such.

## Alternatives considered

- **Password in the window → `sudo -S`**: works where root is locked, but a
  password crosses the webview and the IPC for every action, and the shell
  would have to decide how long to keep it. Lost: more exposure for no
  access `wsl.exe -u root` does not already give.
- **Read-only views**: safest, but leaves the person with the terminal for
  the exact fixes the view points at. Lost on the owner's call.
- **A generic "run as root" command with an allowlist checked at runtime**:
  one string-typed door is one bug away from arbitrary root; the closed list
  of argv builders has no such door.
