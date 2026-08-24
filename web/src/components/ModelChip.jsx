import SearchCombo from "./SearchCombo.jsx";
import { modelChipLabel, modelChoices } from "../lib/chip.js";

export default function ModelChip({ catalog, cfg, onChange }) {
  const provider = (cfg && cfg.provider) || "";
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";
  const options = modelChoices(catalog, provider);

  function pick(id) {
    const next = options.find((m) => m.id === id);
    const levels = (next && next.thinkingLevels) || [];
    const keep = thinking && levels.includes(thinking) ? thinking : "";
    onChange({ provider, model: id, thinking: keep });
  }

  return (
    <>
      <button type="button" id="agent-model" className="sr-only" tabIndex={-1} onFocus={() => document.getElementById("agent-model-btn")?.click()}>model</button>
      <SearchCombo
        id="agent-model-btn"
        value={model}
        onChange={pick}
        options={options}
        label={modelChipLabel(cfg)}
        searchPlaceholder="Search models"
        disabled={!provider}
      />
    </>
  );
}
