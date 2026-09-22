# 2026-09-22 — feat/one-cli-door

Docs-only branch, no code changed. ADR-0184 (accepted by the owner, 2026-09-22) closes the second CLI-launch door: every user-facing launch — New agent, a profile, the palette, resuming a session, a handoff — becomes an agent, and `POST /api/clis/{cli}/terminals` is removed. Two terminals stay unbound: shells, and sign-in terminals, which the server creates internally, keeps off the sidebar and Terminals list, and reaps when the credential appears, the process exits, after 15 min idle, and at boot. A catalog CLI seen running in a PiCode shell only offers "Make agent" (a click binds it; CLIs outside PiCode are out of scope). The owner was explicit: no terminal stays open without visibility.

Measured 2026-09-22 on the owner's instance: 11 terminals bound to agents, 1 unbound CLI terminal (`checklist-mirror`, omp).

`docs/plans/one-cli-door.md` splits the work into four branches, each leaving the app working. Nothing implemented yet.

## Next up

- `feat/cli-door-agents` first: every launch (New terminal/profile Use, palette, Sessions resume, handoff) creates an agent.
- Then migrate unbound terminals, close the route, and adopt from a shell, per `docs/plans/one-cli-door.md`.
