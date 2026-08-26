// Silent bridge: adapter status → ~/.picode/mcp-live/<agent>.json
// Loaded with pi -e. Registers nothing the model can see.
import fs from "node:fs";
import path from "node:path";

export default function (pi) {
  const dest = process.env.PICODE_MCP_LIVE;
  if (!dest || !pi || !pi.events || typeof pi.events.on !== "function") return;
  pi.events.on("pi-mcp-adapter/status/v1", (snap) => {
    try {
      fs.mkdirSync(path.dirname(dest), { recursive: true, mode: 0o700 });
      fs.writeFileSync(dest, JSON.stringify(snap || {}), { mode: 0o600 });
    } catch {
      // ignore: GUI falls back to Idle
    }
  });
}
