import { readdirSync } from "node:fs";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";
import { assertResolvedBoundaries, checkBoundaries } from "./boundaries.mjs";

const webRoot = fileURLToPath(new URL("../", import.meta.url));

// Check the resolved graph: aliases and transitive CSS imports also obey
// the application boundary.
function applicationBoundary(application) {
  return {
    name: "picode-application-boundary",
    async buildStart() { await checkBoundaries(webRoot, [application, "shared"]); },
    buildEnd(error) {
      if (error) return;
      assertResolvedBoundaries(webRoot, application, this);
    },
  };
}

// Brand files (favicon, Apple touch icon, PWA manifest) are site-root files
// served by the launcher build — ADR-0072 keeps the root PWA files. Vite
// prefixes the application base to root-absolute hrefs in index.html, so
// undo exactly that for these links after the built-in rewrite, or the
// browser tab loses its icon and the manifest 404s.
function rootBrandLinks(application) {
  const rootBrandFiles = ["manifest.json", "favicon.svg", "apple-touch-icon.png"];
  const rootBrand = new RegExp(
    `(href="/)${application}/(${rootBrandFiles.join("|")})"`,
    "g",
  );
  return {
    name: `picode-root-brand-links-${application}`,
    transformIndexHtml: {
      order: "post",
      handler: (html) => html.replace(rootBrand, '$1$2"'),
    },
  };
}

export function applicationConfig(application, port) {
  const appRoot = fileURLToPath(new URL(`../${application}/`, import.meta.url));
  // Every top-level .html in the application directory is a page (the shell
  // opens /desktop/management.html in its own window; without an entry here
  // the build silently drops it and the tray item 404s — 2026-09-12).
  const pages = Object.fromEntries(
    readdirSync(appRoot)
      .filter((f) => f.endsWith(".html"))
      .map((f) => [f.replace(/\.html$/, ""), appRoot + f]),
  );
  return defineConfig({
    root: appRoot,
    base: `/${application}/`,
    publicDir: fileURLToPath(new URL("../public/", import.meta.url)),
    plugins: [react(), tailwindcss(), applicationBoundary(application), rootBrandLinks(application)],
    // esbuild's minifier miscompiles the requestMode enum in xterm 6's ESM
    // build: it inlines the enum variable as `void 0` but keeps the write
    // `(n = {})` without its declaration, so the first DECRQM query a TUI
    // sends (OpenCode boots straight into one) throws
    // "ReferenceError: n is not defined" inside the parser and stalls xterm's
    // write pipeline for good — the terminal freezes until a reload. The UMD
    // build ships the same code with the enum already compiled away, so pin
    // the exact specifier to it (`$` anchor keeps subpaths like css intact).
    // Revisit when xterm removes the enum or esbuild stops dropping the
    // declaration: grep the built bundle for `(void 0||(` next to requestMode.
    resolve: {
      alias: [{ find: /^@xterm\/xterm$/, replacement: "@xterm/xterm/lib/xterm.js" }],
    },
    build: {
      rollupOptions: { input: pages },
      outDir: `../../internal/web/public/${application}`,
      emptyOutDir: true,
      assetsDir: "assets",
      manifest: true,
      copyPublicDir: false,
    },
    server: {
      port,
      strictPort: true,
      hmr: { clientPort: port },
      proxy: {
        "/api": { target: "https://localhost:8445", secure: false },
        "/ws": { target: "wss://localhost:8445", ws: true, secure: false },
      },
    },
  });
}
