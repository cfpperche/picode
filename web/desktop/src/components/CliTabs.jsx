import { useEffect, useRef } from "react";

export default function CliTabs({ view, packagesHref = "#/clis/packages/pi", hasPackageUpdates = false }) {
  const nav = useRef(null);
  useEffect(() => {
    const reveal = () => {
      const rail = nav.current, selected = rail?.querySelector("[aria-current=page]");
      if (!selected) return;
      const item = selected.getBoundingClientRect(), box = rail.getBoundingClientRect();
      if (item.left < box.left) rail.scrollLeft += item.left - box.left;
      else if (item.right > box.right) rail.scrollLeft += item.right - box.right;
    };
    reveal();
    window.addEventListener("resize", reveal);
    return () => window.removeEventListener("resize", reveal);
  }, [view]);
  return <nav ref={nav} className="cli-tabs" aria-label="Agent CLIs">
    <a href="#/clis" aria-current={!["terminals", "sessions", "settings", "packages", "providers", "messages"].includes(view) ? "page" : undefined}>CLIs</a>
    <a href="#/clis/terminals" aria-current={view === "terminals" ? "page" : undefined}>Terminals</a>
    <a href="#/clis/sessions" aria-current={view === "sessions" ? "page" : undefined}>Sessions</a>
    <a href="#/clis/settings/pi" aria-current={view === "settings" ? "page" : undefined}>Settings</a>
    <a href="#/clis/providers/pi" aria-current={view === "providers" ? "page" : undefined}>Providers</a>
    <a href={packagesHref} aria-current={view === "packages" ? "page" : undefined}>Packages{hasPackageUpdates ? <span aria-label="Package updates available"> •</span> : null}</a>
    <a href="#/clis/messages" aria-current={view === "messages" ? "page" : undefined}>Messages</a>
  </nav>;
}
