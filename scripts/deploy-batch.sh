#!/usr/bin/env bash
# make deploy-batch — ship main to the installed service, in batches, only
# when nobody is mid-turn (ADR-0086).
#
# Deploys were one per merged branch (26–63 service restarts a day) and each
# restart ended every managed CLI/agent pane. This script is what the
# systemd timer (scripts/systemd/picode-deploy.timer) and `make deploy-batch`
# run: from the root checkout on main, refresh the public captures if the
# UI changed since the last capture, commit them, and deploy — `picode
# deploy` itself refuses while agents work (exit 2), so a busy fleet just
# means "next slot". Nothing here forces.
set -uo pipefail
cd "$(dirname "$0")/.."

# systemd starts us with a bare PATH.
[ -s "$HOME/.nvm/nvm.sh" ] && . "$HOME/.nvm/nvm.sh" >/dev/null 2>&1
export PATH="$HOME/.local/bin:$HOME/go-sdk/go/bin:$HOME/go/bin:/usr/local/go/bin:$PATH"

log() { printf '%s deploy-batch: %s\n' "$(date +%FT%T)" "$*"; }

if [ "$(git branch --show-current)" != "main" ]; then log "root is not on main; skipping"; exit 0; fi
for st in MERGE_HEAD CHERRY_PICK_HEAD REVERT_HEAD rebase-merge rebase-apply; do
  [ -e ".git/$st" ] && { log "a $st is in progress; skipping"; exit 0; }
done

head=$(git rev-parse --short=7 HEAD)
data=${PICODE_DATA:-$HOME/.picode}
deployed=$(tail -n 1 "$data/var/deploy-log.jsonl" 2>/dev/null | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{process.stdout.write(JSON.parse(s).version||"")}catch{}})')

# Captures follow the UI; the fingerprint gate decides, not the calendar.
if ! DOCS_STRICT=1 node scripts/docs-check.mjs >/dev/null 2>&1; then
  if node scripts/docs-check.mjs --strict 2>&1 | grep -q 'inputs changed'; then
    log "public captures are stale; recapturing"
    if make --no-print-directory docs-shots >/tmp/picode-deploy-batch-shots.log 2>&1; then
      if [ -n "$(git status --porcelain -- docs-site/img)" ]; then
        git add docs-site/img && git commit -q -m "docs: refresh public captures" && log "committed refreshed captures"
        head=$(git rev-parse --short=7 HEAD)
      fi
    else
      log "docs-shots failed (see /tmp/picode-deploy-batch-shots.log); deploying without recapture"
    fi
  fi
fi

case "$deployed" in
  *"$head") log "main ($head) is already deployed ($deployed); nothing to do"; exit 0 ;;
esac

log "deploying main $head (installed: ${deployed:-unknown})"
if make --no-print-directory deploy; then
  log "deployed $head"
elif [ $? -eq 2 ]; then
  log "refused: agents are working; next slot"
  exit 0
else
  log "deploy failed"
  exit 1
fi
