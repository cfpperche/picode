/* PiCode M1 — workspace list + terminal grid (vanilla JS; framework
   decision deferred, see docs/decisions/0004-defer-frontend-framework.md) */
"use strict";

const $ = (sel) => document.querySelector(sel);

const state = {
  workspaces: [],
  selectedId: null,
  terms: new Map(), // workspaceId -> { term, fit, ws, paneEl, tabEl }
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

// ---------- system + version ----------
async function loadSystem() {
  try {
    const [sys, ver] = await Promise.all([api("/api/system"), api("/api/version")]);
    $("#ver").textContent = "v" + ver.version;

    const el = $("#warnings");
    if (sys.warnings && sys.warnings.length) {
      el.hidden = false;
      el.innerHTML = sys.warnings.map((w) => `<div>${escapeHTML(w)}</div>`).join("");
    }
    const ok = sys.tmux.installed && sys.pi.installed;
    const status = $("#sys-status");
    status.textContent = ok
      ? `tmux ${sys.tmux.version || "?"} · ${sys.pi.version || "pi"}`
      : "setup needed — see warning above";
    status.classList.toggle("ok", ok);
  } catch {
    $("#sys-status").textContent = "offline — server unreachable";
  }
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
    const li = document.createElement("li");
    li.className = "ws-item" + (ws.id === state.selectedId ? " active" : "");
    li.innerHTML = `
      <div class="ws-row1">
        <span class="ws-dot ${ws.running ? "running" : ""}"></span>
        <span class="ws-name" title="${escapeHTML(ws.name)}">${escapeHTML(ws.name)}</span>
        <span class="ws-actions">
          ${ws.running
            ? `<button class="btn btn-ghost btn-sm btn-stop" title="Stop the agent">Stop</button>`
            : `<button class="btn btn-ghost btn-sm btn-open" title="Start a Pi agent in this workspace">Open agent</button>`}
        </span>
      </div>
      <div class="ws-row2">
        <span class="ws-path" title="${escapeHTML(ws.path)}">${escapeHTML(ws.path)}</span>
        <span class="ws-actions"><button class="btn btn-ghost btn-sm btn-danger btn-remove" title="Remove workspace (files untouched)">Remove</button></span>
      </div>`;

    li.addEventListener("click", (e) => {
      if (e.target.closest("button")) return;
      selectWorkspace(ws.id);
    });
    const openBtn = li.querySelector(".btn-open");
    if (openBtn) openBtn.addEventListener("click", () => openAgent(ws.id));
    const stopBtn = li.querySelector(".btn-stop");
    if (stopBtn) stopBtn.addEventListener("click", () => closeAgent(ws.id));
    li.querySelector(".btn-remove").addEventListener("click", async () => {
      if (!confirm(`Remove workspace "${ws.name}"?\n(The project folder is not deleted.)`)) return;
      try {
        await api(`/api/workspaces/${ws.id}`, { method: "DELETE" });
        closeTerm(ws.id, true);
        await loadWorkspaces();
      } catch (err) { alert(err.message); }
    });

    ul.appendChild(li);
  }
}

function renderEmpty() {
  const has = state.workspaces.length > 0;
  $("#empty").hidden = has;
  $("#term-area").hidden = !has;
}

function selectWorkspace(id) {
  state.selectedId = id;
  renderWorkspaceList();
  const ws = state.workspaces.find((w) => w.id === id);
  if (ws && ws.running) attachTerm(id, ws);
}

async function openAgent(id) {
  try {
    await api(`/api/workspaces/${id}/open`, { method: "POST" });
    await loadWorkspaces();
    attachTerm(id, state.workspaces.find((w) => w.id === id));
  } catch (err) { alert(err.message); }
}

async function closeAgent(id) {
  try {
    await api(`/api/workspaces/${id}/close`, { method: "POST" });
    closeTerm(id, true);
    await loadWorkspaces();
  } catch (err) { alert(err.message); }
}

// ---------- new workspace form ----------
function wireForm() {
  const form = $("#form-new"), btnNew = $("#btn-new"), cancel = $("#btn-cancel");
  const show = (on) => { form.hidden = !on; if (on) $("#inp-name").focus(); };
  btnNew.addEventListener("click", () => show(true));
  $("#btn-new-empty").addEventListener("click", () => { show(true); });
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
        body: JSON.stringify({ name, path: expandPath(path) }),
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

function expandPath(p) {
  if (p === "~") return p;
  if (p.startsWith("~/")) return p; // server resolves; keeps UI simple
  return p;
}

// ---------- terminals ----------
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
  term.writeln("\x1b[90mattaching to Pi agent…\x1b[0m");

  const entry = { term, fit, paneEl, tabEl, sock: null, onWinResize: null, closedByUser: false };

  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${proto}//${location.host}/ws/term?session=picode-${id}`;
  const sock = new WebSocket(wsUrl);
  sock.binaryType = "arraybuffer";

  sock.onopen = () => {
    term.reset();
    const sendResize = () => {
      sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
    };
    fit.fit();
    sendResize();
    entry.onWinResize = () => { if (paneEl.classList.contains("active")) { fit.fit(); sendResize(); } };
    window.addEventListener("resize", entry.onWinResize);
    term.onData((data) => {
      if (sock.readyState === WebSocket.OPEN) sock.send(new TextEncoder().encode(data));
    });
    term.onResize(sendResize);
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
    if (!entry.closedByUser) {
      term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
    }
  };
  entry.sock = sock;
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
wireForm();
loadSystem();
loadWorkspaces()
  .then(() => {
    // Come back to running agents: auto-attach the first one.
    const running = state.workspaces.find((w) => w.running);
    if (running) selectWorkspace(running.id);
  })
  .catch((e) => console.error("boot:", e));
