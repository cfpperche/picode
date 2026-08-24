# PiCode web UI

React 19 + Vite + Tailwind CSS v4. Design tokens and component classes
live in `src/styles/app.css` — same palette as the pre-ADR-0008 UI.

```bash
npm install
npm run dev     # HMR on :5173 (proxies /api and /ws to :8445)
npm run build   # writes ../internal/web/public (embedded by the Go binary)
```

From the repo root: `make ui` (HMR) or `make web` (production assets).
