/* PiCode — workspace list, terminal grid, user menu, settings, theme.
   Vanilla ES modules-free (ADR-0004). Docs live in the repo, not in chrome. */
"use strict";

const $ = (sel) => document.querySelector(sel);

const state = {
  workspaces: [],
  selectedId: null,
  system: null,
  version: "",
  terms: new Map(), // workspaceId -> { term, fit, sock, paneEl, tabEl, onWinResize, closedByUser }
  panel: null,     // { agentId, ws, sock, tools: Map }
};

// ---------- api ----------
async function api(path, opts) {
  const res = await fetch(path, opts);
  if (!res.ok) {
    let msg = res.statusText;
    try { msg = (await res.json()).error || msg; } catch {}
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

// ---------- theme ----------
const Theme = {
  mode: localStorage.getItem("picode-theme") || "system",
  media: matchMedia("(prefers-color-scheme: dark)"),
  resolved() {
    return this.mode === "system"
      ? (this.media.matches ? "dark" : "light")
      : this.mode;
  },
  apply() {
    const r = this.resolved();
    document.documentElement.dataset.theme = r;
    document.documentElement.style.colorScheme = r;
    document.querySelectorAll("[data-theme-option]").forEach((el) => {
      const active = el.dataset.themeOption === this.mode;
      if (el.classList.contains("theme-card")) el.setAttribute("aria-checked", String(active));
      else el.dataset.active = active ? "1" : "";
    });
  },
  set(mode) {
    this.mode = mode;
    localStorage.setItem("picode-theme", mode);
    this.apply();
  },
};
Theme.media.addEventListener("change", () => { if (Theme.mode === "system") Theme.apply(); });

// ---------- router (hash) ----------
function route() {
  const onSettings = location.hash === "#/settings";
  $("#workspace-view").hidden = onSettings;
  $("#settings-view").hidden = !onSettings;
  closeUserMenu();
}

// ---------- system + version ----------
async function loadSystem() {
  try {
    const [sys, ver] = await Promise.all([api("/api/system"), api("/api/version")]);
    state.system = sys;
    state.version = ver.version;
    $("#ver").textContent = "v" + ver.version;
    $("#um-ver").textContent = "v" + ver.version;
    $("#about-ver").textContent = "v" + ver.version;

    const host = sys.host || "local";
    $("#um-name").textContent = host;
    $("#um-sub").textContent = "this machine";
    $("#um-name2").textContent = host;

    const el = $("#warnings");
    if (sys.warnings && sys.warnings.length) {
      el.hidden = false;
      el.innerHTML = sys.warnings.map((w) => `<div>${escapeHTML(w)}</div>`).join("");
    } else {
      el.hidden = true;
    }
    renderSettingsSystem();
  } catch {
    $("#um-sub").textContent = "offline";
  }
}

function renderSettingsSystem() {
  const sys = state.system;
  const dl = $("#settings-sys");
  if (!sys) {
    dl.innerHTML = `<div class="sys-row"><dt>Status</dt><dd>unavailable</dd></div>`;
    return;
  }
  const rows = [
    ["tmux", sys.tmux.installed ? (sys.tmux.version || "installed") : "not installed"],
    ["pi", sys.pi.installed ? (sys.pi.version || "installed") : "not installed"],
    ["extended-keys-format", sys.tmux.extendedKeysFormat || "—"],
  ];
  dl.innerHTML = rows
    .map(([k, v]) => `<div class="sys-row"><dt>${escapeHTML(k)}</dt><dd>${escapeHTML(v)}</dd></div>`)
    .join("");
}

// ---------- user menu ----------
function closeUserMenu() {
  const pop = $("#um-popover");
  if (pop.hidden) return;
  pop.hidden = true;
  $("#um-trigger").setAttribute("aria-expanded", "false");
}

function wireUserMenu() {
  const trigger = $("#um-trigger");
  const pop = $("#um-popover");

  trigger.addEventListener("click", (e) => {
    e.stopPropagation();
    const open = pop.hidden;
    pop.hidden = !open;
    trigger.setAttribute("aria-expanded", String(open));
    if (open) pop.querySelector("button").focus();
  });
  pop.addEventListener("click", (e) => e.stopPropagation());
  document.addEventListener("click", closeUserMenu);
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") closeUserMenu(); });

  $("#um-settings").addEventListener("click", () => {
    location.hash = "#/settings";
    closeUserMenu();
  });
}

// theme buttons (user menu segmented + settings cards) — shared by data attribute
document.addEventListener("click", (e) => {
  const btn = e.target.closest("[data-theme-option]");
  if (!btn) return;
  Theme.set(btn.dataset.themeOption);
});

// ---------- workspaces ----------
async function loadWorkspaces() {
  state.workspaces = await api("/api/workspaces");
  renderWorkspaceList();
  renderEmpty();
}

function renderWorkspaceList() {
  const ul = $("#ws-list");
  ul.innerHTML = "";
  for (const ws of state.workspaces) {
    const mode = ws.agent ? ws.agent.mode : "stopped";
    const running = mode !== "stopped";
    const li = document.createElement("li");
    li.className = "ws-item" + (ws.id === state.selectedId ? " active" : "");
    let actions;
    if (mode === "stopped") {
      actions = `<button class="btn btn-ghost btn-sm btn-managed" title="Run with the task panel">Run</button>` +
                `<button class="btn btn-ghost btn-sm btn-open" title="Open in the terminal">Terminal</button>`;
    } else {
      actions = `<button class="btn btn-ghost btn-sm btn-stop" title="Stop the agent">Stop</button>`;
      if (mode === "interactive") {
        actions = `<button class="btn btn-ghost btn-sm btn-open" title="Open in the terminal">Terminal</button>` + actions;
      }
    }
    li.innerHTML = `
      <div class="ws-row1">
        <span class="ws-dot ${running ? "running" : ""}"></span>
        <span class="ws-name" title="${escapeHTML(ws.name)}">${escapeHTML(ws.name)}</span>
        <span class="ws-actions">${actions}</span>
      </div>
      <div class="ws-row2">
        <span class="ws-path" title="${escapeHTML(ws.path)}">${escapeHTML(ws.path)}</span>
        <span class="ws-actions"><button class="btn btn-ghost btn-sm btn-danger btn-remove" title="Remove workspace (files untouched)">Remove</button></span>
      </div>`;

    li.addEventListener("click", (e) => {
      if (e.target.closest("button")) return;
      selectWorkspace(ws.id);
    });
    const managedBtn = li.querySelector(".btn-managed");
    if (managedBtn) managedBtn.addEventListener("click", () => startManaged(ws.id));
    const openBtn = li.querySelector(".btn-open");
    if (openBtn) openBtn.addEventListener("click", () => openAgent(ws.id));
    const stopBtn = li.querySelector(".btn-stop");
    if (stopBtn) stopBtn.addEventListener("click", () => stopAgent(ws.id));
    li.querySelector(".btn-remove").addEventListener("click", async () => {
      if (!confirm(`Remove workspace "${ws.name}"?\n(The project folder is not deleted.)`)) return;
      try {
        await api(`/api/workspaces/${ws.id}`, { method: "DELETE" });
        closeTerm(ws.agent ? ws.agent.id : ws.id, true);
        if (state.panel && ws.agent && state.panel.agentId === ws.agent.id) closePanel();
        await loadWorkspaces();
      } catch (err) { alert(err.message); }
    });

    ul.appendChild(li);
  }
}

function renderEmpty() {
  const has = state.workspaces.length > 0;
  $("#empty").hidden = has;
  if (!has) {
    $("#term-area").hidden = true;
    closePanel();
  }
}

function selectWorkspace(id) {
  state.selectedId = id;
  renderWorkspaceList();
  const ws = state.workspaces.find((w) => w.id === id);
  if (!ws || !ws.agent) return;
  if (ws.agent.mode === "managed") {
    attachTermClose();
    showAgentPanel(ws);
  } else if (ws.agent.mode === "interactive") {
    closePanel();
    attachTerm(ws.agent.id, ws);
  }
}

function attachTermClose() {
  for (const id of [...state.terms.keys()]) closeTerm(id, true);
  $("#term-area").hidden = true;
}

async function startManaged(id) {
  const ws = state.workspaces.find((w) => w.id === id);
  if (!ws || !ws.agent) return;
  try {
    await api(`/api/agents/${ws.agent.id}/managed/start`, { method: "POST" });
    state.selectedId = id;
    await loadWorkspaces();
    showAgentPanel(state.workspaces.find((w) => w.id === id));
  } catch (err) { alert(err.message); }
}

async function stopAgent(id) {
  const ws = state.workspaces.find((w) => w.id === id);
  if (!ws || !ws.agent) return;
  try {
    if (ws.agent.mode === "managed") {
      await api(`/api/agents/${ws.agent.id}/managed/stop`, { method: "POST" });
      if (state.panel && state.panel.agentId === ws.agent.id) closePanel();
    } else {
      await api(`/api/workspaces/${ws.id}/close`, { method: "POST" });
      closeTerm(ws.agent.id, true);
    }
    await loadWorkspaces();
  } catch (err) { alert(err.message); }
}

// ---------- new workspace form ----------
function wireForm() {
  const form = $("#form-new"), btnNew = $("#btn-new"), cancel = $("#btn-cancel");
  const show = (on) => { form.hidden = !on; if (on) $("#inp-name").focus(); };
  btnNew.addEventListener("click", () => show(true));
  $("#btn-new-empty").addEventListener("click", () => show(true));
  cancel.addEventListener("click", () => show(false));

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const errEl = $("#form-error");
    errEl.hidden = true;
    const name = $("#inp-name").value.trim();
    const path = $("#inp-path").value.trim();
    if (!name || !path) {
      errEl.textContent = "Name and folder path are required.";
      errEl.hidden = false;
      return;
    }
    try {
      const ws = await api("/api/workspaces", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, path }),
      });
      form.reset();
      show(false);
      state.selectedId = ws.id;
      await loadWorkspaces();
    } catch (err) {
      errEl.textContent = err.message;
      errEl.hidden = false;
    }
  });
}

