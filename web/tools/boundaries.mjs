import { readFileSync, readdirSync, realpathSync, existsSync } from "node:fs";
import { resolve, relative, dirname, extname } from "node:path";
import { parseAst, transformWithEsbuild } from "vite";
import { fileURLToPath } from "node:url";

export function sourceFiles(root) {
  return readdirSync(root, { withFileTypes: true }).flatMap(entry => {
    if (entry.name === "node_modules") return [];
    const path = resolve(root, entry.name);
    return entry.isDirectory() ? sourceFiles(path) : /\.(?:[cm]?[jt]sx?|css)$/.test(path) && !/\.test\.[cm]?[jt]sx?$/.test(path) ? [path] : [];
  });
}

export async function importSpecifiers(source, filename = "module.js") {
  if (filename.endsWith(".css")) return [...source.matchAll(/@import\s+(?:url\(\s*)?["']([^"']+)["']/g)].map(match => match[1]);
  const code = /\.(?:jsx|tsx|[cm]?ts)$/.test(filename)
    ? (await transformWithEsbuild(source, filename, { jsx: "preserve", tsconfigRaw: { compilerOptions: { verbatimModuleSyntax: true } } })).code
    : source;
  // JSX must be lowered before Rollup's parser; retain every import.
  const js = /\.(?:jsx|tsx)$/.test(filename)
    ? (await transformWithEsbuild(code, "module.jsx", { jsx: "transform" })).code : code;
  const found = new Set();
  function visit(node) {
    if (!node || typeof node !== "object") return;
    if (["ImportDeclaration", "ExportNamedDeclaration", "ExportAllDeclaration", "ImportExpression"].includes(node.type) && typeof node.source?.value === "string") found.add(node.source.value);
    if (node.type === "CallExpression" && node.callee?.name === "require" && typeof node.arguments?.[0]?.value === "string") found.add(node.arguments[0].value);
    for (const value of Object.values(node)) {
      if (Array.isArray(value)) value.forEach(visit);
      else if (value && typeof value === "object") visit(value);
    }
  }
  visit(parseAst(js));
  return [...found];
}

function resolveFile(path) {
  for (const candidate of [path, ...[".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts", ".css", ".json"].map(ext => path + ext)]) {
    if (existsSync(candidate)) return realpathSync(candidate);
  }
  return path;
}

// The desktop bundle composes the browser app's exported App (ADR-0122:
// one app, two entries). The compose may import exactly the exports the
// browser package names — nothing else crosses.
const COMPOSES = { desktop: "browser" };

export async function checkBoundaries(webRoot, applications = ["browser", "desktop", "mobile", "shared"]) {
  const root = realpathSync(webRoot);
  const shared = JSON.parse(readFileSync(resolve(root, "shared/package.json"), "utf8"));
  const errors = [];
  for (const [name, target] of Object.entries(shared.exports || {})) {
    if (typeof target !== "string" || !target.startsWith("./") || !existsSync(resolve(root, "shared", target)) || relative(resolve(root, "shared"), resolveFile(resolve(root, "shared", target))).startsWith("..")) {
      errors.push(`shared: invalid public export ${name}`);
    }
  }
  for (const app of applications) {
    const manifest = JSON.parse(readFileSync(resolve(root, app, "package.json"), "utf8"));
    const directory = resolve(root, app, app === "shared" ? "." : "src");
    for (const path of sourceFiles(directory)) {
      const name = relative(root, path).replaceAll("\\", "/");
      if (app === "shared" && [".jsx", ".tsx"].includes(extname(path))) errors.push(`${name}: shared must not contain JSX`);
      for (const spec of await importSpecifiers(readFileSync(path, "utf8"), path)) {
        if (spec.startsWith(".")) {
          const target = relative(resolve(root, app), resolveFile(resolve(dirname(path), spec))).replaceAll("\\", "/");
          if (target.startsWith("../") || target === "..") errors.push(`${name}: cross-application import ${spec}`);
        } else if (spec.startsWith("@picode/shared/")) {
          if (app !== "shared" && !Object.hasOwn(manifest.dependencies || {}, "@picode/shared")) errors.push(`${name}: undeclared dependency @picode/shared`);
          const subpath = "./" + spec.slice("@picode/shared/".length);
          if (!Object.hasOwn(shared.exports || {}, subpath)) errors.push(`${name}: private shared import ${spec}`);
        } else if (spec.startsWith("@picode/browser/")) {
          const composed = COMPOSES[app];
          if (!composed) errors.push(`${name}: ${spec} — only the desktop bundle composes the browser app`);
          const browserManifest = JSON.parse(readFileSync(resolve(root, "browser/package.json"), "utf8"));
          const subpath = "./" + spec.slice("@picode/browser/".length);
          if (!Object.hasOwn(browserManifest.exports || {}, subpath)) errors.push(`${name}: private browser import ${spec}`);
          if (!Object.hasOwn(manifest.dependencies || {}, "@picode/browser")) errors.push(`${name}: undeclared dependency @picode/browser`);
        } else {
          const dependency = spec.startsWith("@") ? spec.split("/").slice(0, 2).join("/") : spec.split("/")[0];
          if (!Object.hasOwn({ ...manifest.dependencies, ...manifest.devDependencies }, dependency)) errors.push(`${name}: undeclared dependency ${dependency}`);
          if (app === "shared" && ["react", "react-dom", "sonner", "vaul"].includes(dependency)) errors.push(`${name}: presentation dependency in shared`);
        }
      }
    }
  }
  if (Object.hasOwn(shared.exports || {}, ".") || Object.keys(shared.exports || {}).some(key => key.includes("*"))) errors.push("shared: exports must name individual public modules");
  if (errors.length) throw new Error(errors.join("\n"));
}

export function assertResolvedBoundaries(root, app, graph) {
  const ids = [...graph.getModuleIds()];
  for (const id of ids) {
    if (!id.startsWith(root) || id.includes("/node_modules/")) continue;
    const path = relative(root, id).replaceAll("\\", "/");
    const composes = app === "desktop" && path.startsWith("browser/");
    if (!path.startsWith(app + "/") && !path.startsWith("shared/") && !composes) throw new Error(`${app} imports code outside its boundary: ${path}`);
    // Shared roots ban presentation dependencies; a composed browser root
    // is the app itself — react and friends are its point.
    const bansPresentation = path.startsWith("shared/");
    if (!path.startsWith("shared/") && !composes) continue;
    const pending = [id], seen = new Set();
    while (pending.length) {
      const dependency = pending.pop();
      if (seen.has(dependency)) continue;
      seen.add(dependency);
      if (bansPresentation && /\/node_modules\/(?:react|react-dom|vaul|sonner)(?:\/|$)/.test(dependency)) throw new Error(`Presentation dependency in shared graph: ${dependency}`);
      const info = graph.getModuleInfo(dependency);
      if (info) pending.push(...info.importedIds, ...info.dynamicallyImportedIds);
    }
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  await checkBoundaries(fileURLToPath(new URL("../", import.meta.url)));
  console.log("Application boundaries: PASS");
}
