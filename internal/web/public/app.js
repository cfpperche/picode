/* PiCode — Cursor-class anatomy: conversation hero, terminal dock.
   Vanilla ES (ADR-0004). Visual spec: docs/design/benchmark-visual-anatomy.md */
"use strict";

const $ = (sel) => document.querySelector(sel);

const state = {
  workspaces: [],
  selectedId: null,
  system: null,
  version: "",
  terms: new Map(),   // agentId -> { term, fit, sock, paneEl, tabEl, onWinResize, closedByUser }
  panel: null,        // { agentId, ws, sock, tools: Map, nearBottom, assistantEl }
  dockOpen: false,
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
    return this.mode === "system" ? (this.media.matches ? "dark" : "light") : this.mode;
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
document.addEventListener("click", (e) => {
  const btn = e.target.closest("[data-theme-option]");
  if (btn) Theme.set(btn.dataset.themeOption);
});

// ---------- router ----------
function route() {
  const onSettings = location.hash === "#/settings";
  $("#workspace-view").hidden = onSettings;
  $("#settings-view").hidden = !onSettings;
  closeUserMenu();
}

// ---------- system ----------
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
    $("#um-name2").textContent = host;
    const el = $("#warnings");
    if (sys.warnings && sys.warnings.length) {
      el.hidden = false;
      el.innerHTML = sys.warnings.map((w) => `<div>${escapeHTML(w)}</div>`).join("");
    } else el.hidden = true;
    renderSettingsSystem();
  } catch {
    $("#um-sub").textContent = "offline";
  }
}

