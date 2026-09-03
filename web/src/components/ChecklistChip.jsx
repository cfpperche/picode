import SearchCombo from "./SearchCombo.jsx";
import { IconMode } from "./Icons.jsx";
import { checklistChoices, checklistLevelLabel } from "../lib/checklist.js";

// The checklist obligation (ADR-0055), read by pi-checklist when it is
// installed: before changes (default), always, never. A read-only agent
// cannot change anything, so it is never asked — the chip says so.
export default function ChecklistChip({ level, readonly, onChange }) {
  if (readonly) {
    return <span className="set-note" title="A read-only agent cannot change anything, so no plan is required">Never (read-only)</span>;
  }
  return (
    <SearchCombo
      id="agent-checklist"
      value={level || "changes"}
      onChange={(id) => onChange({ checklist: id })}
      options={checklistChoices()}
      label={checklistLevelLabel(level)}
      searchPlaceholder="Search levels"
      icon={<IconMode />}
    />
  );
}
