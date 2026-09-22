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
  cerebras: "cerebras",
  nvidia: "nvidia",
  azure: "azure",
  "azure-openai-responses": "azure",
  cloudflare: "cloudflare",
  "cloudflare-ai-gateway": "cloudflare",
  "cloudflare-workers-ai": "cloudflare",
  zai: "zhipu",
  "zai-coding-cn": "zhipu",
  minimax: "minimax",
  // Omp's /login catalog (ADR-0183), matched against the icon set's own
  // names. A provider the set has no mark for keeps the letter plate: no
  // guessed look-alikes.
  "bedrock-mantle": "bedrock",
  "google-gemini-cli": "gemini",
  "google-vertex": "vertexai",
  "kimi-coding": "kimi",
  "kimi-code": "kimi",
  moonshot: "moonshot",
  "qwen-portal": "qwen",
  "alibaba-coding-plan": "alibaba",
  "alibaba-token-plan": "bailian",
  "zai-coding-plan": "zhipu",
  "zhipu-coding-plan": "zhipu",
  "minimax-code": "minimax",
  "minimax-code-cn": "minimax",
  "minimax-cn": "minimax",
  baseten: "baseten",
  deepinfra: "deepinfra",
  novita: "novita",
  siliconflow: "siliconcloud",
  "siliconflow-cn": "siliconcloud",
  "vercel-ai-gateway": "vercel",
  "lm-studio": "lmstudio",
  vllm: "vllm",
  cursor: "cursor",
  exa: "exa",
  tavily: "tavily",
  "ollama-cloud": "ollama",
  "xai-oauth": "grok",
  meta: "meta",
  "meta-ai": "meta",
  "cline-pass": "cline",
  qianfan: "baiducloud",
  firepass: "fireworks",
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
    for (const a of list) if (a && a.id) out.push(a);
  }
  return out;
}
