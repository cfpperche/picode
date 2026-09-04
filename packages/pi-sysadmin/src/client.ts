import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { request as httpRequest } from "node:http";
import { request as httpsRequest } from "node:https";

export function connection(env: Record<string, string | undefined>, home: string, read: (path: string) => string) {
  const dir = env.PICODE_DATA?.trim() || join(home, ".picode");
  let base = env.PICODE_URL?.trim();
  if (!base) base = JSON.parse(read(join(dir, "server.json"))).url;
  const url = new URL(base || "");
  if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.search || url.hash) throw new Error("Invalid PiCode server URL");
  let token = env.PICODE_TOKEN?.trim();
  if (!token) { try { token = read(join(dir, "token")).trim(); } catch { token = ""; } }
  if (token && !/^[a-f0-9]{32,128}$/i.test(token)) throw new Error("Invalid PiCode token");
  const loopback = ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname);
  if (url.protocol === "http:" && !loopback) throw new Error("Remote PiCode connections require HTTPS");
  return { base: url.toString().replace(/\/+$/, ""), token, rejectUnauthorized: !loopback };
}

export async function requestJSON(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<any> {
  const cfg = connection(process.env, homedir(), (path) => readFileSync(path, "utf8"));
  const url = new URL(cfg.base + path);
  const payload = body === undefined ? "" : JSON.stringify(body);
  return new Promise((resolve, reject) => {
    const req = (url.protocol === "https:" ? httpsRequest : httpRequest)(url, {
      method, signal, rejectUnauthorized: cfg.rejectUnauthorized, timeout: 15000,
      headers: { "content-type": "application/json", "content-length": Buffer.byteLength(payload), ...(cfg.token ? { authorization: "Bearer " + cfg.token } : {}) },
    }, (res) => {
      let bytes = 0;
      const chunks: Buffer[] = [];
      res.on("data", (chunk: Buffer) => {
        bytes += chunk.length;
        if (bytes > 1024 * 1024) { res.destroy(new Error("PiCode response exceeded 1 MiB")); return; }
        chunks.push(chunk);
      });
      res.on("error", reject);
      res.on("end", () => {
        try {
          const value = JSON.parse(Buffer.concat(chunks).toString("utf8"));
          if ((res.statusCode || 500) >= 300) reject(new Error(value.error || `PiCode returned HTTP ${res.statusCode}`));
          else resolve(value);
        } catch (err) { reject(err); }
      });
    });
    req.on("timeout", () => req.destroy(new Error("PiCode request timed out; check operation history before retrying")));
    req.on("error", reject);
    req.end(payload);
  });
}

export async function confirmOperation(action: string, name: string, ctx: { hasUI?: boolean; ui?: { confirm: (title: string, message: string) => Promise<boolean> } }): Promise<boolean> {
  if (action === "start") return true;
  if (action !== "stop" && action !== "restart") throw new Error("Unsupported Docker action");
  if (!ctx?.hasUI || !ctx.ui) throw new Error("Open this container in the Docker App to confirm a disruptive operation");
  return ctx.ui.confirm(`${action === "stop" ? "Stop" : "Restart"} ${name}?`, "Connections to this container may be interrupted.");
}
