#!/usr/bin/env bash
# scripts/qa-scratch.sh — an isolated PiCode instance of THIS worktree for
# browser QA (ADR-0086). Production is other agents' working instance; UI
# work is verified here.
#
#   qa-scratch.sh start <name> [port]   build (UI embedded) + run detached
#                                       QA_LOGINS=1: also copy every CLI's login
#                                       and PiCode's vault (removed on stop)
#   qa-scratch.sh seed  <name>          one workspace, agent and terminal via the API
#   qa-scratch.sh stop  <name>          remove its terminals, then stop the daemon
#   qa-scratch.sh status <name>         its port, its daemon's pid, whether it answers
#   qa-scratch.sh url   <name>
#
# Layout: var/qa/<name>/{home,data,picode,port,pid,server.log}. HOME is isolated so
# pi reads a copy of your credentials and nothing you do here touches
# ~/.picode. HTTP on localhost (PICODE_INSECURE=1): a secure context without
# a certificate dance. Detached with setsid so a tool timeout never kills it.
set -euo pipefail
cd "$(dirname "$0")/.."

cmd=${1:-}; name=${2:-}
[ -n "$cmd" ] && [ -n "$name" ] || { sed -n '2,14p' "$0"; exit 1; }
dir="var/qa/$name"
portfile="$dir/port"
pidfile="$dir/pid"

url() { echo "http://localhost:$(cat "$portfile")"; }

# daemon_field reads one field out of the daemon's own record in its data
# directory (settings.go writes url/port/pid as it binds). It is the
# authoritative answer to "which port, which pid" — the port *file* is a copy
# that a scan can leave stale, and `stop` keys decisions on these two.
daemon_field() {
  [ -f "$dir/data/server.json" ] || return 1
  node -e 'const fs=require("fs");try{const j=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));const v=j[process.argv[2]];if(v!==undefined&&v!==null&&v!=="")process.stdout.write(String(v))}catch{}' \
    "$dir/data/server.json" "$1" 2>/dev/null
}

# port_pid names the process listening on a port. It is how `start` learns the
# pid to remember and how `stop` refuses to touch a port that is no longer
# ours — the old stop ran `fuser -k <port>/tcp`, so a stale port file made it
# kill a neighbouring scratch's daemon and leave its own alive (2026-09-15).
port_pid() { fuser "$1/tcp" 2>/dev/null | tr -d ' ' | awk '{print $1}'; }

# QA_LOGINS=1 (owner, 2026-09-25): live CLI checks on a scratch use the
# person's own logins. Each CLI's login files and PiCode's vault are copied
# into the scratch HOME/data — only the files that hold the login, never the
# rest of the CLI's state — and every copy is listed in $dir/logins.list so
# `stop` removes exactly those. The worktree is marked trusted in the copied
# Claude Code and Codex configs only, so a start run is not stopped by the
# trust question; the originals are never written.
logins_list() { echo "$dir/logins.list"; }

copy_login() { # copy_login <source> <dest relative to the scratch>
  local src=$1 dst="$dir/$2"
  [ -f "$src" ] || return 0
  mkdir -p "$(dirname "$dst")"
  case "$src" in
    *.db) python3 -c 'import sqlite3,sys; s=sqlite3.connect(sys.argv[1]); d=sqlite3.connect(sys.argv[2]); s.backup(d); d.close(); s.close()' "$src" "$dst" ;;
    *) cp "$src" "$dst" ;;
  esac
  chmod 600 "$dst"
  echo "$2" >> "$(logins_list)"
}

