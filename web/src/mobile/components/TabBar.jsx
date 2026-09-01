import { IconHome, IconInbox, IconFolders, IconMore } from "../../components/Icons.jsx";
import { goTab } from "../hooks/useHashRoute.js";

const TABS = [
  { id: "now", label: "Now", Icon: IconHome },
  { id: "inbox", label: "Inbox", Icon: IconInbox },
  { id: "work", label: "Work", Icon: IconFolders },
  { id: "more", label: "More", Icon: IconMore },
];

// Four tabs, icon + label, 44px targets, in the thumb zone. A badge is a
// count of decisions waiting (Now) or blocking inbox items (Inbox) —
// never a dot for "something changed", that is what the row itself says.
export default function TabBar({ active, badges }) {
  return (
    <nav className="m-nav" aria-label="Sections">
      {TABS.map(({ id, label, Icon }) => {
        const n = badges && badges[id];
        return (
          <button
            key={id}
            type="button"
            className={"m-tab" + (active === id ? " on" : "")}
            aria-current={active === id ? "page" : undefined}
            onClick={() => goTab(id)}
          >
            <span className="m-tab-icon">
              <Icon size={20} />
              {n ? <span className="m-badge">{n > 99 ? "99+" : n}</span> : null}
            </span>
            <span className="m-tab-label">{label}</span>
          </button>
        );
      })}
    </nav>
  );
}
