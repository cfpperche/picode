import SearchCombo from "./SearchCombo.jsx";
import { IconKind } from "./Icons.jsx";

const KINDS = [
  { id: "prompt", label: "Prompt" },
  { id: "steer", label: "Steer" },
  { id: "follow_up", label: "Follow-up" },
];

// `options` narrows the list to what the target takes (the attach
// composer offers only the modes a working CLI has, ADR-0206).
// `closeFocus` moves focus to the owner's field once the list closes;
// `search={false}` drops the filter field for a list of two.
export default function KindChip({ value, onChange, options = KINDS, id = "task-kind", closeFocus, search = true }) {
  const cur = options.find((k) => k.id === value) || options[0];
  return (
    <SearchCombo
      id={id}
      value={cur.id}
      onChange={onChange}
      options={options}
      label={cur.label}
      searchPlaceholder={search ? "Delivery" : false}
      ariaLabel="Delivery"
      closeFocus={closeFocus}
      icon={<IconKind />}
    />
  );
}
