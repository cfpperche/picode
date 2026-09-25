# 2026-09-25 — model-pick-action: a model list that cannot be read offers the step that unblocks it

Shipped (2d371f756): `climodels.Blocked{Msg, Action}`; `/api/cli-models`
answers 502 `{"error","action"}`. `signin` (Grok with no models_cache.json and
no auth.json; Antigravity with no token, or agy saying "sign in" → "sign-in has
expired") links to `#/clis/<cli>/providers/new` (the CLI's sign-in dialog);
`open` (Grok signed in, never started) links to `#/clis/new/<cli>` (new-agent
screen); anything else offers "Try again" (`fresh=1`). A damaged Grok cache says
"Grok's saved model list is damaged — open Grok, then try again" instead of
parser text. Shared `modelsStep` in `web/shared/domain/cliModels.js`;
`ModelsNote` in both apps' `CliNativeSettings.jsx` (also Omp's role picker);
`.combo-note-act` styles, 36px tap height on mobile. cli-settings.md updated.

- Pays the `## Debts` item of `2026-09-25-agy-models.md` (the picker's error
  state had one line and no action). Owner asked for it on 2026-09-25.
- Scratch QA, desktop + mobile 390px: all four states; Sign in opened Grok's
  sign-in dialog, Open Grok landed on the new-agent form, Try again loaded the
  list after the file was fixed. Two visual-review rounds, the second PASS;
  overlay audit ok.

## Debts

- Antigravity's expired-sign-in path was not exercised live: it is a string
  match on agy's "sign in" message; unit tests cover only the missing-token path.
