import { rmSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { checkBoundaries } from "./boundaries.mjs";

const web = fileURLToPath(new URL("../", import.meta.url));
await checkBoundaries(web);
rmSync(fileURLToPath(new URL("../../internal/web/public", import.meta.url)), { recursive: true, force: true });
for (const app of ["launcher", "browser", "desktop", "mobile"]) {
  const result = spawnSync(process.execPath, [web + "node_modules/vite/bin/vite.js", "build", "--config", app + "/vite.config.js"], { cwd: web, stdio: "inherit" });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status || 1);
}
