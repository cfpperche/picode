# HTTP API

Everything the PiCode UI does, it does over an HTTP JSON API served by the same daemon you can talk to yourself. The reference below is not hand-written documentation: it is **generated from the server's route registration**, so it can never describe an endpoint the binary doesn't serve — or miss one it does.

## Browse it

The full reference lives in Scalar's viewer:

**[Open the API reference →](/api/)**

Every operation shows its method, path, and grouping (workspaces, agents, sessions, automations, terminals, backup, …). Authenticated requests use the same session your browser already has after pairing.

## Raw spec

- [`api/openapi.json`](/api/openapi.json) — OpenAPI 3.1, 200+ operations, regenerable with `make openapi`
- [`llms.txt`](/llms.txt) — a machine-readable map of this whole site, including the spec, for LLM consumption

## Call it yourself

Pair once (the same pairing your browser uses — see [Security and pairing](/guide/security)), then reuse that session cookie:

```bash
# Pair a shell: prints a URL to approve in your browser
curl -X POST http://127.0.0.1:8445/api/auth/pair/start

# With an approved session cookie, list your workspaces
curl -b "picode_session=<secret>" http://127.0.0.1:8445/api/workspaces
```

::: tip Ungated mode
A daemon started with `PICODE_INSECURE=1` (development only) skips pairing entirely — useful when scripting against a throwaway instance.
:::

## Why generated

The spec is produced by `cmd/picode-openapi`, which walks the **same `registerAll` call the binary makes at startup**. CI re-runs the generator and byte-compares the result with the committed spec (`make docs-check`): a route added in Go without regenerating the spec fails the build, exactly like a UI change without fresh screenshots.