async function openAgent(id) {
  try {
    await api(`/api/workspaces/${id}/open`, { method: "POST" });
    state.selectedId = id;
    await loadWorkspaces();
    const ws = state.workspaces.find((w) => w.id === id);
    if (ws && ws.agent && ws.agent.mode === "interactive") {
      closePanel();
      attachTerm(ws.agent.id, ws);
    }
  } catch (err) { alert(err.message); }
}

// ---------- agent panel (managed mode, M2) ----------
function setAgentStreaming(on) {
  $("#agent-dot").classList.toggle("streaming", on);
  $("#agent-state-text").textContent = on ? "streaming" : "idle";
}

function showAgentPanel(ws) {
  closePanel();
  const agent = ws.agent;
  $("#agent-area").hidden = false;
  $("#term-area").hidden = true;
  $("#empty").hidden = true;
  $("#agent-title").textContent = `${ws.name} · ${agent.name}`;
  const stream = $("#agent-stream");
  stream.innerHTML = "";

  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const sock = new WebSocket(`${proto}//${location.host}/ws/agent?agent=${agent.id}`);
  const panel = { agentId: agent.id, ws, sock, tools: new Map(), nearBottom: true };
  state.panel = panel;

  stream.addEventListener("scroll", () => {
    panel.nearBottom = stream.scrollHeight - stream.scrollTop - stream.clientHeight < 40;
  });

  sock.onmessage = (ev) => {
    try { renderAgentEvent(JSON.parse(ev.data), panel); } catch {}
  };
  sock.onclose = () => {
    if (state.panel === panel) {
      setAgentStreaming(false);
      appendStreamLine("— panel disconnected —", "as-thinking");
    }
  };
  setAgentStreaming(false);
}

