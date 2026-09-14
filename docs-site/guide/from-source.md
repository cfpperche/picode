---
description: Build PiCode from this repository when you are changing it.
---

# From source

This is how you change PiCode. To run it, use [Getting started](/guide/getting-started).

Needs [Go 1.26+](https://go.dev), [Node.js 22](https://nodejs.org) (the version used by CI), [Pi](https://www.npmjs.com/package/@earendil-works/pi-coding-agent), and tmux 3.5+.

```bash
git clone https://github.com/cfpperche/picode.git
cd picode
make build
./bin/picode install
```

| Command | What it does |
|---|---|
| `make web` | Build the UI once — needed before `make dev` on a fresh clone |
| `make dev` | Run from this checkout without installing |
| `make deploy` | Rebuild this checkout and restart the installed service. Refuses while an agent or a terminal is mid-turn (`picode deploy --force` overrides) |
| `make cert` | Install a locally trusted mkcert certificate |

Bind and TLS details live in the [README](https://github.com/cfpperche/picode#quick-start).
