const GUIDE = "https://cfpperche.github.io/picode/guide/browser-extension";
const MAX_IMAGE = 700000;

const els = {
  skel: $("skel"),
  empty: $("empty"),
  actionWrap: $("action-wrap"),
  emptyAction: $("empty-action"),
  form: $("form"),
  agent: $("agent"),
  agentNote: $("agent-note"),
  page: $("page"),
  msg: $("msg"),
  shotRow: $("shot-row"),
  shot: $("shot"),
  send: $("send"),
  status: $("status"),
  open: $("open"),
};

let picodeUrl = "";
let tab = { url: "", title: "", selection: "" };
let agents = [];

init();

async function init() {
  const preview = new URLSearchParams(location.search).get("preview");
  if (preview) {
    showPreview(preview);
    return;
  }
  showSkeleton();
  await refreshTab();
  await load();
}

function showPreview(kind) {
  picodeUrl = "https://localhost:8445";
  setOpen(picodeUrl);
  if (kind === "host") {
    showEmpty({ code: "host_missing" });
    return;
  }
  if (kind === "down") {
    showEmpty({ code: "picode_down" });
    return;
  }
  if (kind === "none") {
    showEmpty({ code: "no_agents" });
    return;
  }
  agents = [
    { id: "a1", name: "builder", workspace: "picode", mode: "managed" },
    { id: "a2", name: "reviewer", workspace: "picode", mode: "stopped" },
  ];
  if (kind === "terminal") {
    agents[0].mode = "interactive";
  }
  if (kind === "chrome") {
    tab = { url: "chrome://extensions", title: "Extensions", selection: "" };
  } else {
    tab = { url: "https://linear.app/issue/x", title: "LIN-12 Fix the empty state", selection: "the empty well" };
  }
  showForm();
}

async function load() {
  const ping = await call({ type: "ping" });
  if (!ping.ok) {
    showEmpty(ping);
    return;
  }
  picodeUrl = ping.url || "";
  setOpen(picodeUrl);
  const roster = await call({ type: "agents" });
  if (!roster.ok) {
    showEmpty(roster);
    return;
  }
  picodeUrl = roster.url || picodeUrl;
  setOpen(picodeUrl);
  agents = roster.agents || [];
  if (!agents.length) {
    showEmpty({ code: "no_agents", error: "No agents yet." });
    return;
  }
  showForm();
}

function showSkeleton() {
  els.skel.hidden = false;
  els.empty.hidden = true;
  els.actionWrap.hidden = true;
  els.form.hidden = true;
}

function showEmpty(res) {
  els.skel.hidden = true;
  els.form.hidden = true;
  els.empty.hidden = false;
  els.actionWrap.hidden = false;
  const code = res.code || "";
  if (code === "host_missing") {
    els.empty.textContent = res.error && res.error !== "Install the PiCode host."
      ? res.error
      : "Install the PiCode host.";
    els.emptyAction.textContent = "Retry";
    els.emptyAction.onclick = () => {
      showSkeleton();
      load();
    };
    setOpen("");
    return;
  }
  if (code === "no_agents") {
    els.empty.textContent = "No agents yet.";
    els.emptyAction.textContent = "Open PiCode";
    els.emptyAction.onclick = () => openPiCode();
    return;
  }
  els.empty.textContent = "PiCode is not running.";
  els.emptyAction.textContent = "Retry";
  els.emptyAction.onclick = () => {
    showSkeleton();
    load();
  };
}

function showForm() {
  els.skel.hidden = true;
  els.empty.hidden = true;
  els.actionWrap.hidden = true;
  els.form.hidden = false;
  fillAgents();
  renderPage();
  syncSend();
}

function fillAgents() {
  const prev = els.agent.value;
  els.agent.replaceChildren();
  for (const a of agents) {
    const opt = document.createElement("option");
    opt.value = a.id;
    let label = a.workspace ? a.workspace + " / " + a.name : a.name;
    if (a.mode === "stopped") label += " — stopped";
    if (a.mode === "interactive") label += " — in terminal";
    opt.textContent = label;
    els.agent.appendChild(opt);
  }
  if ([...els.agent.options].some((o) => o.value === prev)) {
    els.agent.value = prev;
  }
}

