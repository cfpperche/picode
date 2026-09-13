# 2026-09-13 — feat/provider-thinking-levels: thinking levels in the custom endpoint form

Shipped: the Custom endpoint form declares a reasoning model and picks the
thinking levels it answers on (`minimal`…`max`); `customProviderPayload` sends
`reasoning` + `thinkingLevels` per model and `mergeThinkingLevels`
(`internal/catalog/modelsjson.go`) writes pi's per-model `thinkingLevelMap` —
selected level keeps its name, unselected managed level becomes `null`,
unmanaged keys (a hand-set `off`) survive, a model switched back to
non-reasoning drops the managed keys. The catalog returns the stored map in
`definitions`, so Edit prefills the same chips. `.dlg-create` caps at `80dvh`
and scrolls inside: the taller form had clipped the dialog title off the top
(`overlayAudit` clipTop, top −64 at a 633px viewport).
Verified: `make ci-scoped` PASS after the merge of main; end-to-end on scratch
`thinklevels` (:8473, isolated HOME) — a GUI save wrote
`{"reasoning":true,"thinkingLevelMap":{"high":"high","max":"max","minimal":null,…}}`,
`pi auth check` answered `ready`, `pi --list-models` showed `thinking: yes`,
Edit reopened with `high`+`max` checked, and the empty-selection error was
captured. The provider picker is unaffected (480px: dialog over=0, list scrolls).
visual-review: PASS (7 PNGs in `var/screenshots/`; desktop 633px + 480px,
mobile 390×844; card 5/5; `__picodeOverlayAudit` ok:true in every state)
Not done / debts: see `docs/handoff/open/providers-custom.md` (per-model
fields, `off`, sticky actions).
Merge: fast-forward ready (2 commits + merge of main)

## Next up

- Owner's option 2: prefill from `GET /v1/models` (ids, reasoning, vision,
  per-model context) — the durable item already sits in the topic file.