function closePanel() {
  if (!state.panel) return;
  try { state.panel.sock.close(); } catch {}
  state.panel = null;
  $("#agent-area").hidden = true;
  if (state.workspaces.length > 0) $("#term-area").hidden = state.terms.size === 0;
}

function streamEl() { return $("#agent-stream"); }

function appendStreamNode(node) {
  const s = streamEl();
  const stick = state.panel && state.panel.nearBottom;
  s.appendChild(node);
  if (stick) s.scrollTop = s.scrollHeight;
}

function appendStreamLine(text, cls) {
  const div = document.createElement("div");
  if (cls) div.className = cls;
  div.textContent = text;
  appendStreamNode(div);
}

function renderAgentEvent(env, panel) {
  const ev = env.event || {};
  switch (ev.type) {
    case "snapshot":
      setAgentStreaming(!!ev.streaming);
      break;
    case "agent_start":
      setAgentStreaming(true);
      break;
    case "agent_settled":
      setAgentStreaming(false);
      break;
    case "message_update": {
      const d = ev.assistantMessageEvent;
      if (!d) break;
      if (d.type === "text_delta") {
        const t = document.createTextNode(d.delta || "");
        const last = streamEl().lastChild;
        if (last && last.nodeType === 3) last.appendData(d.delta || "");
        else appendStreamNode(t);
      } else if (d.type === "thinking_delta") {
        const last = streamEl().lastChild;
        const txt = d.delta || "";
        if (last && last.nodeType === 1 && last.classList.contains("as-thinking")) {
          last.textContent += txt;
        } else {
          const div = document.createElement("div");
          div.className = "as-thinking";
          div.textContent = txt;
          appendStreamNode(div);
        }
      }
      break;
    }
    case "tool_execution_start":
      addToolRow(panel, ev);
      break;
    case "tool_execution_end":
      finishToolRow(panel, ev);
      break;
    case "enqueue_accepted": {
      $("#task-input").value = "";
      appendStreamLine(`» queued (${ev.kind})`, "as-thinking");
      break;
    }
    case "task_delivered":
      appendStreamLine(`✓ delivered (${ev.kind})`, "as-thinking");
      break;
    case "task_failed":
      appendStreamLine(`✗ failed: ${ev.error}`, "tr-err as-thinking");
      break;
    case "enqueue_rejected":
      alert(ev.error);
      break;
  }
}

