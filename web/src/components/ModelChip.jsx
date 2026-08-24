import SearchCombo from "./SearchCombo.jsx";
import { modelChipLabel, modelChoices } from "../lib/chip.js";

export default function ModelChip({ catalog, cfg, onChange }) {
  const provider = (cfg && cfg.provider) || "";
  const model = (cfg && cfg.model) || "";
  const thinking = (cfg && cfg.thinking) || "";
  const options = modelChoices(catalog, provider);
  const levels = (catalog && catalog.thinking) || ["off", "minimal", "low", "medium", "high", "xhigh", "max"];
  const selected = options.find((m) => m.id === model);
  const canThink = !!(selected && selected.thinking);

  return (
    <>
      <button type="button" id="agent-model" className="sr-only" tabIndex={-1} onFocus={() => document.getElementById("agent-model-btn")?.click()}>model</button>
      <button type="button" id="agent-thinking" className="sr-only" tabIndex={-1} onFocus={() => document.getElementById("agent-model-btn")?.click()}>thinking</button>
      <SearchCombo
        id="agent-model-btn"
        value={model}
        onChange={(id) => onChange({ provider, model: id, thinking })}
        options={options}
        label={modelChipLabel(cfg)}
        searchPlaceholder="Search models"
        disabled={!provider}
        footer={canThink ? (
          <div className="cockpit-think">
            <button
              type="button"
              className={"think-pill" + (!thinking ? " on" : "")}
              onClick={() => onChange({ provider, model, thinking: "" })}
            >default</button>
            {levels.map((l) => (
              <button
                key={l}
                type="button"
                className={"think-pill" + (thinking === l ? " on" : "")}
                onClick={() => onChange({ provider, model, thinking: l })}
              >{l}</button>
            ))}
          </div>
        ) : null}
      />
    </>
  );
}
