import test from "node:test";
import assert from "node:assert/strict";
import { integrationSection, destinationLabel, cliConnectorsHash, cliConnectorsLocation, supportsCliConnectors, connectorDriver, blockedLayers, CLI_CONNECTORS, connectorAddBody, connectorScopeNames, connectorDocsUrl } from "./integrations.js";
import { webhookSchema } from "../contracts/schemas.js";

test("integration routes and safe destination labels", () => {
  assert.equal(integrationSection("#/integrations"), "connectors");
  assert.equal(integrationSection("#/integrations/webhooks"), "webhooks");
  assert.equal(integrationSection("#/more/integrations/webhooks"), "webhooks");
  assert.equal(destinationLabel("https://example.com/hook?token=secret"), "example.com/hook");
});

test("connectors nest on the selected CLI; webhooks stay platform", () => {
  assert.equal(supportsCliConnectors("pi"), true);
  assert.equal(supportsCliConnectors("claude-code"), true);
  assert.equal(supportsCliConnectors("claude"), false);
  assert.equal(supportsCliConnectors("codex"), true);
  assert.equal(supportsCliConnectors("opencode"), true);
  assert.equal(supportsCliConnectors("grok"), true);
  assert.equal(supportsCliConnectors("muse"), true);
  assert.equal(supportsCliConnectors("hermes"), true);
  assert.equal(supportsCliConnectors("claude"), false);
  assert.equal(connectorDriver("pi").name, "Pi");
  assert.equal(connectorDriver("pi").status, "live");
  assert.equal(connectorDriver("opencode").name, "OpenCode");
  assert.equal(connectorDriver("hermes").name, "Hermes Agent");
  assert.equal(connectorDriver(""), null);
  assert.equal(cliConnectorsHash("pi"), "#/clis/pi/connectors");
  assert.equal(cliConnectorsHash("claude-code"), "#/clis/claude-code/connectors");
  assert.equal(cliConnectorsLocation("#/clis/claude-code/connectors").id, "claude-code");
  assert.equal(cliConnectorsLocation("#/integrations").redirect, "#/clis/pi/connectors");
  assert.equal(cliConnectorsLocation("#/integrations/connectors").redirect, "#/clis/pi/connectors");
  assert.equal(cliConnectorsLocation("#/mcps").redirect, "#/clis/pi/connectors");
  assert.equal(cliConnectorsLocation("#/clis/pi/connectors").redirect, "");
  assert.equal(cliConnectorsLocation("#/integrations/webhooks"), null);
  assert.equal(cliConnectorsLocation("#/clis/connectors").redirect, "#/clis/pi/connectors");
  assert.equal(cliConnectorsLocation("#/clis/connectors").id, "pi");
  assert.equal(cliConnectorsLocation("#/clis/connectors").adoptPane, undefined);
  assert.equal(cliConnectorsLocation("#/clis/pi/connectors/extra").invalid, true);
  const scoped = cliConnectorsLocation("#/clis/pi/connectors?workspaceId=w&agentId=a");
  assert.equal(scoped.agentId, "a");
  assert.equal(scoped.workspaceId, "w");
  assert.equal(scoped.redirect, "");
  const adopted = cliConnectorsLocation("#/mcps", { workspaceId: "w", agentId: "a" });
  assert.equal(adopted.adoptPane, true);
  assert.equal(adopted.redirect, "#/clis/pi/connectors?workspaceId=w&agentId=a");
});

// ADR-0150: each driver declares its honest capability set — the pane shows
// exactly this and nothing more.
test("every connector driver declares its capability set", () => {
  assert.deepEqual(CLI_CONNECTORS, [
    { id: "pi", name: "Pi", status: "live", auth: ["oauth", "bearer"], toggle: "entry" },
    { id: "claude-code", name: "Claude Code", status: "live", auth: ["oauth"], toggle: "none", signIn: { command: "claude mcp login {name}" } },
    { id: "codex", name: "Codex", status: "configured", auth: ["oauth", "bearer"], toggle: "entry", signIn: { command: "codex mcp login {name}" } },
    { id: "omp", name: "Omp", status: "configured", auth: ["oauth"], toggle: "entry", signIn: { command: "/mcp reauth {name}", where: "the Omp TUI" } },
    { id: "agy", name: "Antigravity", status: "configured", auth: ["oauth"], toggle: "entry", signIn: { text: "Authenticate in Antigravity (Agent Settings → Authenticate)" } },
    { id: "opencode", name: "OpenCode", status: "live", auth: ["oauth"], toggle: "entry", signIn: { command: "opencode mcp auth {name}" } },
    { id: "grok", name: "Grok", status: "configured", auth: ["oauth"], toggle: "entry", signIn: { text: "Sign in happens on first use inside Grok" } },
    { id: "muse", name: "Muse Code", status: "configured", auth: ["oauth"], toggle: "entry", signIn: { command: "muse mcp login {name}" } },
    { id: "hermes", name: "Hermes Agent", status: "live", auth: ["oauth"], toggle: "entry", signIn: { command: "hermes mcp login {name}" } },
  ]);
  for (const d of CLI_CONNECTORS) {
    assert.ok(["live", "configured"].includes(d.status), d.id + " status");
    assert.ok(["entry", "none"].includes(d.toggle), d.id + " toggle");
    assert.ok(d.auth.length > 0, d.id + " auth");
    if (d.id !== "pi") {
      assert.ok(d.signIn, d.id + " sign-in hint");
      if (d.signIn.command) assert.ok(d.signIn.command.includes("{name}"), d.id + " sign-in command template");
      else assert.ok(d.signIn.text, d.id + " sign-in text");
    }
  }
});

