import SearchCombo from "./SearchCombo.jsx";
import { IconKind } from "./Icons.jsx";

const KINDS = [
  { id: "prompt", label: "Prompt" },
  { id: "steer", label: "Steer" },
  { id: "follow_up", label: "Follow-up" },
];

export default function KindChip({ value, onChange }) {
  const cur = KINDS.find((k) => k.id === value) || KINDS[0];
  return (
    <SearchCombo
      id="task-kind"
      value={cur.id}
      onChange={onChange}
      options={KINDS}
      label={cur.label}
      searchPlaceholder="Delivery"
      icon={<IconKind />}
    />
  );
}
