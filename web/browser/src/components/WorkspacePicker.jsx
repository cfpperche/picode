import SearchCombo from "./SearchCombo.jsx";
import { IconGit } from "./Icons.jsx";

// The git graph's first toolbar item (ADR-0022 amendment): the workspace whose
// folder the history is read through. One pick, closes on select, and wears the
// same `.btn.btn-sm` control height as the branch picker beside it — a second
// widget language in one toolbar row is the thing this is not.
//
// The caller decides when the control exists at all: fewer than two choices
// is not a dropdown but a dead end, so the surface renders its plain title
// (`graph.name`) instead. The guard here keeps that rule true even if a caller
// forgets it.
export default function WorkspacePicker({ options, value, label, onPick, ariaLabel }) {
  if ((options || []).length < 2) return null;
  return (
    <SearchCombo
      ariaLabel={ariaLabel || "Workspace this tab reads"}
      label={label}
      icon={<IconGit size={13} />}
      triggerClassName="btn btn-sm ws-picker-trigger"
      popoverClassName="ws-picker-pop"
      markCurrent
      value={value}
      options={options}
      onChange={(id) => onPick && onPick(id)}
      searchPlaceholder="Find a workspace"
      side="bottom"
      align="start"
    />
  );
}
