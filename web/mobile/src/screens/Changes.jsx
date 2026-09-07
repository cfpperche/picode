import ScreenHeader from "../components/ScreenHeader.jsx";
import UncommittedDetail from "../components/UncommittedDetail.jsx";
import "../styles/mobile-tools.css";

// Read-only working-tree changes for one owner (ADRs 0044/0072).
// Expand a file to inspect its patch.
export default function Changes({ kind, id, title, onBack }) {
  return (
    <div className="m-screen m-changes">
      <ScreenHeader title="Changes" sub={title} onBack={onBack} />
      <UncommittedDetail key={`${kind}:${id}`} owner={{ kind, id }} onClose={onBack} />
    </div>
  );
}
