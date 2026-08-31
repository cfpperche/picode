import SearchCombo from "./SearchCombo.jsx";
import { IconLock, IconMode } from "./Icons.jsx";
import { shortModel } from "../lib/chip.js";

/**
 * Active pi-roles state in the composer (ADR-0033 amendment #2). Rendered
 * only when the extension published a state file — no package, no chip.
 * Picking an entry sends the matching command through the normal task path;
 * "Edit roles…" opens the in-thread stepper.
 */
export default function RoleChip({ state, onCommand }) {
  if (!state || !onCommand) return null;
  const lock = state.mode === "lock";
  const modelId = lock && state.model ? state.model.split("/").pop() : "";
  const label = lock ? state.role + (modelId ? " · " + shortModel(modelId) : "") : "auto";

  const options = [{ id: "auto", label: "auto", hint: "route by content" }];
  for (const r of state.roles || []) {
    if (!r || !r.name || r.name === "auto") continue;
    options.push({
      id: "role:" + r.name,
      label: r.name,
      hint: r.model ? r.model + (r.thinking ? " · " + r.thinking : "") : "",
    });
  }
  options.push({ id: "edit", label: "Edit roles…", hint: "" });

  function pick(id) {
    if (id === "edit") return onCommand("/roles edit");
    if (id === "auto") return onCommand("/auto");
    if (id.startsWith("role:")) return onCommand("/role " + id.slice(5));
  }

  return (
    <SearchCombo
      id="agent-role"
      value={lock ? "role:" + state.role : "auto"}
      onChange={pick}
      options={options}
      label={label}
      searchPlaceholder="Search roles"
      icon={lock ? <IconLock /> : <IconMode />}
      triggerClassName={lock ? "cockpit-chip role-chip role-chip-lock" : "cockpit-chip role-chip"}
    />
  );
}
