import test from "node:test";
import assert from "node:assert/strict";
import { vaultProblemText,
  healthChip, kindLabel, orderAccounts, providerName, sourceLabel, VERIFY_LABEL,
} from "./credentials.js";

test("every vault provider the guest CLIs declare has a name a person reads", () => {
  const names = {
    anthropic: "Anthropic",
    openai: "OpenAI",
    "openai-codex": "ChatGPT (Codex)",
    xai: "xAI",
    google: "Google Gemini",
    "meta-ai": "Meta AI",
    meta: "Meta AI",
    muse: "Muse Code",
    openrouter: "OpenRouter",
    "github-copilot": "GitHub Copilot",
    opencode: "OpenCode",
    zai: "Z.ai",
    "kimi-coding": "Kimi (Moonshot)",
    "llama.cpp": "llama.cpp",
  };
  for (const [id, name] of Object.entries(names)) assert.equal(providerName(id), name, id);
  // An id we have no name for stays itself, and an empty id names nothing.
  assert.equal(providerName("some-new-gateway"), "some-new-gateway");
  // The CLI's own name answers where the vocabulary is silent, never over it.
  assert.equal(providerName("sakana", "Sakana AI"), "Sakana AI");
  assert.equal(providerName("anthropic", "Anthropic (Claude Pro/Max)"), "Anthropic");
  assert.equal(providerName("some-new-gateway", "  "), "some-new-gateway");
  assert.equal(providerName(""), "");
  assert.equal(providerName(null), "");
});

test("a kind chip names the two shapes a row can hold", () => {
  assert.equal(kindLabel("oauth"), "Subscription");
  assert.equal(kindLabel("api_key"), "API key");
  // A shape we do not know keeps its own name instead of being filed wrongly.
  assert.equal(kindLabel("token"), "token");
  assert.equal(kindLabel(""), "");
  assert.equal(kindLabel(undefined), "");
});

test("a source chip names where the row came from, and says nothing when unknown", () => {
  assert.equal(sourceLabel("vault"), "Vault");
  assert.equal(sourceLabel("migrated"), "Migrated");
  assert.equal(sourceLabel("imported:codex"), "From Codex");
  assert.equal(sourceLabel("imported:claude"), "From Claude Code");
  // A CLI we have no label for is named by its own id, never "Terminal".
  assert.equal(sourceLabel("imported:whatever"), "From whatever");
  assert.equal(sourceLabel("imported:"), "From a CLI");
  assert.equal(sourceLabel(""), "");
  assert.equal(sourceLabel(undefined), "");
});

test("a health chip is the outcome, its tone and the vendor's own line", () => {
  const at = "2026-09-20T12:00:00Z";
  const now = Date.parse(at) + 4 * 60 * 1000;
  const cases = [
    ["ok", "Works", "ok"],
    ["invalid", "Key refused", "danger"],
    ["expired", "Expired", "danger"],
    ["no_credit", "No credit", "danger"],
    ["rate_limited", "Rate limited", "warn"],
    ["unknown", "Unverified", "muted"],
  ];
  for (const [state, label, tone] of cases) {
    const chip = healthChip({ state, at }, now);
    assert.equal(chip.label, label, state);
    assert.equal(chip.tone, tone, state);
    assert.equal(chip.age, "4m", state);
  }
  const refused = healthChip({ state: "invalid", at, message: "The provider refused this key (401)." }, now);
  assert.equal(refused.message, "The provider refused this key (401).");
  // No record at all is no chip — not an "unknown" one.
  assert.equal(healthChip(null), null);
  assert.equal(healthChip({}), null);
  // A state the server invents later is shown as itself, muted, and an
  // unparseable timestamp costs the age, not the chip.
  assert.deepEqual(healthChip({ state: "throttled", at: "not a date" }), {
    label: "throttled", tone: "muted", state: "throttled", message: "", age: "",
  });
});

test("rows in play come first, then by the name the person gave them", () => {
  const rows = [
    { id: "c", label: "Work", paused: true },
    { id: "a", label: "work", paused: false },
    { id: "b", label: "Personal", paused: false },
    { id: "d", label: "Personal", paused: false },
  ];
  // Equal names are broken by id, so two rows never swap places between renders.
  assert.deepEqual(orderAccounts(rows).map((r) => r.id), ["b", "d", "a", "c"]);
  // The vault's `active` slot is pi's alone; it never reorders a guest CLI.
  const active = [{ id: "1", label: "Zed", active: true }, { id: "2", label: "Alpha", active: false }];
  assert.deepEqual(orderAccounts(active).map((r) => r.id), ["2", "1"]);
  assert.deepEqual(orderAccounts(null), []);
  const input = [{ id: "x", label: "B" }, { id: "y", label: "A" }];
  orderAccounts(input);
  assert.deepEqual(input.map((r) => r.id), ["x", "y"], "the caller's array is not reordered in place");
  assert.equal(VERIFY_LABEL, "Verify with the provider (1 request)");
});

test("vaultProblemText names the file and the action", () => {
  const missing = vaultProblemText("credentials: the vault key is missing — secrets are unavailable on this machine");
  assert.match(missing, /credentials\.key/);
  assert.match(missing, /Put it back/);
  const corrupt = vaultProblemText("credentials: the vault cannot be read — restore a backup or the key file that matches it");
  assert.match(corrupt, /does not open it/);
  assert.equal(vaultProblemText("something else"), "This machine’s vault can’t be read. something else");
});
