import { scopeKind } from "@picode/shared/domain/scopeIcon.js";
import { IconAgent, IconFolder, IconGlobe } from "./Icons.jsx";

// The mark before a scope chip's label: globe for this computer, folder for
// a workspace, the agent glyph for one agent (shared/domain/scopeIcon.js).
export default function ScopeIcon({ scope }) {
  const kind = scopeKind(scope);
  const Icon = kind === "global" ? IconGlobe : kind === "workspace" ? IconFolder : kind === "agent" ? IconAgent : null;
  return Icon ? <Icon size={13} className="pkg-scope-icon" /> : null;
}
