#!/usr/bin/env bash
# Fills an inbox with one item of every shape, for visual QA of the app
# (ADR-0037). Usage: scripts/inbox-smoke.sh [base-url] [agent-id]
# The agent id, when given, makes the question/approval items replyable
# for real (the reply lands as a follow_up task on that agent).
set -euo pipefail

BASE="${1:-http://127.0.0.1:8612}"
AGENT="${2:-}"

post() { curl -sk -X POST "$BASE/api/inbox" -H 'content-type: application/json' -d "$1" -o /dev/null -w "%{http_code} "; }
id_of() { curl -sk -X POST "$BASE/api/inbox" -H 'content-type: application/json' -d "$1" | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])'; }

echo "filing into $BASE (agent: ${AGENT:-none})"

post '{"kind":"fyi","sourceKind":"system","reason":"deploy watch","title":"Nightly deploy finished","body":"Everything green. Nothing for you to do."}'
post '{"kind":"fyi","sourceKind":"terminal","sourceId":"term-smoke","reason":"long task finished","title":"Terminal build finished","body":"`make release` exited 0 in 4m12s."}'
post "{\"kind\":\"result\",\"sourceKind\":\"agent\",\"sourceId\":\"${AGENT:-helper-x}\",\"reason\":\"run finished unobserved\",\"title\":\"Auth refactor finished\",\"body\":\"Refactored the auth module while you were away.\n\n- 12 files touched\n- **all tests green**\n- one TODO left in \`session.go\`\n\n\`\`\`go\nfunc Login(ctx context.Context) error { ... }\n\`\`\`\"}"
post "{\"kind\":\"question\",\"sourceKind\":\"agent\",\"sourceId\":\"${AGENT:-helper-x}\",\"reason\":\"agent needs your input\",\"title\":\"Move config to TOML or keep JSON?\",\"body\":\"The repo uses JSON today. TOML would allow comments, at the cost of one more dependency and a migration for anyone on an older build.\"}"
post "{\"kind\":\"approval\",\"sourceKind\":\"agent\",\"sourceId\":\"${AGENT:-helper-x}\",\"reason\":\"destructive action\",\"title\":\"May I drop the legacy messages table?\",\"body\":\"Confirmed dead — zero reads anywhere in the tree. **Irreversible** on the local database.\n\n\`\`\`sql\nDROP TABLE messages;\n\`\`\`\"}"
# A deliberately long title, to prove the ellipsis and time-column contracts.
post "{\"kind\":\"question\",\"sourceKind\":\"agent\",\"sourceId\":\"${AGENT:-helper-x}\",\"reason\":\"agent needs your input\",\"title\":\"Should the migration keep the legacy column names for one more release, or rename them now and ship a compatibility shim for the older desktop builds?\",\"body\":\"Renaming now is cleaner; keeping them is safer for anyone who has not updated.\"}"
post '{"kind":"question","sourceKind":"agent","sourceId":"ghost-agent","reason":"agent needs your input","title":"Question from an agent that was removed","body":"Answering this must fail visibly and leave the item open."}'
post '{"kind":"approval","sourceKind":"system","reason":"maintenance window","title":"Approve the 03:00 maintenance window?","body":"Backup plus a SQLite vacuum.","allowedResponses":["accept","ignore"]}'
echo

# One snoozed item (hidden until due) and one already answered (done).
SNOOZE_ID=$(id_of '{"kind":"fyi","sourceKind":"system","reason":"smoke test","title":"Snoozed note — back in 2 minutes","body":"Hidden from the list until it is due."}')
UNTIL=$(date -u -d "+2 minutes" +%Y-%m-%dT%H:%M:%SZ)
curl -sk -X POST "$BASE/api/inbox/$SNOOZE_ID/state" -H 'content-type: application/json' -d "{\"snoozedUntil\":\"$UNTIL\"}" -o /dev/null -w "snoozed:%{http_code} "

DONE_ID=$(id_of '{"kind":"question","sourceKind":"system","reason":"agent needs your input","title":"Which port should the smoke server use?","body":"8080 or 8612?"}')
curl -sk -X POST "$BASE/api/inbox/$DONE_ID/respond" -H 'content-type: application/json' -d '{"verb":"respond","text":"8612 — it is the scratch port."}' -o /dev/null -w "answered:%{http_code}\n"

curl -sk "$BASE/api/inbox" | python3 -c '
import sys, json
items = json.load(sys.stdin)["items"]
print(str(len(items)) + " visible:")
for i in items:
    print("  [%-8s] blocking=%-5s %-6s %s" % (i["kind"], i["blocking"], i["state"], i["title"][:48]))'
