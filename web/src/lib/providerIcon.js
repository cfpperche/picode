export const FACE_MAX = 5;

const ICON = {
  anthropic: "claude",
  openai: "openai",
  "openai-codex": "openai",
  xai: "grok",
  google: "gemini",
  gemini: "gemini",
  "google-gemini": "gemini",
  groq: "groq",
  openrouter: "openrouter",
  mistral: "mistral",
  deepseek: "deepseek",
  ollama: "ollama",
  cohere: "cohere",
  together: "together",
  fireworks: "fireworks",
  perplexity: "perplexity",
  huggingface: "huggingface",
  "amazon-bedrock": "bedrock",
  bedrock: "bedrock",
  "github-copilot": "githubcopilot",
  copilot: "githubcopilot",
};

const ICON_BASE = "https://unpkg.com/@lobehub/icons-static-svg@1.73.0/icons/";

export function providerId(agent) {
  return String((agent && agent.provider) || "").toLowerCase();
}

export function providerFaviconUrl(id) {
  const name = ICON[String(id || "").toLowerCase()];
  if (!name) return "";
  return ICON_BASE + name + ".svg";
}

export function providerLetter(id) {
  const s = String(id || "").trim();
  return s ? s[0].toUpperCase() : "?";
}

export function faceSlice(agents) {
  const list = agents || [];
  return { shown: list.slice(0, FACE_MAX), extra: Math.max(0, list.length - FACE_MAX) };
}

export function workspaceAgents(workspaces) {
  const out = [];
  for (const ws of workspaces || []) {
    const list = (ws && ws.agents && ws.agents.length) ? ws.agents : (ws && ws.agent ? [ws.agent] : []);
    for (const a of list) out.push(a);
  }
  return out;
}
