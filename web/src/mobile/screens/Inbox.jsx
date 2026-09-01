import AppSurface from "../../components/AppSurface.jsx";
import ScreenHeader from "../components/ScreenHeader.jsx";
import PullScreen from "../components/PullScreen.jsx";
import { useState } from "react";

const MANIFEST = { id: "inbox", name: "Inbox", icon: "inbox", apiVersion: 1 };

// The Inbox app (ADR-0037) as two phone screens: the tab is the list
// (tabs, search, rows — pull to refresh), an item is a pushed screen with
// the shell's Back header. The desktop keeps its split.
export default function Inbox({ manifest, onOpenItem }) {
  const [tick, setTick] = useState(0);
  return (
    <PullScreen className="m-inbox" onRefresh={() => { setTick((t) => t + 1); return new Promise((r) => setTimeout(r, 300)); }}>
      <AppSurface appId="inbox" manifest={manifest || MANIFEST} hidden={false} refreshKey={tick} paneMode="list" onOpenItem={(p) => onOpenItem(p.replace(/^item\//, ""))} />
    </PullScreen>
  );
}

export function InboxItem({ manifest, itemId, onBack }) {
  return (
    <div className="m-screen m-inbox-item">
      <ScreenHeader title="Inbox" onBack={onBack} />
      <AppSurface appId="inbox" manifest={manifest || MANIFEST} hidden={false} initialPath={"item/" + itemId} paneMode="detail" onClose={onBack} />
    </div>
  );
}
