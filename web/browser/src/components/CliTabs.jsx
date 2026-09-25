import OverflowTabs from "./OverflowTabs.jsx";

const TABS = [
  { id: "clis", label: "CLIs", href: "#/clis" },
  { id: "messages", label: "Messages", href: "#/clis/messages" },
  { id: "settings", label: "Settings", href: "#/clis/settings" },
];

// Below 768px the three tabs can outgrow a narrow pane; OverflowTabs gives
// the strip the editor strip's arrows and list instead of a scrollbar.
export default function CliTabs({ view }) {
  const current = view === "messages" || view === "settings" ? view : "clis";
  return <OverflowTabs className="cli-tabs" frameClassName="cli-tabs-frame" label="Agent CLIs" items={TABS} selectedId={current}>
    {TABS.map((t) => <a key={t.id} data-tab={t.id} href={t.href} aria-current={t.id === current ? "page" : undefined}>{t.label}</a>)}
  </OverflowTabs>;
}
