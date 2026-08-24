import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// Production assets land in internal/web/public so go:embed (ADR-0001)
// is unchanged. Dev server proxies API/WS to the running picode binary.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "../internal/web/public",
    emptyOutDir: true,
    assetsDir: "assets",
  },
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "https://localhost:8445", secure: false },
      "/ws": { target: "wss://localhost:8445", ws: true, secure: false },
    },
  },
});
