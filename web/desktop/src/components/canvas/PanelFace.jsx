import { ProviderFace } from "../ProviderFaces.jsx";
import TerminalCliBadge from "../TerminalCliBadge.jsx";
import { IconFile, IconGit, IconPin } from "../Icons.jsx";

// PanelFace — the small mark that says what a panel is bound to, in the
// header and on the name-plate (docs/plans/matrix-canvas.md §4.2). One place
// per kind: an agent wears its provider's face, a terminal its CLI badge, a
// note the pin mark the sidebar and Pin Studio already use, a file the same
// file mark the tree and the tabs use, a diff the git mark.
export default function PanelFace({ model, size = 16 }) {
  if (model.kind === "note") return <IconPin size={size - 2} />;
  if (model.kind === "file") return <IconFile size={size - 2} />;
  if (model.kind === "diff") return <IconGit size={size - 2} />;
  if (model.kind === "agent") return model.target ? <ProviderFace agent={model.target} /> : <span className="ws-face">?</span>;
  return <TerminalCliBadge term={model.target || {}} decorative />;
}