function renderSettingsSystem() {
  const sys = state.system;
  const dl = $("#settings-sys");
  if (!sys) { dl.innerHTML = `<div class="sys-row"><dt>Status</dt><dd>unavailable</dd></div>`; return; }
  const rows = [
    ["tmux", sys.tmux.installed ? (sys.tmux.version || "installed") : "not installed"],
    ["pi", sys.pi.installed ? (sys.pi.version || "installed") : "not installed"],
  ];
  dl.innerHTML = rows.map(([k, v]) =>
    `<div class="sys-row"><dt>${escapeHTML(k)}</dt><dd>${escapeHTML(v)}</dd></div>`).join("");
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
  });
  pop.addEventListener("click", (e) => e.stopPropagation());
  document.addEventListener("click", closeUserMenu);
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") closeUserMenu(); });
  $("#um-settings").addEventListener("click", () => { location.hash = "#/settings"; closeUserMenu(); });
}

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
    const li = document.createElement("li");
    li.className = "ws-item" + (ws.id === state.selectedId ? " active" : "");
    let actions = "";
    if (mode === "stopped") {
      actions = `<button class="btn btn-ghost btn-sm btn-managed" title="Run with the task panel">Run</button>`;
    } else {
      actions = `<button class="btn btn-ghost btn-sm btn-stop" title="Stop the agent">Stop</button>`;
    }
    li.innerHTML = `
      <div class="ws-row1">
        <span class="ws-dot ${mode !== "stopped" ? "running" : ""}"></span>
        <span class="ws-name" title="${escapeHTML(ws.name)}">${escapeHTML(ws.name)}</span>
        <span class="ws-actions">${actions}</span>
      </div>
      <div class="ws-row2">
        <span class="ws-path" title="${escapeHTML(ws.path)}">${escapeHTML(ws.path)}</span>
        <span class="ws-mode">${mode}</span>
      </div>`;
    li.addEventListener("click", (e) => {
      if (e.target.closest("button")) return;
      selectWorkspace(ws.id);
    });
    const managedBtn = li.querySelector(".btn-managed");
    if (managedBtn) managedBtn.addEventListener("click", () => startManaged(ws.id));
    const stopBtn = li.querySelector(".btn-stop");
    if (stopBtn) stopBtn.addEventListener("click", () => stopAgent(ws.id));
    li.querySelector(".ws-actions").insertAdjacentHTML("beforeend",
      `<button class="btn btn-ghost btn-sm btn-danger btn-remove" title="Remove workspace (files untouched)">×</button>`);
    li.querySelector(".btn-remove").addEventListener("click", async (e) => {
      e.stopPropagation();
      if (!confirm(`Remove workspace "${ws.name}"?\n(The project folder is not deleted.)`)) return;
      try {
        await api(`/api/workspaces/${ws.id}`, { method: "DELETE" });
        if (ws.agent) closeTerm(ws.agent.id, true);
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
    $("#chat-surface").hidden = true;
    hideDock();
  }
}

function selectWorkspace(id) {
  state.selectedId = id;
  renderWorkspaceList();
  const ws = state.workspaces.find((w) => w.id === id);
  if (!ws || !ws.agent) return;
  openChatSurface(ws);
  if (ws.agent.mode === "interactive") attachTermAndShowDock(ws.agent.id, ws);
}

async function startManaged(id) {
  const ws = state.workspaces.find((w) => w.id === id);
  if (!ws || !ws.agent) return;
  try {
    await api(`/api/agents/${ws.agent.id}/managed/start`, { method: "POST" });
    state.selectedId = id;
    await loadWorkspaces();
    openChatSurface(state.workspaces.find((w) => w.id === id));
  } catch (err) { alert(err.message); }
}

async function stopAgent(id) {
  const ws = state.workspaces.find((w) => w.id === id);
  if (!ws || !ws.agent) return;
  try {
    if (ws.agent.mode === "managed") {
      await api(`/api/agents/${ws.agent.id}/managed/stop`, { method: "POST" });
    } else {
      await api(`/api/workspaces/${ws.id}/close`, { method: "POST" });
      closeTerm(ws.agent.id, true);
    }
    if (state.panel && state.panel.agentId === ws.agent.id) state.panel.stopped = true;
    await loadWorkspaces();
    openChatSurface(state.workspaces.find((w) => w.id === id)); // refresh header state
  } catch (err) { alert(err.message); }
}

// ---------- chat surface ----------
function setChatStatus(text, streaming) {
  $("#chat-status-text").textContent = text;
  $("#chat-dot").classList.toggle("streaming", !!streaming);
}

function openChatSurface(ws) {
  const agent = ws.agent;
  const fresh = !state.panel || state.panel.agentId !== agent.id;
  if (fresh) closePanel();
  $("#empty").hidden = true;
  $("#chat-surface").hidden = false;
  $("#chat-agent-name").textContent = ws.name;
  $("#chat-mode-chip").textContent = agent.mode;
  $("#btn-stop-agent").hidden = agent.mode === "stopped";
  $("#btn-dock").hidden = false;

  if (agent.mode === "managed") {
    if (fresh) connectPanel(ws);
  } else {
    closePanel();
    if (agent.mode === "interactive") {
      setChatStatus("interactive — open the terminal dock", false);
      if (fresh) addSysLine("Agent is running interactively. Use the Terminal dock to pair with it.");
    } else {
      setChatStatus("stopped", false);
      if (fresh) addSysLine("Agent stopped. Press Run to start it in managed mode.");
    }
  }
}

function convCol() {
  const conv = $("#conversation");
  let col = conv.querySelector(".conv-col");
  if (!col) {
    col = document.createElement("div");
    col.className = "conv-col";
    conv.appendChild(col);
  }
  return col;
}

function appendBlock(cls, actor) {
  const b = document.createElement("div");
  b.className = "block " + cls;
  const a = document.createElement("div");
  a.className = "actor";
  a.textContent = actor;
  const c = document.createElement("div");
  c.className = "block-content";
  b.append(a, c);
  convCol().appendChild(b);
  scrollConv();
  return c;
}

function addSysLine(text, err) {
  const div = document.createElement("div");
  div.className = "sys-line" + (err ? " err" : "");
  div.textContent = text;
  convCol().appendChild(div);
  scrollConv();
}

function scrollConv() {
  const conv = $("#conversation");
  const p = state.panel;
  if (!p || p.nearBottom) conv.scrollTop = conv.scrollHeight;
}

function connectPanel(ws) {
  const agent = ws.agent;
  $("#conversation").innerHTML = "";
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const sock = new WebSocket(`${proto}//${location.host}/ws/agent?agent=${agent.id}`);
  const panel = { agentId: agent.id, ws, sock, tools: new Map(), nearBottom: true, stopped: false };
  state.panel = panel;

  $("#conversation").addEventListener("scroll", () => {
    const conv = $("#conversation");
    panel.nearBottom = conv.scrollHeight - conv.scrollTop - conv.clientHeight < 48;
  }, { passive: true });

  sock.onmessage = (ev) => {
    try { renderAgentEvent(JSON.parse(ev.data), panel); } catch {}
  };
  sock.onclose = () => {
    if (state.panel === panel && !panel.stopped) {
      setChatStatus("disconnected", false);
      addSysLine("— panel disconnected —", true);
    }
  };
  setChatStatus("idle", false);
}

function closePanel() {
  if (!state.panel) return;
  const p = state.panel;
  p.stopped = true;
  try { p.sock.close(); } catch {}
  state.panel = null;
}

function renderAgentEvent(env, panel) {
  const ev = env.event || {};
  switch (ev.type) {
    case "snapshot":
      setChatStatus(ev.streaming ? "streaming" : "idle", ev.streaming);
      break;
    case "agent_start":
      setChatStatus("streaming", true);
      panel.assistantEl = null;
      break;
    case "agent_settled":
      setChatStatus("idle", false);
      panel.assistantEl = null;
      panel.thinkingEl = null;
      break;
    case "message_update": {
      const d = ev.assistantMessageEvent;
      if (!d) break;
      if (d.type === "text_delta") {
        if (!panel.assistantEl) panel.assistantEl = appendBlock("", "agent");
        panel.assistantEl.appendData ? null : null;
        panel.assistantEl.textContent += d.delta || "";
        scrollConv();
      } else if (d.type === "thinking_delta") {
        if (!panel.thinkingEl) panel.thinkingEl = appendBlock("thinking", "thinking");
        panel.thinkingEl.textContent += d.delta || "";
        scrollConv();
      }
      break;
    }
    case "tool_execution_start":
      addToolPill(panel, ev);
      break;
    case "tool_execution_end":
      finishToolPill(panel, ev);
      break;
    case "enqueue_accepted": {
      const kind = ev.kind || "prompt";
      const content = appendBlock("user", "You");
      const chip = document.createElement("span");
      chip.className = "chip";
      chip.textContent = kind;
      content.parentElement.querySelector(".actor").appendChild(chip);
      content.textContent = panel.pendingPayload || "";
      panel.pendingPayload = null;
      $("#task-input").value = "";
      autoGrow();
      break;
    }
    case "task_delivered":
      addSysLine(`✓ delivered (${ev.kind})`);
      break;
    case "task_failed":
      addSysLine(`✗ failed: ${ev.error}`, true);
      break;
    case "enqueue_rejected":
      alert(ev.error);
      break;
  }
}

function addToolPill(panel, ev) {
  const pill = document.createElement("div");
  pill.className = "tool-pill";
  const head = document.createElement("div");
  head.className = "tool-pill-head";
  const chev = document.createElement("span");
  chev.className = "tp-chevron";
  chev.textContent = "›";
  const name = document.createElement("span");
  name.className = "tp-name";
  name.textContent = ev.toolName || "tool";
  const argsEl = document.createElement("span");
  argsEl.className = "tp-args";
  const argText = summarizeArgs(ev.args);
  argsEl.textContent = argText;
  const st = document.createElement("span");
  st.className = "tp-status";
  st.textContent = "···";
  head.append(chev, name, argsEl, st);
  const detail = document.createElement("div");
  detail.className = "tp-detail";
  detail.textContent = JSON.stringify(ev.args || {}, null, 2);
  pill.append(head, detail);
  head.addEventListener("click", () => pill.classList.toggle("expanded"));
  convCol().appendChild(pill);
  scrollConv();
  panel.tools.set(ev.toolCallId, { pill, st, detail });
}

function summarizeArgs(args) {
  if (!args) return "";
  if (typeof args.command === "string") return args.command;
  if (typeof args.path === "string") return args.path;
  const s = JSON.stringify(args);
  return s.length > 2 ? s : "";
}

function finishToolPill(panel, ev) {
  const t = panel.tools.get(ev.toolCallId);
  if (!t) return;
  t.st.textContent = ev.isError ? "error" : "ok";
  t.pill.classList.add(ev.isError ? "err" : "ok");
  t.detail.textContent = JSON.stringify(ev.result || {}, null, 2);
  panel.tools.delete(ev.toolCallId);
}

// ---------- composer ----------
function autoGrow() {
  const ta = $("#task-input");
  ta.style.height = "auto";
  ta.style.height = Math.min(ta.scrollHeight, 160) + "px";
}

function wireComposer() {
  const ta = $("#task-input");
  const send = () => {
    const p = state.panel;
    const payload = ta.value.trim();
    if (!p || !payload || p.sock.readyState !== WebSocket.OPEN) return;
    p.pendingPayload = payload;
    p.sock.send(JSON.stringify({ type: "enqueue", kind: $("#task-kind").value, payload }));
  };
  $("#task-send").addEventListener("click", send);
  ta.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); }
  });
  ta.addEventListener("input", autoGrow);
  $("#btn-stop-agent").addEventListener("click", () => state.selectedId && stopAgent(state.selectedId));
  $("#btn-dock").addEventListener("click", () => toggleDock());
  $("#dock-close").addEventListener("click", () => hideDock());
}

