import ScreenHeader from "../components/ScreenHeader.jsx";
import UncommittedDetail from "../../components/UncommittedDetail.jsx";

// Read-only working-tree changes for one owner (ADR-0044 phase 3): the
// git graph's own Uncommitted Changes pane, given the whole screen. A
// file expands into its patch; nothing here edits, stages or commits —
// that stays a workstation act (Cursor's "not a mobile IDE").
export default function Changes({ kind, id, title, onBack }) {
  return (
    <div className="m-screen m-changes">
      <ScreenHeader title="Changes" sub={title} onBack={onBack} />
      <UncommittedDetail owner={{ kind, id }} onClose={onBack} />
    </div>
  );
}
