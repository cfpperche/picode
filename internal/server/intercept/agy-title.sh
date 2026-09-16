#!/bin/sh
# PiCode Antigravity activity reporter (Fatia 5, docs/plans/launch-muse-agy.md).
# Installed as the `title` command in the CLI's settings.json. The CLI pipes
# agent-state JSON on stdin whenever the state changes; this posts it through
# picode-hook (which maps agent_state and reports to the daemon) and prints
# the short title the CLI renders (ANSI is stripped there anyway).
#
# Read-only by design: hook failures (daemon down, timeout) never touch the
# session — picode-hook already degrades to silent, and the title below is
# best-effort. Never add a hooks.json decision hook here: a PreToolUse
# command gates tool execution, and a bad one breaks the owner's tools.
# The HOOK assignment below is replaced at install with the absolute picode-hook:
# the CLI's environment is the user's, not ours, so PATH is not trusted.
PICODE_HOOK="__PICODE_HOOK_ABS__"
input=$(cat)
printf '%s' "$input" | "$PICODE_HOOK" auto agy 2>/dev/null || true
if command -v python3 >/dev/null 2>&1; then
  printf '%s' "$input" | python3 -c 'import json,os,sys
try:
  d = json.load(sys.stdin)
except Exception:
  print("Antigravity"); sys.exit(0)
state = str(d.get("agent_state") or "idle")
cwd = str(d.get("cwd") or "")
base = os.path.basename(cwd.rstrip("/")) or "Antigravity"
print("Antigravity · " + state + " · " + base)' 2>/dev/null || echo "Antigravity"
else
  echo "Antigravity"
fi
