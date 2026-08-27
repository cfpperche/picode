// Headless MCP OAuth: callback only, no paste UI. pi -e
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

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
