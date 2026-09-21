import { Suspense, lazy } from "react";
import * as Sheet from "./MobileSheet.jsx";
import ScreenLoading from "./ScreenBoundary.jsx";

const Inspector = lazy(() => import("../screens/Inspector.jsx"));

// The agent screen's Inspector, as the phone's standard sheet — it rises
// from the bottom over the conversation at the usual sheet height (the
// phone's answer to the desktop rail toggle). Closing it — Back in its
// header, a tap on the overlay, or the drag handle — returns to the agent
// exactly as it was: nothing underneath remounts.
export default function InspectorDrawer({ drawer, onClose, ...pass }) {
  return <Sheet.Root open={!!drawer} onOpenChange={(open) => { if (!open) onClose(); }}>
    <Sheet.Portal>
      <Sheet.Overlay className="dlg-overlay" />
      <Sheet.Content className="dlg m-insp-sheet" aria-describedby={undefined}>
        <Sheet.Title className="sr-only">Inspector</Sheet.Title>
        <Suspense fallback={<ScreenLoading />}>
          {drawer ? <Inspector
            key={drawer.owner.id + ":" + (drawer.root || "")}
            owner={drawer.owner} title={drawer.title || "Inspector"} root={drawer.root || ""}
            onBack={onClose} {...pass} /> : null}
        </Suspense>
      </Sheet.Content>
    </Sheet.Portal>
  </Sheet.Root>;
}
