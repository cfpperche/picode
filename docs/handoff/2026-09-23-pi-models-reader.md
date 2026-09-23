# 2026-09-23 — pi-models-reader: Pi is a climodels reader; the catalog stops stalling

Shipped: `internal/climodels` registers readers per CLI (command, probe, input files) over one cache; omp unchanged,
Pi added (`pi.go`: `pi --list-models --offline`, parser moved here, parity test against the old one).
`catalog.Load` reads Pi's list through it; `/api/catalog`, the status bar and the usage roster stop running Pi
per call; `?fresh=1` from Providers "Try again". The catalog's llama.cpp probe is served from memory and refreshed
in the background after 30 s. ADR-0009 amended (the approved text, plus two sentences on the llama.cpp finding).
Measured (pi 0.87.1, copy of the owner's home): the list depends on auth.json, models.json, models-store.json,
settings.json (packages), npm/package-lock.json and the binary; listing rewrites none; `enabledModels` (global or
project) does not change it; the command takes 0.2–0.5 s. The 2 s per catalog read was the llama.cpp probe: the
owner's configured server 127.0.0.1:8080 accepts and never answers.
Verified: `make ci-scoped` PASS; scratch: catalog 2.0 s on the first read after start, then 8 ms, still 8 ms past
the 30 s bound; a sign-in through PiCode shows at the next read; usage summary 8 ms.
Not changed: pickers still read `/api/catalog` (moving them to `/api/cli-models` is the next step); the Models pane
stays omp-only (`Panes()`); the owner's port 8080 process still hangs every client.

## Next up

- Model pickers read `/api/cli-models` (one source for every CLI), then measure the other seven CLIs for readers
