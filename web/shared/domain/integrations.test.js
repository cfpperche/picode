import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { integrationSection, destinationLabel, readConnectorDefinition } from "./integrations.js";
import { webhookSchema } from "../contracts/schemas.js";

test("integration routes and safe destination labels", () => {
  assert.equal(integrationSection("#/integrations"), "connectors");
  assert.equal(integrationSection("#/integrations/webhooks"), "webhooks");
  assert.equal(integrationSection("#/more/integrations/webhooks"), "webhooks");
  assert.equal(destinationLabel("https://example.com/hook?token=secret"), "example.com/hook");
});

test("webhook form validates and normalizes", () => {
  assert.deepEqual(webhookSchema.parse({url:"https://example.com",types:" Agent.,inbox.,agent. "}).types,["agent.","inbox."]);
  for (const url of ["", "https://", "file:///etc/passwd", "https://u:p@example.com", "https://example.com/#x"]) assert.equal(webhookSchema.safeParse({url,types:"agent."}).success,false);
  for (const types of ["", "*", "webhook.","Agent. whatever"]) assert.equal(webhookSchema.safeParse({url:"https://example.com",types}).success,false);
});

test("external definition imports without a provider-specific branch", () => {
  const deepwiki = readFileSync(new URL("../../../connectors/deepwiki.json", import.meta.url), "utf8");
  assert.equal(readConnectorDefinition(deepwiki).url,"https://mcp.deepwiki.com/mcp");
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
