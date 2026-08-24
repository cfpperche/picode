import SearchCombo from "./SearchCombo.jsx";
import { thinkingChipLabel, thinkingChoices } from "../lib/chip.js";

export default function ThinkingChip({ catalog, cfg, onChange }) {
  const provider = (cfg && cfg.provider) || "";
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";
  const options = thinkingChoices(catalog, provider, model);
  const canThink = options.length > 1;

  return (
    <SearchCombo
      id="agent-thinking"
      value={thinking}
      onChange={(id) => onChange({ provider, model, thinking: id })}
      options={options}
      label={thinkingChipLabel(cfg)}
      searchPlaceholder="Search thinking"
      disabled={!canThink}
    />
  );
}
