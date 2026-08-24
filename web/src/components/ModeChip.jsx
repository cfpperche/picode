import SearchCombo from "./SearchCombo.jsx";
import { modeChipLabel, modeChoices } from "../lib/chip.js";

export default function ModeChip({ cfg, onChange }) {
  const mode = (cfg && cfg.opMode) || "full";
  return (
    <SearchCombo
      id="agent-mode"
      value={mode === "readonly" ? "readonly" : "full"}
      onChange={(id) => onChange({ opMode: id })}
      options={modeChoices()}
      label={modeChipLabel(cfg)}
      searchPlaceholder="Search modes"
    />
  );
}