function addToolRow(panel, ev) {
  const row = document.createElement("div");
  row.className = "tool-row";
  const args = JSON.stringify(ev.args || {});
  const head = document.createElement("div");
  head.className = "tr-head";
  head.style.cssText = "display:flex;align-items:center;gap:8px;flex:1;min-width:0;";
  const name = document.createElement("span");
  name.className = "tr-name"; name.textContent = ev.toolName || "tool";
  const argsEl = document.createElement("span");
  argsEl.className = "tr-args"; argsEl.textContent = args.length > 2 ? args : "";
  const st = document.createElement("span");
  st.className = "tr-status"; st.textContent = "…";
  head.append(name, argsEl, st);
  const detail = document.createElement("div");
  detail.className = "tr-detail"; detail.textContent = args;
  row.append(head, detail);
  row.addEventListener("click", () => row.classList.toggle("expanded"));
  appendStreamNode(row);
  panel.tools.set(ev.toolCallId, { row, st, detail });
}

function finishToolRow(panel, ev) {
  const t = panel.tools.get(ev.toolCallId);
  if (!t) return;
  t.st.textContent = ev.isError ? "error" : "ok";
  t.row.classList.add(ev.isError ? "tr-err" : "tr-ok");
  const result = JSON.stringify(ev.result || {}, null, 2);
  t.detail.textContent = result;
  panel.tools.delete(ev.toolCallId);
}

function wireComposer() {
  const send = () => {
    const p = state.panel;
    const input = $("#task-input");
    const payload = input.value.trim();
    if (!p || !payload || p.sock.readyState !== WebSocket.OPEN) return;
    p.sock.send(JSON.stringify({
      type: "enqueue",
      kind: $("#task-kind").value,
      payload,
    }));
    input.value = "";
  };
  $("#task-send").addEventListener("click", send);
  $("#task-input").addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  });
}
function setTermState(connected) {
  $("#sb-dot").classList.toggle("connected", connected);
  $("#sb-state-text").textContent = connected ? "connected" : "detached";
}

