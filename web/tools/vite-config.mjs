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

export function applicationConfig(application, port) {
  return defineConfig({
    root: fileURLToPath(new URL(`../${application}/`, import.meta.url)),
    base: `/${application}/`,
    publicDir: fileURLToPath(new URL("../public/", import.meta.url)),
    plugins: [react(), tailwindcss(), applicationBoundary(application)],
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
