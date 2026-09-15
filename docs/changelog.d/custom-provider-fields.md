### Added

- Custom provider form: per-model **display name**, **input** (`text`,
  `image`) and **cost** (USD per 1M tokens, all four rates or none), a
  **streaming usage** compat flag, and a provider string per thinking level
  (`xhigh` → `high`), so a gateway like `meta-ai` is fully describable in
  the GUI — no hand-edited `models.json`. Clearing a row field removes the
  stored key; Edit prefills everything the file holds. The API key flow is
  unchanged: a literal credential into `auth.json`, exactly as
  cheaperinference.