copy_logins() {
  : > "$(logins_list)"
  copy_login "$HOME/.claude/.credentials.json" home/.claude/.credentials.json
  copy_login "$HOME/.claude.json"              home/.claude.json
  copy_login "$HOME/.codex/auth.json"          home/.codex/auth.json
  copy_login "$HOME/.codex/config.toml"        home/.codex/config.toml
  copy_login "$HOME/.grok/auth.json"           home/.grok/auth.json
  copy_login "$HOME/.grok/config.toml"         home/.grok/config.toml
  copy_login "$HOME/.grok/trusted_folders.toml" home/.grok/trusted_folders.toml
  copy_login "$HOME/.hermes/auth.json"         home/.hermes/auth.json
  copy_login "$HOME/.hermes/.env"              home/.hermes/.env
  copy_login "$HOME/.hermes/config.yaml"       home/.hermes/config.yaml
  copy_login "$HOME/.local/share/opencode/auth.json" home/.local/share/opencode/auth.json
  for f in "$HOME"/.config/opencode/opencode.json "$HOME"/.config/opencode/opencode.jsonc; do
    copy_login "$f" "home/.config/opencode/$(basename "$f")"
  done
  copy_login "$HOME/.omp/agent/agent.db"       home/.omp/agent/agent.db
  copy_login "$HOME/.omp/agent/config.yml"     home/.omp/agent/config.yml
  copy_login "$HOME/.config/muse/auth.json"    home/.config/muse/auth.json
  copy_login "$HOME/.gemini/oauth_creds.json"  home/.gemini/oauth_creds.json
  copy_login "$HOME/.gemini/google_accounts.json" home/.gemini/google_accounts.json
  # PiCode's own vault: Grok's and Omp's chosen keys travel through it.
  copy_login "$HOME/.picode/credentials.json"  data/credentials.json
  copy_login "$HOME/.picode/credentials.key"   data/credentials.key
  # Trust this worktree in the copies (the seed workspace is the worktree).
  local ws=$PWD
  if [ -f "$dir/home/.claude.json" ]; then
    python3 - "$dir/home/.claude.json" "$ws" <<'PY'
import json, sys
p, ws = sys.argv[1], sys.argv[2]
d = json.load(open(p))
proj = d.setdefault("projects", {}).setdefault(ws, {})
proj["hasTrustDialogAccepted"] = True
proj["hasCompletedProjectOnboarding"] = True
json.dump(d, open(p, "w"))
PY
  fi
  if [ -f "$dir/home/.codex/config.toml" ] && ! grep -qF "[projects.\"$ws\"]" "$dir/home/.codex/config.toml"; then
    printf '\n[projects."%s"]\ntrust_level = "trusted"\n' "$ws" >> "$dir/home/.codex/config.toml"
  fi
  echo "qa-scratch: copied $(wc -l < "$(logins_list)") login file(s) into $dir (removed on stop)"
}

