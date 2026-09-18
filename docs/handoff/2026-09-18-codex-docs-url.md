# 2026-09-18 — codex-docs-url

Full documentation-reference audit for all nine catalog CLIs (owner request
after the Grok mis-attribution). Every docs URL, npm package and version
channel verified live on 2026-09-18.

## Result
- All 9 docs URLs live and the right product; 5/5 npm packages exist; both
  vendor channels (muse-stable, antigravity manifests) serve versions.
- One drift: developers.openai.com/codex/cli now redirects to
  learn.chatgpt.com/docs/codex/cli (OpenAI moved the Codex docs). Catalog
  updated to the canonical target.
- Process note: the first landing attempt of this branch merged a conflicted
  state and swept another session's unlanded commits in (git add -A on a
  conflicted merge — the same trap another session hit on main today).
  Branch rebuilt from main with only the intended change.
