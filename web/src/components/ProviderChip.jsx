import SearchCombo from "./SearchCombo.jsx";
import { providerChipLabel, providerChoices } from "../lib/chip.js";

export default function ProviderChip({ catalog, cfg, onChange }) {
  const provider = (cfg && cfg.provider) || "";
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";
  const options = providerChoices(catalog, provider);

  function pick(id) {
    if (id === provider) return;
    const models = ((catalog && catalog.providers) || []).find((p) => p.id === id);
    const keep = models && (models.models || []).some((m) => m.id === model);
    onChange({
      provider: id,
      model: keep ? model : "",
      thinking: keep ? thinking : "",
    });
  }

  return (
    <SearchCombo
      id="agent-provider"
      value={provider}
      onChange={pick}
      options={options}
      label={providerChipLabel(cfg)}
      searchPlaceholder="Search providers"
    />
  );
}
