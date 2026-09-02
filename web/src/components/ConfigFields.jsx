import { usableProviders } from "../lib/providers.js";

export default function ConfigFields({ catalog, provider, model, thinking, onChange, idPrefix, row }) {
  const providers = usableProviders(catalog, provider);
  const current = providers.find((p) => p.id === provider);
  const models = current ? current.models : [];
  const levels = (catalog && catalog.thinking) || ["off", "minimal", "low", "medium", "high", "xhigh", "max"];
  const pfx = idPrefix || "cfg";

  return (
    <div className={row ? "cfg-fields cfg-row" : "cfg-fields"}>
      <select id={pfx + "-provider"} value={provider} onChange={(e) => onChange({ provider: e.target.value, model: "", thinking })}>
        <option value="" disabled>Provider</option>
        {provider && !providers.some((p) => p.id === provider) ? <option value={provider}>{provider}</option> : null}
        {providers.map((p) => (
          <option key={p.id} value={p.id}>{p.id}{p.signedIn ? "" : " (not signed in)"}</option>
        ))}
      </select>
      <select id={pfx + "-model"} value={model} onChange={(e) => onChange({ provider, model: e.target.value, thinking })} disabled={!provider}>
        <option value="" disabled>Model</option>
        {model && !models.some((m) => m.id === model) ? <option value={model}>{model}</option> : null}
        {models.map((m) => <option key={m.id} value={m.id}>{m.id}</option>)}
      </select>
      <select id={pfx + "-thinking"} value={thinking} onChange={(e) => onChange({ provider, model, thinking: e.target.value })}>
        <option value="" disabled>Thinking</option>
        {thinking && !levels.includes(thinking) ? <option value={thinking}>{thinking}</option> : null}
        {levels.map((l) => <option key={l} value={l}>{l}</option>)}
      </select>
    </div>
  );
}
