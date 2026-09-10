#!/usr/bin/env bash
# scripts/qa-scratch.sh — an isolated PiCode instance of THIS worktree for
# browser QA (ADR-0086). Production is other agents' working instance; UI
# work is verified here.
#
#   qa-scratch.sh start <name> [port]   build (UI embedded) + run detached
#   qa-scratch.sh seed  <name>          one workspace, agent and terminal via the API
#   qa-scratch.sh stop  <name>          remove its terminals, then stop the daemon
#   qa-scratch.sh url   <name>
#
# Layout: var/qa/<name>/{home,data,picode,server.log}. HOME is isolated so
# pi reads a copy of your credentials and nothing you do here touches
# ~/.picode. HTTP on localhost (PICODE_INSECURE=1): a secure context without
# a certificate dance. Detached with setsid so a tool timeout never kills it.
set -euo pipefail
cd "$(dirname "$0")/.."

cmd=${1:-}; name=${2:-}
[ -n "$cmd" ] && [ -n "$name" ] || { sed -n '2,14p' "$0"; exit 1; }
dir="var/qa/$name"
portfile="$dir/port"

url() { echo "http://localhost:$(cat "$portfile")"; }

case "$cmd" in
  start)
    port=${3:-}
    if [ -z "$port" ]; then
      port=8470
      while fuser -s "$port/tcp" 2>/dev/null; do port=$((port + 1)); done
    fi
    mkdir -p "$dir/home/.pi/agent" "$dir/data"
    for f in auth.json models-store.json; do
      [ -f "$HOME/.pi/agent/$f" ] && cp "$HOME/.pi/agent/$f" "$dir/home/.pi/agent/$f"
    done
    make --no-print-directory web >/dev/null
    go build -tags embedui -o "$dir/picode" ./cmd/picode
    fuser -k "$port/tcp" 2>/dev/null || true
    echo "$port" > "$portfile"
    HOME="$PWD/$dir/home" PICODE_DATA="$PWD/$dir/data" PICODE_PORT="$port" PICODE_INSECURE=1 \
      setsid nohup "$PWD/$dir/picode" > "$dir/server.log" 2>&1 < /dev/null &
    for _ in $(seq 40); do
      curl -sf "http://localhost:$port/api/health" >/dev/null 2>&1 && break
      sleep 0.5
    done
    if ! curl -sf "http://localhost:$port/api/health" >/dev/null 2>&1; then
      echo "qa-scratch: daemon did not answer on :$port — see $dir/server.log" >&2
      exit 1
    fi
    echo "qa-scratch: $name is up at $(url)  (log: $dir/server.log)"
    echo "  agent_browser open $(url)/desktop/    # mobile: $(url)/mobile/?mobile=1#/"
    ;;
  seed)
    base=$(url)
    ws=$(curl -sf -X POST "$base/api/workspaces" -H 'content-type: application/json' \
      -d "{\"name\":\"QA\",\"path\":\"$PWD\"}")
    wsid=$(printf '%s' "$ws" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{const j=JSON.parse(s);process.stdout.write(j.id||(j.workspace&&j.workspace.id)||"")})')
    [ -n "$wsid" ] || { echo "qa-scratch: could not create a workspace: $ws" >&2; exit 1; }
    curl -sf -X POST "$base/api/workspaces/$wsid/agents" -H 'content-type: application/json' -d '{"name":"Atlas"}' >/dev/null
    curl -sf -X POST "$base/api/terminals" -H 'content-type: application/json' -d "{\"workspaceId\":\"$wsid\",\"name\":\"shell\"}" >/dev/null || true
    echo "qa-scratch: seeded workspace $wsid (agent Atlas, terminal shell) at $base"
    ;;
  stop)
    port=$(cat "$portfile" 2>/dev/null || true)
    # tmux sessions outlive the daemon by design (ADR-0018 KillMode=process),
    # so the instance has to forget its terminals while it can still name
    # them: exact ids from its own API. Never `tmux ls | grep picode-` — that
    # sweep killed six production sessions on 2026-09-06.
    if [ -n "$port" ] && curl -sf -m 3 "http://localhost:$port/api/health" >/dev/null 2>&1; then
      ids=$(curl -sf -m 5 "http://localhost:$port/api/terminals" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{for(const t of (JSON.parse(s).terminals||[]))if(t.id)console.log(t.id)}catch{}})')
      gone=0
      for id in $ids; do
        curl -sf -m 5 -X DELETE "http://localhost:$port/api/terminals/$id" -o /dev/null && gone=$((gone + 1))
      done
      [ "$gone" -gt 0 ] && echo "qa-scratch: removed $gone terminal(s) and their tmux sessions"
    elif [ -n "$port" ]; then
      echo "qa-scratch: the daemon on :$port is not answering; its terminals stay as orphan tmux sessions." >&2
      echo "  Start it again and stop it through this script to have them removed." >&2
    fi
    [ -n "$port" ] && fuser -k "$port/tcp" 2>/dev/null || true
    echo "qa-scratch: $name stopped"
    ;;
  url) url ;;
  *) sed -n '2,14p' "$0"; exit 1 ;;
esac