// The claude-code lesson: a driver id must be a CLI catalog id, so deep
// links and the roster resolve. The catalog itself lives server-side
// (internal/clilaunch, covered by TestDriverIDsMatchCatalog in Go); here we
// cross against the shared id normalizer.
test("connector driver ids are canonical catalog ids", async () => {
  const { normalizeTerminalCli } = await import("./terminalCli.js");
  for (const d of CLI_CONNECTORS) assert.equal(normalizeTerminalCli(d.id), d.id, d.id);
});

test("webhook form validates and normalizes", () => {
  assert.deepEqual(webhookSchema.parse({url:"https://example.com",types:" Agent.,inbox.,agent. "}).types,["agent.","inbox."]);
  for (const url of ["", "https://", "file:///etc/passwd", "https://u:p@example.com", "https://example.com/#x"]) assert.equal(webhookSchema.safeParse({url,types:"agent."}).success,false);
  for (const types of ["", "*", "webhook.","Agent. whatever"]) assert.equal(webhookSchema.safeParse({url:"https://example.com",types}).success,false);
});

// A blocked config layer names the file for the one-line pane state
// (ADR-0150): empty when healthy, one entry per errored layer, Windows
// separators included in the basename.
test("blocked layers name the file and stay empty when healthy", () => {
  assert.deepEqual(blockedLayers(null), []);
  assert.deepEqual(blockedLayers({}), []);
  assert.deepEqual(blockedLayers({ layers: [{ id: "opencode-user", path: "/home/u/.config/opencode/opencode.json" }] }), []);
  assert.deepEqual(blockedLayers({ layers: [{ id: "a", error: "" }, { id: "b", error: "x" }] }).map((l) => l.scope), ["user"]);
  assert.deepEqual(
    blockedLayers({
      layers: [
        { id: "opencode-user", path: "/home/u/.config/opencode/opencode.json", exists: true, scope: "user", error: "is not valid JSON" },
        { id: "opencode-project", path: "/home/u/w/opencode.json", exists: true, scope: "project", error: "" },
      ],
      servers: [],
    }),
    [{ scope: "user", error: "is not valid JSON", path: "/home/u/.config/opencode/opencode.json", file: "opencode.json" }],
  );
  assert.deepEqual(
    blockedLayers({ layers: [{ id: "grok-user", path: "C:\\Users\\u\\.grok\\config.toml", exists: true, scope: "user", error: "is not valid TOML" }] }),
    [{ scope: "user", error: "is not valid TOML", path: "C:\\Users\\u\\.grok\\config.toml", file: "config.toml" }],
  );
});

// Marketplace (ADR-0157): a gallery hit builds the same POST /api/mcp entry
// the preset rows built, and "Added" reads only the selected scope's layer.
test("catalog hits build add bodies and Added-checks per scope", () => {
  assert.deepEqual(
    connectorAddBody({ id: "deepwiki", name: "DeepWiki", summary: "Ask questions about public GitHub repositories.", kind: "url", url: "https://mcp.deepwiki.com/mcp", auth: "oauth", featured: true, source: "picode" }),
    { name: "deepwiki", url: "https://mcp.deepwiki.com/mcp", command: "", args: [], auth: "oauth" },
  );
  assert.deepEqual(
    connectorAddBody({ id: "files", name: "Files", kind: "stdio", command: "npx", args: ["-y", "@picode/mcp-files"] }),
    { name: "files", url: "", command: "npx", args: ["-y", "@picode/mcp-files"], auth: "" },
  );
  assert.deepEqual(connectorAddBody(null), { name: "", url: "", command: "", args: [], auth: "" });
  assert.deepEqual(connectorAddBody({ id: "x", kind: "stdio", args: "not-a-list", auth: "oauth" }), { name: "x", url: "", command: "", args: [], auth: "oauth" });
  const servers = [
    { name: "deepwiki", scope: "user" },
    { name: "deepwiki", scope: "project" },
    { name: "other", scope: "agent" },
  ];
  assert.deepEqual([...connectorScopeNames(servers, "user")], ["deepwiki"]);
  assert.deepEqual([...connectorScopeNames(servers, "project")], ["deepwiki"]);
  assert.deepEqual([...connectorScopeNames(servers, "agent")], ["other"]);
  assert.equal(connectorScopeNames(null, "user").size, 0);
  assert.deepEqual([...connectorScopeNames([{ name: "a", scope: "user" }, null], "user")], ["a"]);
});

test("connector docs links are http(s) only", () => {
  assert.equal(connectorDocsUrl({ docsUrl: "https://latlong.ai" }), "https://latlong.ai/");
  assert.equal(connectorDocsUrl({ docsUrl: "http://example.com/docs" }), "http://example.com/docs");
  assert.equal(connectorDocsUrl({ docsUrl: "  https://acme.example/docs  " }), "https://acme.example/docs");
  assert.equal(connectorDocsUrl({ docsUrl: "javascript:alert(1)" }), "");
  assert.equal(connectorDocsUrl({ docsUrl: "" }), "");
  assert.equal(connectorDocsUrl(null), "");
  assert.equal(connectorDocsUrl({}), "");
});
