import AppSurface from "../../components/AppSurface.jsx";

// The Inbox app (ADR-0037) rendered by the same host surface as the
// desktop; below 880px it already stacks list over detail. The route's
// item id opens straight into that item so a notification tap lands on
// the decision, not the list.
export default function Inbox({ manifest, itemId }) {
  return (
    <div className="m-screen m-inbox">
      <AppSurface appId="inbox" manifest={manifest || { id: "inbox", name: "Inbox", icon: "inbox", apiVersion: 1 }} hidden={false} initialPath={itemId ? "item/" + itemId : ""} />
    </div>
  );
}