remove_logins() {
  local list; list=$(logins_list)
  [ -f "$list" ] || return 0
  local n=0 f
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    case "$f" in /*|*..*) continue ;; esac # only paths inside the scratch
    rm -f "$dir/$f" "$dir/$f-wal" "$dir/$f-shm" && n=$((n + 1))
  done < "$list"
  rm -f "$list"
  echo "qa-scratch: removed $n copied login file(s)"
}

health() { curl -sf -m 3 "http://localhost:$1/api/health" >/dev/null 2>&1; }

# owns_port says whether the process answering on a port is *this* scratch's
# daemon: it wrote server.json in this data dir, for this port, and its pid is
# the listener. A healthy answer alone proves nothing — on 2026-09-22 another
# session's scratch took the port while this one was building, this daemon
# failed to bind, the health check was answered by the other instance, and
# `seed` then created a workspace, an agent and a terminal inside it.
owns_port() {
  local want_port=$1 pid port owner
  pid=$(daemon_field pid || true)
  port=$(daemon_field port || true)
  owner=$(port_pid "$want_port" || true)
  [ -n "$pid" ] && [ "$port" = "$want_port" ] && [ -n "$owner" ] && [ "$owner" = "$pid" ]
}

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
    [ "${QA_LOGINS:-}" = 1 ] && copy_logins
    make --no-print-directory web >/dev/null
    go build -tags embedui -o "$dir/picode" ./cmd/picode
    # Checked after the build, not before it: the build takes long enough for
    # another scratch to take a port that was free when the scan ran.
    if held=$(port_pid "$port") && [ -n "$held" ]; then
      echo "qa-scratch: :$port is held by pid $held — refusing to kill it; start with another port." >&2
      exit 1
    fi
    # A server.json left by an earlier run would name an old pid and port, and
    # the ownership check below would read it as this launch's.
    rm -f "$dir/data/server.json"
    echo "$port" > "$portfile"
    # The daemon must not inherit the identity of whatever started it: run from
    # inside a PiCode terminal — which is how an agent runs QA — PICODE_AGENT_ID
    # came through to every scratch terminal, so a terminal here resolved a
    # principal that belongs to another instance's data dir (measured
    # 2026-09-21), and an inherited TMUX put its sessions in that server.
    # Nor Claude Code's: a Claude Code launched in a scratch started from
    # inside Claude Code inherited its child-session marker and wrote no
    # transcript (memory: scratch-inherits-claude-child-session).
    setsid nohup env -u PICODE_AGENT_ID -u PICODE_TERM_ID -u PICODE_TERM_URL -u PICODE_INSTANCE -u TMUX -u TMUX_PANE \
      -u CLAUDE_CODE_CHILD_SESSION -u CLAUDECODE \
      HOME="$PWD/$dir/home" PICODE_DATA="$PWD/$dir/data" PICODE_PORT="$port" PICODE_INSECURE=1 \
      "$PWD/$dir/picode" > "$dir/server.log" 2>&1 < /dev/null &
    launched=$!
    # Ready means this daemon answers on this port — not that something does.
    for _ in $(seq 40); do
      owns_port "$port" && health "$port" && break
      kill -0 "$launched" 2>/dev/null || break
      sleep 0.5
    done
    if ! owns_port "$port" || ! health "$port"; then
      owner=$(port_pid "$port" || true)
      if [ -n "$owner" ] && ! owns_port "$port"; then
        echo "qa-scratch: :$port is answered by pid $owner, which is not this scratch's daemon — it did not bind. Start $name with another port (see $dir/server.log)." >&2
      else
        echo "qa-scratch: daemon did not answer on :$port — see $dir/server.log" >&2
      fi
      kill "$launched" 2>/dev/null || true
      rm -f "$portfile" "$pidfile"
      exit 1
    fi
    # The daemon's own pid, from the server.json it wrote as it bound — the
    # listener was checked against it above, so `stop` ends exactly this
    # process and never a neighbour.
    pid=$(daemon_field pid)
    echo "$pid" > "$pidfile"
    echo "qa-scratch: $name is up at $(url)  (pid ${pid:-unknown}; log: $dir/server.log)"
    echo "  agent_browser open $(url)/desktop/    # mobile: $(url)/mobile/?mobile=1#/"
    ;;
  seed)
    port=$(cat "$portfile" 2>/dev/null || true)
    if [ -z "$port" ] || ! owns_port "$port"; then
      echo "qa-scratch: :${port:-?} is not answered by $name's own daemon — refusing to seed another instance. Start $name first." >&2
      exit 1
    fi
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
    # Which port, which pid: the daemon's own server.json first (it wrote it
    # as it bound), the remembered pid next, the port file last — a scan can
    # leave that one stale, and the old `fuser -k <port>/tcp` then killed a
    # neighbouring scratch's daemon while leaving its own alive (2026-09-15).
    port=$(daemon_field port || true)
    [ -n "$port" ] || port=$(cat "$portfile" 2>/dev/null || true)
    pid=$(daemon_field pid || true)
    [ -n "$pid" ] || pid=$(cat "$pidfile" 2>/dev/null || true)
    # And if a *different* process holds that port now, this stop owns nothing
    # there: leave it — terminals, API and all — and say so.
    if [ -n "$port" ] && [ -n "$pid" ]; then
      owner=$(port_pid "$port" || true)
      if [ -n "$owner" ] && [ "$owner" != "$pid" ]; then
        echo "qa-scratch: :$port is held by pid $owner, not this scratch's daemon ($pid) — leaving that instance alone." >&2
        port=""
      fi
    fi
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
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      for _ in $(seq 20); do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.25
      done
      if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null || true
        sleep 0.25
      fi
    fi
    rm -f "$pidfile"
    if [ -n "$port" ] && health "$port"; then
      echo "qa-scratch: :$port still answers after stopping pid ${pid:-?} — not killing by port; inspect it." >&2
      exit 1
    fi
    remove_logins
    echo "qa-scratch: $name stopped${port:+ (: $port free)}"
    ;;
  status)
    port=$(daemon_field port || true)
    [ -n "$port" ] || port=$(cat "$portfile" 2>/dev/null || true)
    pid=$(daemon_field pid || true)
    [ -n "$pid" ] || pid=$(cat "$pidfile" 2>/dev/null || true)
    alive=no
    [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null && alive=yes
    code=000
    [ -n "$port" ] && code=$(curl -sf -m 2 -o /dev/null -w '%{http_code}' "http://localhost:$port/api/health" 2>/dev/null || echo 000)
    echo "qa-scratch: $name port=${port:-?} pid=${pid:-?} alive=$alive health=$code"
    ;;
  url) url ;;
  *) sed -n '2,14p' "$0"; exit 1 ;;
esac
