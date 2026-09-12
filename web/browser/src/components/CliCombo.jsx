import SearchCombo from "./SearchCombo.jsx";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";

function cliOption(option) {
  const id = typeof option === "string" ? option : option.id;
  const name = (typeof option === "string" ? "" : option.name) || terminalCliLabel(id);
  return { id, label: name, icon: <TerminalCliBadge term={{ cli: id }} decorative /> };
}

export default function CliCombo({
  id,
  value,
  onChange,
  options = [],
  disabled,
  ariaLabel = "CLI",
  side = "bottom",
  align = "start",
}) {
  const items = (options || []).map(cliOption);
  if (value && !items.some((item) => item.id === value)) items.unshift(cliOption(value));
  const selected = items.find((item) => item.id === value) || items[0];
  return (
    <SearchCombo
      id={id}
      value={selected ? selected.id : value}
      onChange={onChange}
      options={items}
      label={selected ? selected.label : "CLI"}
      searchPlaceholder={items.length > 4 ? "Find a CLI…" : false}
      icon={selected ? selected.icon : null}
      disabled={disabled || !items.length}
      triggerClassName="cli-combo"
      side={side}
      align={align}
      ariaLabel={selected ? ariaLabel + ": " + selected.label : ariaLabel}
    />
  );
}
