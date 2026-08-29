# Getting started

Requires [Go 1.22+](https://go.dev), [Pi](https://www.npmjs.com/package/@earendil-works/pi-coding-agent), tmux 3.5+.

```bash
git clone https://github.com/cfpperche/picode.git
cd picode
make build
./bin/picode install    # systemd --user; starts when this Linux session starts
```

Open https://localhost:8445. Add a workspace (your project folder) or a free agent, click **Run**. Close the tab; the agent keeps running.

`picode uninstall` removes the service. `--purge` also deletes `~/.picode`.

```bash
make dev          # run from the repo without installing
```

Green padlock: `make cert` (mkcert). See the [README](https://github.com/cfpperche/picode#quick-start) for TLS and bind details.

Personal use is free under PolyForm Noncommercial. Company use needs a [commercial license](/license).
