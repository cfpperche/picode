---
description: Point-in-time snapshots of your PiCode environment to a folder you choose, and how to restore one.
---

# Backup and restore

A backup is a point-in-time copy of your PiCode environment written to a
folder you pick — an external drive, a synced folder, anywhere that is not
inside PiCode's own data directory. **Settings → Backup** is the whole
surface.

PiCode takes no backup until you name a folder. There is no default
destination and nothing is copied off this machine.

## Set it up

| Control | What it does |
|---|---|
| **Folder** | Where snapshots are written. **Reveal** opens it in your file manager. |
| **Schedule** | Take a snapshot automatically. Off means **Backup now** only. |
| **Interval** | How often the schedule runs. |
| **Keep** | How many days of snapshots to keep. Older ones are pruned, and the newest is always kept whatever the setting says. |
| **Include sessions** | Add your agent conversations (the `~/.pi` session files). |
| **Include secrets** | Add the credential vault and provider logins. |

If the folder you chose is on the same disk as PiCode, the panel says so:
*"This folder is on the same disk as PiCode. It will not survive a dead
drive."* That is the only warning you get, and it is the one that matters.

**Backup now** runs one snapshot immediately and shows its progress. Each
snapshot appears in the list below with its time and size; the row menu has
**Reveal** and **Remove**.

## What a snapshot holds

- The PiCode database — workspaces, agents, terminals, pins, automations,
  and the rest of what PiCode itself records.
- Pin attachments and sketches.
- Your pi settings and trust file.
- With **Include sessions**: agent conversation files.
- With **Include secrets**: the credential vault and provider logins.

Snapshots after the first one **hard-link** unchanged files to the previous
snapshot where the filesystem allows it, so ten kept days do not cost ten
full copies.

## The vault travels; its key does not

With **Include secrets** on, the snapshot carries `credentials.json` — the
encrypted vault — and deliberately **not** the key that opens it. The key
stays in PiCode's data directory.

This is the point of encrypting credentials in the first place: a snapshot
copied to another machine, a shared drive or a cloud folder cannot be
decrypted there. A restore **on this machine** keeps working, because the
key never moved.

The consequence to plan for: restoring a snapshot onto a fresh install, or
after the data directory was wiped, gives you a vault the new key cannot
open. Your accounts are then not recoverable from that snapshot — sign in
again. If you are migrating machines and want the vault to come with you,
copy the key file out of the data directory yourself, by hand, and treat it
like the secret it is.

## Restore

**Restore** on a snapshot row replaces the live environment with that
snapshot's copy. PiCode asks first, because this is not additive: the
database, pin files and — if the snapshot has them — session files are
replaced, not merged. Your running agents are stopped before it starts, and
**your project folders are not touched** — PiCode never had a copy of them
to restore.

What restore does **not** touch: anything the snapshot does not contain. A
snapshot taken without sessions leaves your current sessions alone.

Each piece is swapped into place only after its replacement is complete, so
a restore that fails partway — a full disk, an unreadable file — leaves what
you had. The credential vault it replaces is kept beside it as
`credentials.json.replaced`, which is what you rename back if the restored
vault turns out to be one this machine's key cannot open.

A snapshot taken by a **newer** PiCode than the one you are running is
refused with the version in the message. Upgrade first.

## What it is not

- **Not off-site.** PiCode writes to a folder. Getting that folder somewhere
  safe is yours to arrange.
- **Not a sync.** A snapshot is a point in time, not a continuously mirrored
  copy.
- **Not a pi install backup.** It copies pi's settings, trust and sessions —
  not pi itself, its packages, or anything under your project folders. Your
  code is in git.
