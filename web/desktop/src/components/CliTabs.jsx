import { useEffect, useRef } from "react";

export default function CliTabs({ view }) {
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
    <a href="#/clis" aria-current={view !== "messages" ? "page" : undefined}>CLIs</a>
    <a href="#/clis/messages" aria-current={view === "messages" ? "page" : undefined}>Messages</a>
  </nav>;
}
