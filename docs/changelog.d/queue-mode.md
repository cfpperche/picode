### Added
- **The integration declaration names its mode (ADR-0186).** `mode` is `local` (PiCode runs the declared commands and the fast-forward), `provider` (the project's own merge queue integrates it; PiCode enqueues and observes), or absent, which runs nothing. It is set in Preferences → Landing work or in a workspace's Settings, beside the rules it applies to, and follows the same workspace → machine fallback.
- Provider mode carries no check commands — its provider runs those — and the declaration refuses the combination instead of keeping commands that would never run.

### Changed
- **A declaration that names no mode no longer executes.** An entry the owner authorizes is ejected with `not run: the project declares no integration mode…`, and the Delivery read and the settings surface both say so; adding `mode` to the declaration restores it.
- Under `provider`, ordering is refused (`the provider owns the queue's order`) rather than being a second opinion beside the provider's queue.