function attachTerm(id, ws) {
  if (state.terms.has(id)) { activateTerm(id); return; }

  const tabs = $("#tabs"), terms = $("#terms");

  const tabEl = document.createElement("div");
  tabEl.className = "tab active";
  tabEl.innerHTML = `<span class="tab-dot"></span><span>${escapeHTML(ws.name)}</span><button class="tab-close" title="Detach (agent keeps running)">×</button>`;
  tabEl.addEventListener("click", () => activateTerm(id));
  tabEl.querySelector(".tab-close").addEventListener("click", () => closeTerm(id));
  tabs.appendChild(tabEl);

  const paneEl = document.createElement("div");
  paneEl.className = "term-pane active";
  terms.appendChild(paneEl);

  const term = new Terminal({
    cursorBlink: true,
    fontSize: 12.5,
    fontFamily: 'ui-monospace, "SF Mono", "Cascadia Code", Menlo, monospace',
    theme: {
      background: "#0d0f12",
      foreground: "#e7eaf0",
      cursor: "#7aa2f7",
      selectionBackground: "#33467c",
    },
    scrollback: 10000,
  });
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(paneEl);

  const entry = { term, fit, paneEl, tabEl, sock: null, onWinResize: null, closedByUser: false };

  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${proto}//${location.host}/ws/term?session=picode-${id}`;
  const sock = new WebSocket(wsUrl);
  sock.binaryType = "arraybuffer";
  entry.sock = sock;

  sock.onopen = () => {
    term.reset();
    setTermState(true);
    const sendResize = () => {
      sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
      $("#sb-dims").textContent = `${term.cols}×${term.rows}`;
    };
    fit.fit();
    sendResize();
    entry.onWinResize = () => { if (paneEl.classList.contains("active")) { fit.fit(); sendResize(); } };
    window.addEventListener("resize", entry.onWinResize);
    term.onData((data) => {
      if (sock.readyState === WebSocket.OPEN) sock.send(new TextEncoder().encode(data));
    });
    term.onResize(() => {
      if (sock.readyState === WebSocket.OPEN) {
        sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
      }
      $("#sb-dims").textContent = `${term.cols}×${term.rows}`;
    });
    term.focus();
  };

  sock.onmessage = (ev) => {
    if (typeof ev.data === "string") {
      try {
        const msg = JSON.parse(ev.data);
        if (msg.type === "error") term.writeln(`\r\n\x1b[31m${msg.message}\x1b[0m`);
      } catch {}
      return;
    }
    term.write(new Uint8Array(ev.data));
  };

  sock.onclose = () => {
    if (entry.onWinResize) window.removeEventListener("resize", entry.onWinResize);
    setTermState(false);
    if (!entry.closedByUser) {
      term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
    }
  };

  state.terms.set(id, entry);
  activateTerm(id);
  $("#sb-session").textContent = `picode-${id} · ${ws.path}`;
}

function activateTerm(id) {
  for (const [wid, t] of state.terms) {
    t.paneEl.classList.toggle("active", wid === id);
    t.tabEl.classList.toggle("active", wid === id);
  }
  const t = state.terms.get(id);
  if (t) { t.term.focus(); }
}

function closeTerm(id, silent) {
  const t = state.terms.get(id);
  if (!t) return;
  t.closedByUser = true;
  try { t.sock.close(); } catch {}
  t.term.dispose();
  t.tabEl.remove();
  t.paneEl.remove();
  state.terms.delete(id);
  if (!silent) {
    const first = state.terms.keys().next();
    if (!first.done) activateTerm(first.value);
  }
}

// ---------- utils ----------
function escapeHTML(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  })[c]);
}

// ---------- boot ----------
Theme.apply();
wireForm();
wireUserMenu();
wireComposer();
window.addEventListener("hashchange", route);
route();
loadSystem();
loadWorkspaces()
  .then(() => {
    // Come back to running agents: attach panel (managed) or terminal.
    const active = state.workspaces.find((w) => w.agent && w.agent.mode !== "stopped");
    if (active) selectWorkspace(active.id);
  })
  .catch((e) => console.error("boot:", e));
