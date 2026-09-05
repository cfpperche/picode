import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const web = fileURLToPath(new URL("../", import.meta.url));
const children = ["desktop", "mobile", "launcher"].map(app => spawn(process.execPath, [web + "node_modules/vite/bin/vite.js", "--config", app + "/vite.config.js"], { cwd: web, stdio: "inherit" }));
let closing = false;
function close(code = 0) {
  if (closing) return;
  closing = true;
  process.exitCode = code;
  for (const child of children) child.kill("SIGTERM");
}
for (const child of children) {
  child.on("error", error => { console.error(error); close(1); });
  child.on("exit", code => close(code || 0));
}
process.on("SIGINT", () => close());
process.on("SIGTERM", () => close());
