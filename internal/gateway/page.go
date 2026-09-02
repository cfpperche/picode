package gateway

import (
	"encoding/json"
	"io"
	"net/http"
)

// PageCSP is the gateway's own pages' policy (mirrors internal/server):
// inline styles, no scripts, nothing external, no framing.
const PageCSP = "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"

func decodeJSON(res *http.Response, dest any) error {
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

// pageTemplate mirrors internal/server's pairPageTemplate (kept in sync
// by eye: the gateway must not import the server).
const pageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="theme-color" content="#0e0e11">
<title>%s · PiCode</title>
<style>
:root { --bg-base: #0e0e11; --bg-panel: #16161c; --border: #232329; --text-primary: #ececf1; --text-secondary: #9b9ba7; --accent: #7c8cf8;
  --sans: -apple-system, "Segoe UI", Inter, Roboto, sans-serif; --serif: Georgia, ui-serif, serif; color-scheme: dark; }
@media (prefers-color-scheme: light) { :root { --bg-base: #ffffff; --bg-panel: #f7f8fa; --border: #dfe3ea; --text-primary: #16181d; --text-secondary: #5b6472; --accent: #2f6fed; color-scheme: light; } }
* { box-sizing: border-box; }
body { margin: 0; min-height: 100dvh; display: flex; align-items: center; justify-content: center; padding: 24px; background: var(--bg-base); color: var(--text-primary); font: 15px/1.55 var(--sans); }
.card { width: 100%%; max-width: 380px; background: var(--bg-panel); border: 1px solid var(--border); border-radius: 14px; padding: 28px 26px 26px; }
.brand { display: flex; align-items: center; gap: 10px; margin-bottom: 22px; }
.mark { font: 700 26px/1 var(--serif); color: var(--accent); width: 32px; text-align: center; }
.name { font-weight: 650; letter-spacing: .2px; font-size: 15px; }
h1 { font-size: 18px; font-weight: 650; margin: 0 0 10px; line-height: 1.3; }
p { margin: 0; color: var(--text-secondary); font-size: 13.5px; }
code { font-family: ui-monospace, Menlo, monospace; font-size: 12.5px; color: var(--text-primary); }
.foot { margin: 18px 0 0; font-size: 11.5px; color: var(--text-secondary); text-align: center; }
.actions { display: flex; flex-direction: column; gap: 10px; margin-top: 20px; }
.actions:empty { display: none; }
.btn { display: block; text-align: center; text-decoration: none; font-weight: 600; font-size: 14px; height: 44px; line-height: 44px; border-radius: 9px; background: var(--accent); color: #fff; }
.btn:hover { filter: brightness(1.08); }
a { color: var(--accent); }
</style>
</head>
<body>
  <main class="card" role="main">
    <div class="brand"><span class="mark" aria-hidden="true">π</span><span class="name">PiCode</span></div>
    <h1>%s</h1>
    <p>%s</p>
    <p class="foot">The browser is a door, not a cage.</p>
  </main>
</body>
</html>
`
