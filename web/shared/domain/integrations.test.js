import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { integrationSection, destinationLabel, readConnectorDefinition, connectorTabs, cliConnectorsHash, cliConnectorsLocation, supportsCliConnectors, connectorDriver, CLI_CONNECTORS } from "./integrations.js";
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
  assert.equal(supportsCliConnectors("grok"), false);
  assert.equal(connectorDriver("pi").name, "Pi");
  assert.equal(connectorDriver("pi").status, "live");
  assert.equal(connectorDriver("grok"), null);
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
    { id: "claude-code", name: "Claude Code", status: "configured", auth: ["oauth"], toggle: "none", signIn: { command: "claude mcp login {name}" } },
    { id: "codex", name: "Codex", status: "configured", auth: ["oauth", "bearer"], toggle: "entry", signIn: { command: "codex mcp login {name}" } },
    { id: "omp", name: "Omp", status: "configured", auth: ["oauth"], toggle: "entry", signIn: { command: "/mcp reauth {name}", where: "the Omp TUI" } },
    { id: "agy", name: "Antigravity", status: "configured", auth: ["oauth"], toggle: "entry", signIn: { text: "Authenticate in Antigravity (Agent Settings → Authenticate)" } },
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

test("connector tabs keep the catalog fixed and list only hosts with servers", () => {
  const found = [
    { kind: "vscode", label: "VS Code", servers: [] },
    { kind: "codex", label: "Codex", servers: [{ name: "ctx", on: false }] },
    { kind: "claude-code", label: "Claude Code", servers: [{ name: "a", on: true }, { name: "b", on: false }] },
  ];
  const tabs = connectorTabs(found);
  assert.deepEqual(tabs.map((t) => t.id), ["catalog", "claude-code", "codex"]);
  assert.deepEqual(tabs.map((t) => t.label), ["Catalog", "Claude Code", "Codex"]);
  assert.equal(tabs[0].fixed, true);
  assert.deepEqual(connectorTabs([]).map((t) => t.id), ["catalog"]);
  assert.deepEqual(connectorTabs().map((t) => t.id), ["catalog"]);
});

test("webhook form validates and normalizes", () => {
  assert.deepEqual(webhookSchema.parse({url:"https://example.com",types:" Agent.,inbox.,agent. "}).types,["agent.","inbox."]);
  for (const url of ["", "https://", "file:///etc/passwd", "https://u:p@example.com", "https://example.com/#x"]) assert.equal(webhookSchema.safeParse({url,types:"agent."}).success,false);
  for (const types of ["", "*", "webhook.","Agent. whatever"]) assert.equal(webhookSchema.safeParse({url:"https://example.com",types}).success,false);
});

test("external definition imports without a provider-specific branch", () => {
  const deepwiki = readFileSync(new URL("../../../connectors/deepwiki.json", import.meta.url), "utf8");
  assert.equal(readConnectorDefinition(deepwiki).url,"https://mcp.deepwiki.com/mcp");
  const gmail = readConnectorDefinition(readFileSync(new URL("../../../connectors/gmail.json", import.meta.url), "utf8"));
  assert.deepEqual(gmail,{name:"gmail",command:"npx",args:["-y","@gongrzhe/server-gmail-autoauth-mcp"]});
  const arbitrary = {mcpServers:{"custom-vendor":{command:"node",args:["/opt/my connector/server.js","--readonly"],env:{API_TOKEN:"example"}}}};
  assert.deepEqual(readConnectorDefinition(JSON.stringify(arbitrary)),{name:"custom-vendor",...arbitrary.mcpServers["custom-vendor"]});
  const remote = {mcpServers:{vendor:{url:"https://example.com/mcp",auth:"bearer",bearerToken:"example",headers:{"X-Read-Only":"true"}}}};
  assert.deepEqual(readConnectorDefinition(JSON.stringify(remote)),{name:"vendor",...remote.mcpServers.vendor});
});

test("definition import decision table refuses ambiguity or unsupported options", () => {
  for (const value of [null, {}, {mcpServers:{a:{url:"https://example.com"},b:{url:"https://example.com"}}},
    {mcpServers:{a:{url:"file:///private"}}}, {mcpServers:{a:{url:"https://example.com",command:"node"}}},
    {mcpServers:{a:{command:"node",args:"--flag"}}}, {mcpServers:{a:{command:"node",cwd:"/private"}}},
    {mcpServers:{a:{url:"https://example.com",headers:{x:42}}}}, {mcpServers:{a:{url:"https://example.com",disabled:true}}},
    {mcpServers:{a:{command:"node",auth:"bearer"}}}, {mcpServers:{a:{url:"https://example.com",env:{TOKEN:"x"}}}},
    {mcpServers:{a:{url:"https://example.com",headers:{Authorization:"!secret-command"}}}},
    {mcpServers:{a:{url:"https://example.com",unknown:"x"}}}, {mcpServers:{a:{url:"https://example.com",auth:"magic"}}}
  ]) assert.throws(() => readConnectorDefinition(JSON.stringify(value)));
  assert.throws(() => readConnectorDefinition("x".repeat(65537)));
  assert.throws(() => readConnectorDefinition("not json"));
});
