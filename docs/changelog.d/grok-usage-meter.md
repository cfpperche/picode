### Fixed
- **Dashboard: Grok's tokens, cost, turns, tool calls and timings now show
  up.** The Grok coverage row said "prompt history only" and rendered `—`
  beside a CLI that has been writing a `summary.json`, an `events.jsonl`
  turn/tool timeline and a `usage.json` per-turn count into every session
  directory. Turns, tools and durations come from `events.jsonl`; the model
  from `summary.json`; tokens and cost from `usage.json`, which Grok only
  began writing in 1.0.x. Because older sessions have no `usage.json`,
  tokens and cost report as *partial* with both counts — the coverage row
  says exactly how many turns are priced — instead of a total that quietly
  omits them.
