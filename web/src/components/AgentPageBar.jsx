import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconSettings, IconMcp, IconPackage, IconEllipsis } from "./Icons.jsx";

const PAGES = [
  { id: "settings", label: "Settings", Icon: IconSettings },
  { id: "mcps", label: "MCPs", Icon: IconMcp },
  { id: "packages", label: "Packages", Icon: IconPackage },
];

export default function AgentPageBar({ onGo, pkgUpdates, children }) {
  if (!onGo && !children) return null;
  const hasPkgUp = !!(pkgUpdates && pkgUpdates.length);
  return (
    <div className="composer-pages" role="toolbar" aria-label="Agent">
      {children}
      {onGo ? (
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button type="button" className="composer-page" aria-label="More" title="More">
              <IconEllipsis />
              {hasPkgUp ? <span className="um-dot composer-page-dot" aria-label="Updates available" /> : null}
            </button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="composer-more-pop" side="top" align="end" sideOffset={6} collisionPadding={8}>
              {PAGES.map((p) => (
                <DropdownMenu.Item key={p.id} className="composer-more-item" onSelect={() => onGo(p.id)}>
                  <p.Icon />
                  <span>{p.label}</span>
                  {p.id === "packages" && hasPkgUp ? <span className="um-dot" aria-label="Updates available" /> : null}
                </DropdownMenu.Item>
              ))}
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      ) : null}
    </div>
  );
}
