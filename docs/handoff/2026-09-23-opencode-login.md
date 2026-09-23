# 2026-09-23 — feat/opencode-login: OpenCode signs in through the GUI (the last of the nine)

Shipped: ADR-0201. `internal/server/opencode_login.go` runs `opencode serve` on localhost while the dialog needs it (10 minutes idle, then stopped). Measured on 1.18.32, its API covers:
- `GET /provider`: 223 providers from models.dev, with their env and which are connected;
- `GET /provider/auth`: the plugin methods and their prompts;
- `PUT /auth/{id}`: a key, with metadata;
- `POST /provider/{id}/oauth/authorize|callback`: auto or code.

PiCode serves `GET /api/opencode/catalog`, `POST`/`GET`/`DELETE /api/opencode/credential` and `POST /api/opencode/credential/code`. The catalog rows reach clicreds through `SetOpencodeCatalog`, and `OpencodeVaultID` maps ids (`openai`→`openai-codex`). `OpenCodeLoginDialog.jsx` (browser and mobile) walks provider → method → prompts (select or text, with `when`) → key, or page (auto waiting / pasted code), with the controlled search and scroll pattern.

With this branch **all nine CLIs sign in from the GUI**, each with its own dialog.

Fixed along the way:
- The boot probe uses its own 2s client: a request that lands while `opencode serve` boots is accepted and never answered, which hung the catalog in 5/5 cold starts.
- The dialog bounds the catalog wait at 40s and offers Try again.
- `opencode serve` now stops with PiCode: `cmd/picode/shutdown.go`'s `gracefulShutdown` calls `server.StopSidecars()` synchronously.

Verified: `make ci-scoped` PASS. `TestOpencodeGUICredentials` runs a fake `opencode serve` whose first boot probe hangs, and covers the catalog, a key with metadata, a key filed in the vault, code OAuth (wrong code refused, right one filed under `openai-codex`), 409, and auto OAuth. `TestOMPRosterSaysHowEachSigninRuns` now expects OpenCode's own dialog.

Scratch QA with the real OpenCode and an isolated HOME. The first pass was FAIL: the catalog hung, loading never ended, and serve outlived PiCode. After the fixes, the second pass PASSED steps 1–7: 223 providers; DeepSeek and Cloudflare keys, the latter with `accountId` metadata; Copilot's select and conditional Enterprise URL leading to the device page; OpenAI's three methods and the headless code; mobile. The shutdown rechecks: pid gone 1s after stop. The production OpenCode `auth.json` was untouched throughout.
visual-review: PASS (ocl2/03-picker.png, ocl2/13-copilot-page.png, ocl2/17-m-picker.png, ocl2/18-m-form.png; card 5/5)

Not verified: a real OpenCode OAuth or key end to end. Nits: a missing prompt answer shows under the key field, not under its own field; the picker opens with focus on "Sign in from a terminal", not on the search.

## Next up

- Owner: deploy, then exercise each CLI's GUI sign-in live (every branch of this run carries a live-verification debt)

## Debts

- `cmd/picode/main.go` serves `server.New`'s handler from its own http.Server, so the RegisterOnShutdown hooks in `server.New` (LlamaJobs.Close, CLIJobs.Close, LlamaService.Close) have never run. Making them run changes those subsystems' shutdown, so it is the owner's call; `StopSidecars` is called directly instead.
- OpenCode GUI sign-in not yet exercised live with a real account or key (ADR-0201)

Merge: fast-forward ready.
