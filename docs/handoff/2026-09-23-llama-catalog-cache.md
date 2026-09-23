# 2026-09-23 — llama-catalog-cache: the catalog's llama.cpp cache, made honest

Shipped: fixes for the three cache findings of today's adversarial llama.cpp review (`internal/server/llama.go`).
A probe stores under the URL/key it asked (a switch mid-probe can no longer serve A's models for B); forget bumps
a generation so an older answer stays stale; model jobs reaching a terminal state and any `llama.service` event
forget (feed listener); `?fresh=1` waits for a new answer; concurrent cold reads share one probe per setting.
The llama pane's `/api/llama` files its answer under the setting its request read and returns that URL.
Verified: `make ci-scoped` PASS; six new tests (keep a failure, shared cold probe, A-never-under-B, forget during
a probe, fresh, listener) pass 5× under `-race`.
Found on the way: on the owner's WSL (mirrored networking) every closed 127.0.0.1 port drops the connection
instead of refusing it, so an unused llama.cpp setting costs a full timeout; nothing listens on :8080.

## Next up

- Branch 2 of the review: `/api/llama` bounded by the request context (~3 s), no Capabilities after a failed List,
  Save not blocked while checking, honest copy (TLS/cancelled), LlamaService.jsx in-flight guard, HF pick race
