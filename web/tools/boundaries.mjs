import { readFileSync, readdirSync, realpathSync, existsSync } from "node:fs";
import { resolve, relative, dirname, extname } from "node:path";
import { fileURLToPath } from "node:url";

export function sourceFiles(root) {
  return readdirSync(root, { withFileTypes: true }).flatMap(entry => {
    if (entry.name === "node_modules") return [];
    const path = resolve(root, entry.name);
    return entry.isDirectory() ? sourceFiles(path) : /\.(?:jsx?|css)$/.test(path) && !path.endsWith(".test.js") ? [path] : [];
  });
}

export function importSpecifiers(source) {
  const patterns = [
    /(?:^|\n)\s*(?:import|export)\s+(?:[^;'"`]*?\s+from\s*)?["']([^"']+)["']/g,
    /\bimport\s*\(\s*["']([^"']+)["']\s*\)/g,
    /@import\s+(?:url\(\s*)?["']([^"']+)["']/g,
  ];
  return [...new Set(patterns.flatMap(pattern => [...source.matchAll(pattern)].map(match => match[1])))];
}

function resolveFile(path) {
  for (const candidate of [path, ...[".js", ".jsx", ".css", ".json"].map(ext => path + ext)]) {
    if (existsSync(candidate)) return realpathSync(candidate);
  }
  return path;
}

export function checkBoundaries(webRoot, applications = ["desktop", "mobile", "shared"]) {
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
      if (app === "shared" && extname(path) === ".jsx") errors.push(`${name}: shared must not contain JSX`);
      for (const spec of importSpecifiers(readFileSync(path, "utf8"))) {
        if (spec.startsWith(".")) {
          const target = relative(resolve(root, app), resolveFile(resolve(dirname(path), spec))).replaceAll("\\", "/");
          if (target.startsWith("../") || target === "..") errors.push(`${name}: cross-application import ${spec}`);
        } else if (spec.startsWith("@picode/shared/")) {
          if (app !== "shared" && !Object.hasOwn(manifest.dependencies || {}, "@picode/shared")) errors.push(`${name}: undeclared dependency @picode/shared`);
          const subpath = "./" + spec.slice("@picode/shared/".length);
          if (!Object.hasOwn(shared.exports || {}, subpath)) errors.push(`${name}: private shared import ${spec}`);
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

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  checkBoundaries(fileURLToPath(new URL("../", import.meta.url)));
  console.log("Application boundaries: PASS");
}
