### Added

- **Development flow: "When the flow bends".** The guide now walks the
  thirteen situations where a step refuses — CI red after the fast-forward, a
  land that cannot fast-forward, a dirty root checkout, a deploy mid-turn, a
  handoff topic the board cannot see, a board over its target, an abandoned
  session (`stalled:`), a clobbered living doc, a note refused on `main`, a
  `make vale` vocabulary hit, a dead docs link, a `worktree-gc` keep, and a
  debt outliving its note. Each card quotes the message the command actually
  prints and names the action that unblocks it: a refusal describes state, it
  is not a wall to route around.

- **Development flow: escape hatches.** A table of the five deliberate
  overrides — `PICODE_ALLOW_SWITCH`, `PICODE_ALLOW_MAIN_REWIND`,
  `picode deploy --force` / `PICODE_DEPLOY_FORCE=1`, `FORCE=1` and
  `--no-verify` — with the guard each one disables and the use it is
  legitimate for. The mutation lock is not on the list.

### Changed

- The development-flow diagram's phase-1 ADR diamond draws its `no` leg now:
  a UI refinement or a route move needs no record, while behavior changes
  still land in `docs/architecture/`.
