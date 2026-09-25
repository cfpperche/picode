### Removed

- **More old links.** `#/app/inbox`, `#/app/matrix`, `#/preferences/status`,
  `#/clis/terminals` and mobile `#/changes/...` no longer redirect; open
  `#/inbox`, `#/app/canvas`, `#/clis` and the Inspector instead. A saved
  `Matrix` tab from before the Canvas rename is dropped on reload.
- **Old scope words in links.** Setup links carry only `?scope=global`,
  `workspace` or `agent`; `?layer=` and the older words (`user`, `project`,
  `machine`, `local`) now read as an unknown scope.
- **Old browser settings.** The terminal theme and font size stored before
  terminal preferences existed, and the Canvas keys from before the rename,
  are no longer read.
- **`picode gateway --insecure-listen`.** Use `--plain`, which it was an
  alias of.
