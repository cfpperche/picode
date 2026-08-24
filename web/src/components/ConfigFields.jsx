export default function ConfigFields({ catalog, provider, model, thinking, onChange, idPrefix }) {
  const providers = catalog && catalog.providers ? catalog.providers : [];
  const current = providers.find((p) => p.id === provider);
  const models = current ? current.models : [];
  const levels = (catalog && catalog.thinking) || ["off", "minimal", "low", "medium", "high", "xhigh", "max"];
  const pfx = idPrefix || "cfg";

  return (
    <div className="cfg-fields">
      <select id={pfx + "-provider"} value={provider} onChange={(e) => onChange({ provider: e.target.value, model: "", thinking })}>
        <option value="">Inherit</option>
        {providers.map((p) => (
          <option key={p.id} value={p.id}>{p.id}{p.signedIn ? "" : ""}</option>
        ))}
      </select>
      <select id={pfx + "-model"} value={model} onChange={(e) => onChange({ provider, model: e.target.value, thinking })} disabled={!provider}>
        <option value="">Inherit</option>
        {models.map((m) => <option key={m.id} value={m.id}>{m.id}</option>)}
      </select>
      <select id={pfx + "-thinking"} value={thinking} onChange={(e) => onChange({ provider, model, thinking: e.target.value })}>
        <option value="">Inherit</option>
        {levels.map((l) => <option key={l} value={l}>{l}</option>)}
      </select>
    </div>
  );
}