function currentAgent() {
  return agents.find((a) => a.id === els.agent.value) || agents[0];
}

function renderPage() {
  const ok = canCapture(tab.url);
  els.shotRow.hidden = !ok;
  if (!ok) els.shot.checked = false;
  if (!tab.url) {
    els.page.textContent = "No tab.";
    return;
  }
  if (!ok) {
    els.page.textContent = "This page can't be sent.";
    return;
  }
  els.page.textContent = tab.title ? tab.title + " — " + tab.url : tab.url;
}

function syncSend() {
  const agent = currentAgent();
  const pageOk = canCapture(tab.url);
  const interactive = agent && agent.mode === "interactive";
  els.agentNote.hidden = !interactive;
  if (interactive) {
    els.agentNote.textContent = "This agent is in the terminal.";
    els.send.hidden = true;
    return;
  }
  const canSend = pageOk || !!els.msg.value.trim();
  els.send.hidden = !canSend;
  els.send.disabled = !canSend;
  els.send.textContent = agent && agent.mode === "stopped" ? "Start and send" : "Send";
}

els.agent.addEventListener("change", syncSend);
els.msg.addEventListener("input", syncSend);
els.form.addEventListener("submit", (e) => {
  e.preventDefault();
  send();
});
els.msg.addEventListener("keydown", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
    e.preventDefault();
    send();
  }
});

async function send() {
  const agent = currentAgent();
  if (!agent || els.send.disabled) return;
  const pageOk = canCapture(tab.url);
  els.send.disabled = true;
  els.status.hidden = false;
  els.status.textContent = agent.mode === "stopped" ? "Starting…" : "Sending…";
  const payload = {
    type: "send",
    agentId: agent.id,
    message: els.msg.value.trim(),
  };
  if (pageOk) {
    payload.tab = {
      url: tab.url,
      title: tab.title || "",
      selection: tab.selection || "",
    };
  }
  if (pageOk && els.shot.checked) {
    const image = await screenshot();
    if (image) payload.image = image;
  }
  const res = await call(payload);
  if (!res.ok) {
    els.status.textContent = res.error || "Send failed.";
    els.send.disabled = false;
    return;
  }
  els.status.textContent = "Sent.";
  els.msg.value = "";
  syncSend();
  if (picodeUrl && agent.id) {
    setOpen(picodeUrl + "/#/agent/" + agent.id);
  }
}

async function refreshTab() {
  const [active] = await chrome.tabs.query({ active: true, currentWindow: true });
  tab = { url: active?.url || "", title: active?.title || "", selection: "" };
  if (active?.id && canCapture(tab.url)) {
    try {
      const [inj] = await chrome.scripting.executeScript({
        target: { tabId: active.id },
        func: () => window.getSelection()?.toString() || "",
      });
      tab.selection = (inj && inj.result) || "";
    } catch (_) {
      /* restricted page */
    }
  }
}

async function screenshot() {
  try {
    const dataUrl = await chrome.tabs.captureVisibleTab({ format: "jpeg", quality: 60 });
    const data = (dataUrl || "").split(",")[1] || "";
    if (!data || data.length > MAX_IMAGE) return null;
    return { mimeType: "image/jpeg", data };
  } catch (_) {
    return null;
  }
}

function canCapture(url) {
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch (_) {
    return false;
  }
}

function call(payload) {
  return chrome.runtime.sendMessage({ channel: "picode", payload });
}

function setOpen(url) {
  if (!url) {
    els.open.hidden = true;
    els.open.href = "#";
    return;
  }
  els.open.hidden = false;
  els.open.href = url;
  els.open.onclick = (e) => {
    e.preventDefault();
    chrome.tabs.create({ url });
  };
}

function openPiCode() {
  if (picodeUrl) chrome.tabs.create({ url: picodeUrl });
}

function $(id) {
  return document.getElementById(id);
}
