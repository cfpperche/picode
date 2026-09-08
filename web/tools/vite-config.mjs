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
  return defineConfig({
    root: fileURLToPath(new URL(`../${application}/`, import.meta.url)),
    base: `/${application}/`,
    publicDir: fileURLToPath(new URL("../public/", import.meta.url)),
    plugins: [react(), tailwindcss(), applicationBoundary(application), rootBrandLinks(application)],
    build: {
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
