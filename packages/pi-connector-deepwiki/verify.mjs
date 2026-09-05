// Opt-in public-network smoke check; no LLM turn or owner credentials.
// Run after installing this package and pi-mcp-adapter in your user scope.
import { createRequire } from "node:module";
import { homedir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const root = join(homedir(), ".pi", "agent", "npm", "node_modules", "pi-mcp-adapter");
const require = createRequire(join(root, "package.json"));
const { loadMcpConfig } = await import(pathToFileURL(join(root, "dist", "config.js")));
const { Client, StreamableHTTPClientTransport } = await import(pathToFileURL(require.resolve("@modelcontextprotocol/client")));
const config = loadMcpConfig(undefined, process.cwd());
const name = "pi-connector-deepwiki__docs";
const entry = config.mcpServers[name];
if (!entry || entry.disabled || entry.url !== "https://mcp.deepwiki.com/mcp") {
  throw new Error("Install and enable the unmodified DeepWiki connector package first.");
}
const client = new Client({ name: "picode-connector-check", version: "0.1.0" });
try {
  await client.connect(new StreamableHTTPClientTransport(new URL(entry.url)));
  const { tools } = await client.listTools();
  const result = await client.callTool({ name: "read_wiki_structure", arguments: { repoName: "golang/go" } });
  if (result.isError || !result.content?.length) throw new Error("The public repository read failed.");
  console.log(JSON.stringify({ package: name, tools: tools.map(t => t.name), publicRepositoryRead: "passed" }));
} finally {
  await client.close();
}
