// ADR-0170: best-effort facts, never authorization or a command queue.
import { execFileSync } from "node:child_process";
import { randomUUID } from "node:crypto";
import { constants, openSync, closeSync, fstatSync, lstatSync, mkdirSync, readFileSync, renameSync, unlinkSync, writeFileSync, realpathSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

function git(cwd, ...args) { return execFileSync("git", ["--no-optional-locks", "-c", "core.fsmonitor=false", ...args], { cwd, encoding: "utf8", timeout: 5000, maxBuffer: 1024 * 1024, stdio: ["ignore", "pipe", "ignore"] }).trim(); }
function optional(cwd, ...args) { try { return git(cwd, ...args); } catch { return ""; } }
function directory(cwd) {
  const repo = realpathSync(resolve(cwd, git(cwd, "rev-parse", "--git-common-dir")));
  const dir = join(repo, "picode-delivery");
  try { mkdirSync(dir, { mode: 0o700 }); } catch (e) { if (e.code !== "EEXIST") throw e; }
  const st = lstatSync(dir);
  if (!st.isDirectory() || st.isSymbolicLink() || (process.platform !== "win32" && (st.mode & 0o077))) throw new Error("unsafe evidence directory");
  return { repo, dir };
}
function write(dir, row) {
  const name = join(dir, row.id + ".json");
  const tmp = join(dir, "." + randomUUID() + ".tmp");
  let fd;
  try {
    fd = openSync(tmp, constants.O_CREAT | constants.O_EXCL | constants.O_WRONLY | constants.O_NOFOLLOW, 0o600);
    writeFileSync(fd, JSON.stringify(row) + "\n"); closeSync(fd); fd = undefined;
    renameSync(tmp, name);
  } finally { if (fd !== undefined) closeSync(fd); try { unlinkSync(tmp); } catch {} }
}
export function startReceipt(cwd, kind, sourceRef = "") {
  if (!["scoped", "full-ci", "land"].includes(kind)) throw new Error("invalid receipt kind");
  const { repo, dir } = directory(cwd);
  const ref = sourceRef || optional(cwd, "symbolic-ref", "--short", "HEAD");
  if (ref) git(cwd, "check-ref-format", "refs/heads/" + ref);
  const row = { schemaVersion: 1, id: randomUUID(), kind, repositoryKey: repo, sourceRef: ref,
    source: git(cwd, "rev-parse", "--verify", ref ? `refs/heads/${ref}^{commit}` : "HEAD"),
    tree: git(cwd, "rev-parse", ref ? `refs/heads/${ref}^{tree}` : "HEAD^{tree}"),
    targetBefore: optional(cwd, "rev-parse", "--verify", "refs/heads/main^{commit}"),
    startedAt: new Date().toISOString(), outcome: "started", clean: git(cwd, "status", "--porcelain") === "" };
  write(dir, row); return row.id;
}
export function finishReceipt(cwd, id, code) {
  if (!/^[a-f0-9-]{36}$/.test(id)) throw new Error("invalid receipt id");
  const { repo, dir } = directory(cwd);
  if (lstatSync(join(dir,id+".json")).isSymbolicLink()) throw new Error("unsafe receipt file");
  const fd = openSync(join(dir, id + ".json"), constants.O_RDONLY | constants.O_NOFOLLOW);
  let row;
  try { const st = fstatSync(fd); if (!st.isFile() || st.size > 65536) throw new Error("invalid receipt"); row = JSON.parse(readFileSync(fd, "utf8")); } finally { closeSync(fd); }
  if (row.id !== id || row.repositoryKey !== repo || row.outcome !== "started") throw new Error("receipt mismatch");
  row.finishedAt = new Date().toISOString();
  row.targetAfter = optional(cwd, "rev-parse", "--verify", "refs/heads/main^{commit}");
  row.outcome = code === 0 ? "passed" : "failed";
  row.clean = row.clean && optional(cwd, "rev-parse", "HEAD") === row.source && git(cwd, "status", "--porcelain") === "";
  if (row.kind === "scoped" && code === 0) {
    try { row.scope = JSON.parse(readFileSync(join(resolve(cwd, git(cwd, "rev-parse", "--git-dir")), "picode-ci-scoped.json"), "utf8")); } catch {}
  }
  write(dir, row);
}
// Evidence errors cannot change the command's exit code. No raw Git error/log is retained.
export function recordSafely(fn) { try { return fn(); } catch { console.error("delivery: could not record command evidence"); return ""; } }
if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  const [action, value, extra] = process.argv.slice(2);
  if (action === "start") { const id = recordSafely(() => startReceipt(process.cwd(), value, extra)); if (id) console.log(id); }
  else if (action === "finish" && value) recordSafely(() => finishReceipt(process.cwd(), value, Number(extra)));
}
