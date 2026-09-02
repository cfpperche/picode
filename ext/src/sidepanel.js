const PA = window.PiCodeAct;
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
  actRow: $("act-row"),
  act: $("act"),
  actBox: $("act-box"),
  actLine: $("act-line"),
  actList: $("act-list"),
  actStop: $("act-stop"),
  actAllow: $("act-allow"),
  actDeny: $("act-deny"),
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
  if (kind === "grant") {
    showActLine("Let PiCode act on https://linear.app?");
    actButtons(false, true, true);
  } else if (kind === "acting") {
    showActLine("Acting on https://linear.app — round 1 of 3");
    actButtons(true, false, false);
    els.actList.replaceChildren(
      actItem({ act: "fill", selector: "#title" }, "ok", "ok"),
      actItem({ act: "click", selector: "button[type=submit]" }, "wait", "…"),
      actItem({ act: "read", selector: "main" }, "err", "no match"),
    );
  } else if (kind === "actdone") {
    showActLine("Done — 2 of 3 steps worked.");
  }
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
  els.actRow.hidden = !ok;
  if (!ok) {
    els.shot.checked = false;
    els.act.checked = false;
  }
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
  const wantsAct = pageOk && els.act.checked;
  els.send.disabled = true;
  els.status.hidden = false;
  els.status.textContent = agent.mode === "stopped" ? "Starting…" : "Sending…";
  const payload = {
    type: "send",
    agentId: agent.id,
    message: els.msg.value.trim(),
    act: wantsAct,
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
  els.status.hidden = true;
  els.msg.value = "";
  syncSend();
  if (picodeUrl && agent.id) {
    setOpen(picodeUrl + "/#/agent/" + agent.id);
  }
  if (wantsAct && res.watching) {
    runActLoop(agent.id);
  } else if (wantsAct && res.actError) {
    showActLine(res.actError + " The reply still arrives in PiCode.");
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
  payload.deviceId = "ext:" + chrome.runtime.id;
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

// ---- Track C: actuation loop (ADR-0053) --------------------------------

let actLoop = { running: false, stop: false };
let actGen = 0;          // a new send supersedes the in-flight loop
let grantCancel = null;  // resolves a pending grant ask (declined)
const ACT_POLL_MS = 2000;
const ACT_STEP_MS = 450;
const ACT_WINDOW_MS = 10 * 60 * 1000;

function stopBatch(batch) {
  return call({ type: "act-result", id: batch.id, stopped: true }).catch(() => {});
}

function showActLine(text) {
  els.actBox.hidden = false;
  els.actLine.hidden = false;
  els.actLine.textContent = text;
  els.actList.replaceChildren();
  actButtons(false, false, false);
}

function actButtons(stop, allow, deny) {
  els.actBox.hidden = false;
  els.actStop.hidden = !stop;
  els.actAllow.hidden = !allow;
  els.actDeny.hidden = !deny;
}

async function runActLoop(agentId) {
  actGen += 1;
  const gen = actGen;
  if (grantCancel) { const c = grantCancel; grantCancel = null; c(); }
  actLoop = { running: true, stop: false };
  const deadline = Date.now() + ACT_WINDOW_MS;
  try {
    while (Date.now() < deadline && !actLoop.stop) {
      if (gen !== actGen) return; // a newer send took over
      const res = await call({ type: "act-next", agentId, tab: { url: PA.originOf(tab.url) } });
      if (gen !== actGen) return;
      if (!res.ok) {
        showActLine(res.error || "PiCode is not running.");
        return;
      }
      if (res.batch) {
        let again = false;
        try {
          again = await handleBatch(res.batch, agentId, gen);
        } catch (e) {
          // Never leave the agent hanging on a claimed batch.
          await stopBatch(res.batch);
          if (gen === actGen) showActLine("Act loop failed: " + ((e && e.message) || e) + ". The agent was told to stop.");
          return;
        }
        if (!again) return;
        continue;
      }
      if (!res.watching) {
        showActLine("Done. The agent didn't ask to act.");
        return;
      }
      showActLine("Watching for the agent's next step…");
      await sleep(ACT_POLL_MS);
    }
    if (!actLoop.stop && gen === actGen) showActLine("Paused. The wait ran out — send again to continue.");
  } finally {
    if (gen === actGen) actLoop.running = false;
  }
}

// handleBatch runs one granted batch; true = keep polling for round N+1.
async function handleBatch(batch, agentId, gen) {
  const origin = batch.origin;
  const grants = await PA.loadGrants();
  if (!grants[origin]) {
    showActLine("Let PiCode act on " + origin + "?");
    const granted = await askGrant(origin);
    if (gen !== actGen) { await stopBatch(batch); return false; }
    if (!granted) {
      await stopBatch(batch);
      showActLine("Skipped. Nothing was touched.");
      return false;
    }
  }
  if (gen !== actGen) { await stopBatch(batch); return false; }
  const [active] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (PA.originOf(active?.url || "") !== origin) {
    showActLine("Open the " + origin + " tab to continue. The steps are waiting.");
    return false;
  }
  showActLine("Acting on " + origin + " — round " + batch.round + " of " + batch.rounds);
  actButtons(true, false, false);
  els.actList.replaceChildren(...batch.actions.map((a) => actItem(a, "wait", "…")));
  const outcomes = await executeBatch(active.id, batch.actions);
  if (actLoop.stop) {
    await call({ type: "act-result", id: batch.id, outcomes, stopped: true });
    showActLine("Stopped. The page is as the steps left it.");
    return false;
  }
  paintOutcomes(outcomes);
  if (gen !== actGen) return false;
  const res = await call({ type: "act-result", id: batch.id, outcomes });
  const oks = outcomes.filter((o) => o.ok).length;
  if (batch.round >= batch.rounds) {
    showActLine("Done — " + oks + " of " + outcomes.length + " steps worked. That was the last round.");
    return false;
  }
  if (!res.ok || !res.watching) {
    showActLine("Done — " + oks + " of " + outcomes.length + " steps worked.");
    return false;
  }
  showActLine("Round " + (batch.round + 1) + " — waiting for the agent…");
  return true;
}

function askGrant(origin) {
  return new Promise((resolve) => {
    actButtons(false, true, true);
    const done = (v) => { grantCancel = null; resolve(v); };
    els.actAllow.onclick = async () => {
      await PA.grantOrigin(origin);
      done(true);
    };
    els.actDeny.onclick = () => done(false);
    grantCancel = () => done(false);
  });
}

async function executeBatch(tabId, actions) {
  const outcomes = [];
  try {
    await chrome.scripting.executeScript({
      target: { tabId },
      func: PA.actDriver,
      args: [{ init: actions }],
    });
    for (let i = 0; i < actions.length; i++) {
      if (actLoop.stop) break;
      await sleep(ACT_STEP_MS);
      const [inj] = await chrome.scripting.executeScript({
        target: { tabId },
        func: PA.actDriver,
        args: [{ step: true }],
      });
      const r = inj && inj.result;
      if (!r || r.done) break;
      outcomes.push(r.outcome);
      paintOne(i, r.outcome);
    }
  } catch (e) {
    for (let i = outcomes.length; i < actions.length; i++) {
      outcomes.push({ act: actions[i].act, selector: actions[i].selector || "", ok: false, error: "page changed" });
      paintOne(i, outcomes[i]);
    }
  }
  return outcomes;
}

function actItem(a, cls, st) {
  const li = document.createElement("li");
  li.className = "act-item " + cls;
  const name = document.createElement("span");
  name.textContent = a.act + (a.selector ? " " + a.selector : a.to ? " to " + a.to : "");
  const s = document.createElement("span");
  s.className = "st";
  s.textContent = st;
  li.append(name, s);
  return li;
}

function paintOne(i, o) {
  const li = els.actList.children[i];
  if (!li) return;
  li.className = "act-item " + (o.ok ? "ok" : "err");
  li.querySelector(".st").textContent = o.ok ? "ok" : (o.error || "failed");
}

function paintOutcomes(outcomes) {
  outcomes.forEach((o, i) => paintOne(i, o));
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

els.actStop.addEventListener("click", () => {
  actLoop.stop = true;
});