// ---------- terminal dock ----------
function showDock() {
  state.dockOpen = true;
  $("#dock").hidden = false;
  const first = state.terms.entries().next();
  if (!first.done) activateTerm(first.value[0]);
}
function hideDock() {
  state.dockOpen = false;
  $("#dock").hidden = true;
}
function toggleDock() {
  if (state.dockOpen) hideDock();
  else showDock();
}

function setTermState(connected) {
  $("#sb-dot").classList.toggle("connected", connected);
  $("#sb-state-text").textContent = connected ? "connected" : "detached";
}

function attachTermAndShowDock(agentId, ws) {
  attachTerm(agentId, ws);
  showDock();
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
    fontSize: 12,
    fontFamily: 'ui-monospace, "SF Mono", "Cascadia Code", Menlo, monospace',
    theme: {
      background: "#0e0e11",
      foreground: "#ececf1",
      cursor: "#7c8cf8",
      selectionBackground: "#33467c",
    },
    scrollback: 10000,
  });
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(paneEl);

  const entry = { term, fit, paneEl, tabEl, sock: null, onWinResize: null, closedByUser: false };

  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const sock = new WebSocket(`${proto}//${location.host}/ws/term?session=picode-${id}`);
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
    entry.onWinResize = () => { if (paneEl.classList.contains("active") && state.dockOpen) { fit.fit(); sendResize(); } };
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
    if (!entry.closedByUser) term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
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
  if (t) { t.term.focus(); t.fit && requestAnimationFrame(() => t.fit.fit()); }
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
    else hideDock();
  }
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
    const active = state.workspaces.find((w) => w.agent && w.agent.mode !== "stopped");
    if (active) selectWorkspace(active.id);
  })
  .catch((e) => console.error("boot:", e));
