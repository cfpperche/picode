import { defineConfig } from "vite";
import { fileURLToPath } from "node:url";

export default defineConfig({
  root: fileURLToPath(new URL("./", import.meta.url)),
  publicDir: "../public",
  build: { outDir: "../../internal/web/public", emptyOutDir: false },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/browser/": "http://localhost:5174",
      "/desktop/": "http://localhost:5176",
      "/mobile/": "http://localhost:5175",
      "/api": { target: "https://localhost:8445", secure: false },
      "/ws": { target: "wss://localhost:8445", ws: true, secure: false },
    },
  },
});
