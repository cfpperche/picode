#!/usr/bin/env node
// docs-video-stills — capture REAL UI stills from the seeded fixture daemon
// for the docs tutorial videos (docs-videos/). Same parity contract as
// docs-shots: every image the videos show is the current working tree's UI,
// captured live, with a content marker before each screenshot.
//
// Drives agent-browser (CLI on PATH) through the tutorial steps — clicks,
// form fills, hash navigation — and writes named PNGs to
// docs-videos/assets/stills/. Run while the fixture is up (make fixture).

import { execFileSync } from "node:child_process";
import { mkdirSync, statSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const outDir = join(root, "docs-videos", "assets", "stills");
const base = process.env.FIXTURE_BASE || "http://127.0.0.1:18740";
mkdirSync(outDir, { recursive: true });

// Fresh agent-browser session per step: matches the docs-shots lesson —
// session state (navigation dedup, SPA caches) must not leak between captures.
let session = `video-stills-${process.pid}`;
const ab = (args, timeout = 60) =>
  execFileSync("agent-browser", [...args, "--session", session], {
    encoding: "utf8",
    timeout: timeout * 1000,
  });

const sleep = (ms) => { const end = Date.now() + ms; while (Date.now() < end); };

const evalJs = (js) => String(ab(["eval", js]));

const waitText = (needle, tries = 24) => {
  // innerText reflects CSS text-transform (mobile headers render UPPER), so
  // match case-insensitively
  const js = `document.body.innerText.toLowerCase().includes(${JSON.stringify(needle.toLowerCase())}) ? 'MARKER_OK' : 'MARKER_NO'`;
  for (let i = 0; i < tries; i++) {
    if (evalJs(js).includes("MARKER_OK")) return true;
    sleep(500);
  }
  return false;
};

const waitSelector = (sel, tries = 24) => {
  const js = `document.querySelector(${JSON.stringify(sel)}) ? 'MARKER_OK' : 'MARKER_NO'`;
  for (let i = 0; i < tries; i++) {
    if (evalJs(js).includes("MARKER_OK")) return true;
    sleep(500);
  }
  return false;
};

const screenshot = (name) => {
  const path = join(outDir, `${name}.png`);
  ab(["screenshot", path]);
  const kb = Math.round(statSync(path).size / 1024);
  if (kb < 20) return { name, ok: false, note: `BLANK ${kb}KB` };
  return { name, ok: true, kb };
};

// First paint sets the app's own dark preference, then a reload applies it —
// the site and every docs screenshot are dark-first, so the videos match.
const openDark = (mode, tag) => {
  // exact viewport first — the app renders responsive shells and the still
  // must match the documented device (desktop 1280x720, phone 390x844)
  const vp = mode === "mobile" ? "390 844" : "1280 720";
  try { ab(["set", "viewport", ...vp.split(" ")]); } catch {}
  ab(["open", `${base}/?${mode}=1&_v=${tag}`]);
  sleep(1200);
  evalJs(`localStorage.setItem('picode-theme','dark'); 'set'`);
  ab(["open", `${base}/?${mode}=1&_v=${tag}r`]);
  sleep(2600);
};
const openDesktop = (tag) => openDark("desktop", tag);
const openMobile = (tag) => openDark("mobile", tag);
const goTo = (hash) => { evalJs(`location.hash = ${JSON.stringify(hash)}; 'nav'`); sleep(1800); };
function atlasId() {
  const out = evalJs(
    `fetch('/api/workspaces').then(r=>r.json()).then(a=>{const ws=(Array.isArray(a)?a:[]).find(w=>w.agent&&w.agent.name==='Atlas'); return ws&&ws.agent ? 'ID:'+ws.agent.id : 'ID:NO'}).catch(()=>'ID:ERR')`,
  );
  const m = out.match(/ID:([A-Za-z0-9-_]+)/);
  return m ? m[1] : null;
}

const steps = [
  {
    // ── V1: create an agent (desktop) ──
    name: "v1-1-dashboard",
    run: () => {
      openDesktop("v11");
      return waitText("Atlas");
    },
  },
  {
    name: "v1-1b-agents-tab",
    run: () => {
      openDesktop("v11b");
      if (!waitText("Atlas")) return false;
      ab(["click", 'button[title="Agents"]']);
      sleep(1000);
      return true;
    },
  },
  {
    name: "v1-2-form",
    run: () => {
      openDesktop("v12");
      if (!waitText("Atlas")) return false;
      // Agents tab → New agent opens the CreateForm dialog (native clicks —
      // React synthetic handlers; eval .click() does not reach them here)
      ab(["click", 'button[title="Agents"]']);
      sleep(600);
      ab(["click", 'button[title="New agent"]']);
      if (!waitSelector('input[placeholder="Name"]')) return false;
      try {
        ab(["fill", 'input[placeholder="Name"]', "Docs demo"]);
      } catch { return false; }
      sleep(400);
      return true;
    },
  },
  {
    name: "v1-3-running",
    run: () => {
      openDesktop("v13");
      if (!waitText("Atlas")) return false;
      const id = atlasId();
      if (!id) return false;
      goTo(`#/agent/${id}`);
      return true;
    },
  },
  {
    // ── V2: automate it (desktop) ──
    name: "v2-1-list",
    run: () => {
      openDesktop("v21");
      if (!waitText("Atlas")) return false;
      goTo("#/automations");
      return waitText("Automation");
    },
  },
  {
    name: "v2-2-detail",
    run: () => {
      openDesktop("v22");
      if (!waitText("Atlas")) return false;
      goTo("#/automations");
      if (!waitText("Automation")) return false;
      // open the first automation's detail (anchor row)
      ab(["click", ".auto-row-main"]);
      sleep(1200);
      return true;
    },
  },
  {
    name: "v2-3-inbox",
    run: () => {
      openDesktop("v23");
      if (!waitText("Atlas")) return false;
      goTo("#/app/inbox");
      return waitText("Inbox");
    },
  },
  {
    // ── V3: take it anywhere (mobile) ──
    // The Now screen does not name agents — the Work screen lists them.
    name: "v3-1-fleet",
    run: () => {
      openMobile("v31");
      if (!waitText("Needs you")) return false;
      goTo("#/work");
      return waitText("Atlas");
    },
  },
  {
    name: "v3-2-agent",
    run: () => {
      openMobile("v32");
      if (!waitText("Needs you")) return false;
      const id = atlasId();
      if (!id) return false;
      goTo(`#/agent/${id}`);
      return waitText("Atlas");
    },
  },
  {
    name: "v3-3-inbox",
    run: () => {
      openMobile("v33");
      if (!waitText("Needs you")) return false;
      goTo("#/inbox");
      return waitText("Inbox");
    },
  },
];

const results = [];
let good = 0;
for (const s of steps) {
  session = `video-stills-${process.pid}-${s.name}`;
  let ok = false;
  try {
    ok = !!s.run();
  } catch (e) {
    console.error(`  [${s.name}] ERROR ${String(e.message).slice(0, 120)}`);
  }
  if (ok) {
    sleep(600); // compositor settle
    const r = screenshot(s.name);
    results.push(r);
    if (r.ok) {
      good++;
      console.log(`  ${s.name}.png (${r.kb}KB)`);
    } else {
      console.error(`  [${s.name}] ${r.note}`);
    }
  } else {
    results.push({ name: s.name, ok: false });
    console.error(`  [${s.name}] MARKER_FAIL`);
  }
}

writeFileSync(join(outDir, "stills.json"), JSON.stringify({ session, results }, null, 2));
console.log(`docs-video-stills: ${good}/${steps.length} stills -> ${outDir}`);
if (good !== steps.length) process.exit(1);
