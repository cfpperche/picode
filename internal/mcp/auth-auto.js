// Headless MCP OAuth: callback only. Pi does not open a tab (WSL PowerShell
// would). The GUI opens it so window.close() works. Success HTML is PiCode's.
import { EventEmitter } from "node:events";
import childProcess from "node:child_process";
import fs from "node:fs";
import http from "node:http";
import { registerHooks } from "node:module";
import path from "node:path";
import { pathToFileURL } from "node:url";

function picodePage(ok, back) {
  const heading = ok ? "Authentication complete" : "Authentication did not complete";
  const msg = ok ? "Returning to PiCode…" : "You can close this tab.";
  let script = "";
  if (ok) {
    const go = back ? "location.replace(" + JSON.stringify(back) + ")" : "";
    script = `<script>(function(){var n=3,el=document.getElementById("n");function tick(){if(el)el.textContent=n;if(n<=0){try{if(window.opener)window.opener.focus()}catch(e){}window.close();setTimeout(function(){${go}},200);return}n--;setTimeout(tick,1000)}setTimeout(tick,400)})()</script>`;
  }
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>PiCode</title>
<style>:root{--text:#fafafa;--dim:#a1a1aa;--bg:#09090b}*{box-sizing:border-box}html{color-scheme:dark}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:var(--bg);color:var(--text);font-family:ui-sans-serif,system-ui,sans-serif;text-align:center}main{max-width:480px}.logo{width:72px;height:72px;margin:0 auto 24px}h1{margin:0 0 10px;font-size:28px;font-weight:650}p{margin:0;color:var(--dim);font-size:15px;line-height:1.6}</style></head>
<body><main>
<svg class="logo" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 800" aria-hidden="true"><path fill="#fff" fill-rule="evenodd" d="M165.29 165.29H517.36V400H400V517.36H282.65V634.72H165.29ZM282.65 282.65V400H400V282.65Z"/><path fill="#fff" d="M517.36 400H634.72V634.72H517.36Z"/></svg>
<h1>${heading}</h1><p>${msg} <span id="n"></span></p>
</main>${script}</body></html>`;
}

function rewrite(chunk) {
  if (typeof chunk !== "string") return chunk;
  if (chunk.includes("Authorization Successful") || chunk.includes("Authorization Received")) {
    return picodePage(true, process.env.PICODE_MCP_RETURN || "");
  }
  if (chunk.includes("Authorization Failed")) {
    return picodePage(false, "");
  }
  return chunk;
}

const origEnd = http.ServerResponse.prototype.end;
http.ServerResponse.prototype.end = function (chunk, enc, cb) {
  if (typeof chunk === "string") chunk = rewrite(chunk);
  else if (Buffer.isBuffer(chunk)) {
    const s = chunk.toString("utf8");
    const n = rewrite(s);
    if (n !== s) chunk = Buffer.from(n);
  }
  return origEnd.call(this, chunk, enc, cb);
};

function fakeChild() {
  const ee = new EventEmitter();
  ee.unref = () => {};
  ee.ref = () => {};
  ee.kill = () => true;
  ee.pid = 0;
  queueMicrotask(() => ee.emit("close", 0));
  return ee;
}

function urlFromSpawn(command, args) {
  const parts = [command, ...(args || [])];
  for (const a of parts) {
    if (typeof a === "string" && /^https?:\/\//i.test(a)) return a;
  }
  const i = parts.findIndex((a) => String(a).toLowerCase() === "-encodedcommand");
  if (i >= 0 && parts[i + 1]) {
    const decoded = Buffer.from(String(parts[i + 1]), "base64").toString("utf16le");
    const m = decoded.match(/https?:\/\/\S+/);
    if (m) return m[0].replace(/"+$/, "");
  }
  return "";
}

function stealOpen(dest) {
  if (!dest) return;
  const orig = childProcess.spawn;
  childProcess.spawn = function (command, args, opts) {
    const url = urlFromSpawn(command, args);
    if (url) {
      fs.writeFileSync(dest, url);
      return fakeChild();
    }
    return orig.apply(this, arguments);
  };
  const stub = dest + ".mjs";
  fs.writeFileSync(
    stub,
    "export default async function open(target) {\n" +
      "  if (typeof target === 'string' && /^https?:\\/\\//i.test(target)) {\n" +
      "    const fs = await import('node:fs');\n" +
      "    fs.writeFileSync(" + JSON.stringify(dest) + ", target);\n" +
      "  }\n" +
      "  return { unref() {} };\n" +
      "}\n",
  );
  try {
    registerHooks({
      resolve(specifier, context, nextResolve) {
        if (specifier === "open") {
          return { url: pathToFileURL(stub).href, shortCircuit: true };
        }
        return nextResolve(specifier, context);
      },
    });
  } catch {
    // spawn intercept still holds
  }
}

export default function () {
  const name = process.env.PICODE_MCP_AUTH;
  const url = process.env.PICODE_MCP_AUTH_URL;
  const out = process.env.PICODE_MCP_AUTH_OUT;
  const adapter = process.env.PICODE_MCP_ADAPTER;
  if (!name || !url || !out || !adapter) return;

  const write = (obj) => {
    try {
      fs.mkdirSync(path.dirname(out), { recursive: true, mode: 0o700 });
      fs.writeFileSync(out, JSON.stringify(obj), { mode: 0o600 });
    } catch {
      // GUI times out
    }
  };

  (async () => {
    try {
      stealOpen(process.env.PICODE_MCP_OPEN);
      const candidates = [
        path.join(adapter, "mcp-auth-flow.ts"),
        path.join(adapter, "dist", "mcp-auth-flow.js"),
        path.join(adapter, "mcp-auth-flow.js"),
      ];
      let authenticate;
      let lastErr;
      for (const p of candidates) {
        try {
          const mod = await import(pathToFileURL(p).href);
          if (typeof mod.authenticate === "function") {
            authenticate = mod.authenticate;
            break;
          }
        } catch (e) {
          lastErr = e;
        }
      }
      if (!authenticate) throw lastErr || new Error("authenticate not found");
      const status = await authenticate(name, url, { url, auth: "oauth" }, {});
      write({ ok: status === "authenticated" });
    } catch (e) {
      write({ ok: false, error: String(e && e.message ? e.message : e) });
    }
  })();
}
