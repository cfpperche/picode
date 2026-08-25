export const FACE_MAX = 5;

const HOST = {
  anthropic: "anthropic.com",
  openai: "openai.com",
  "openai-codex": "openai.com",
  xai: "x.ai",
  google: "gemini.google.com",
  gemini: "gemini.google.com",
  "google-gemini": "gemini.google.com",
  groq: "groq.com",
  openrouter: "openrouter.ai",
  mistral: "mistral.ai",
  deepseek: "deepseek.com",
  ollama: "ollama.com",
  cohere: "cohere.com",
  together: "together.ai",
  fireworks: "fireworks.ai",
  perplexity: "perplexity.ai",
  huggingface: "huggingface.co",
  "amazon-bedrock": "aws.amazon.com",
  bedrock: "aws.amazon.com",
  "github-copilot": "github.com",
  copilot: "github.com",
};

export function providerId(agent) {
  return String((agent && agent.provider) || "").toLowerCase();
}

export function providerHost(id) {
  return HOST[String(id || "").toLowerCase()] || "";
}

export function providerFaviconUrl(id) {
  const host = providerHost(id);
  if (!host) return "";
  return "https://www.google.com/s2/favicons?domain=" + encodeURIComponent(host) + "&sz=32";
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
