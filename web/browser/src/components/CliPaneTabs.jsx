import { useEffect, useRef } from "react";
import { cliModelsHash, cliPaneHash } from "@picode/shared/domain/cliLaunch.js";
import { cliSettingsHash, cliSettingsQuery } from "@picode/shared/domain/cliSettings.js";
import { cliPackagesHash } from "@picode/shared/domain/cliPackages.js";
import { cliMemoryHash } from "@picode/shared/domain/cliNative.js";
import { cliConnectorsHash } from "@picode/shared/domain/integrations.js";
import { cliSkillsHash } from "@picode/shared/domain/cliSkills.js";

const RUN = [
  { id: "launch", label: "Launch" },
  { id: "terminals", label: "Terminals" },
  { id: "sessions", label: "Sessions" },
];
const SETUP = [
  { id: "providers", label: "Providers" },
  // What the CLI can reach, and which of it it may use (ADR-0181). Only CLIs
  // PiCode can ask carry it — cliPanes decides.
  { id: "models", label: "Models" },
  { id: "settings", label: "Settings" },
  // "Keyboard", not "Keys": the pane next door configures providers, where
  // "keys" means credentials (owner picked the name, 2026-09-12).
  { id: "keyboard", label: "Keyboard" },
  // What the CLI has remembered between sessions (ADR-0163). Every CLI has
  // the tab; the ones with no native memory answer in one line.
  { id: "memory", label: "Memory" },
  { id: "packages", label: "Packages" },
  // Standalone Agent Skills each CLI loads, and from where (ADR-0196).
  { id: "skills", label: "Skills" },
  { id: "connectors", label: "Connectors" },
];

export function cliSetupHref(cli, pane, ctx = {}, workspace = "") {
  // The layer rides along: this href is written by the pane's "carry the
  // selected agent" rewrite, and dropping the layer there made a layer pill
  // click look like it did nothing (2026-09-12).
  if (pane === "settings") return cliSettingsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", focus: ctx.focus || "", layer: ctx.layer || "" });
  // The keyboard map is machine-wide, but the link keeps the settings context:
  // going there and back must not move the reader to another agent, workspace or layer.
  if (pane === "keyboard") return cliPaneHash(cli, "keyboard") + cliSettingsQuery({ workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", layer: ctx.layer || "" });
  if (pane === "models") return cliModelsHash(cli, { workspaceId: ctx.workspaceId || "", layer: ctx.layer || "" });
  if (pane === "memory") return cliMemoryHash(cli, { workspaceId: ctx.workspaceId || "", scope: ctx.scope === "workspace" || ctx.scope === "global" ? ctx.scope : "" });
  if (pane === "packages") return cliPackagesHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: ctx.scope || "user" });
  if (pane === "skills") return cliSkillsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "" });
  if (pane === "connectors") return cliConnectorsHash(cli, { workspaceId: ctx.workspaceId || "", agentId: ctx.agentId || "", scope: ctx.scope || "user" });
  return cliPaneHash(cli, pane, pane === "sessions" ? workspace : "");
}

// Bring the selected tab fully into the strip without scrolling the page.
// A tab that is only partly in the rail is hidden so a deep link never shows
// a clipped label like "nch" from Launch (visual review, 2026-09-24).
function maskPartialTabs(nav) {
  if (!nav) return;
  const rail = nav.getBoundingClientRect();
  for (const el of nav.querySelectorAll("[role=tab]")) {
    const box = el.getBoundingClientRect();
    const inView = box.right > rail.left + 1 && box.left < rail.right - 1;
    const fully = box.left >= rail.left - 1 && box.right <= rail.right + 1;
    el.style.visibility = inView && !fully ? "hidden" : "";
  }
}

function revealPaneTab(tab) {
  const nav = tab && tab.closest(".cli-pane-tabs");
  if (!nav || !tab) return;
  const rail = nav.getBoundingClientRect();
  const item = tab.getBoundingClientRect();
  if (item.left < rail.left) nav.scrollLeft += item.left - rail.left;
  else if (item.right > rail.right) nav.scrollLeft += item.right - rail.right;
  maskPartialTabs(nav);
}

function Tab({ cli, pane, workspace, ctx, item, extra, activeRef }) {
  const selected = pane === item.id;
  return (
    <a
      ref={selected ? activeRef : null}
      href={cliSetupHref(cli, item.id, ctx, workspace)}
      role="tab"
      aria-selected={selected}
      aria-current={selected ? "page" : undefined}
    >{item.label}{extra}</a>
  );
}

export default function CliPaneTabs({ cli, pane = "launch", panes = null, workspace = "", workspaceId = "", agentId = "", scope = "user", focus = "", layer = "", actions = null, hasPackageUpdates = false }) {
  const ctx = { workspaceId, agentId, scope, focus, layer };
  // panes comes from cliPanes(cli). Every CLI carries the setup group; the
  // ones without a native editor render it as an in-development placeholder
  // (owner, 2026-09-15), so the strip is the same shape everywhere.
  const allowed = panes ? new Set(panes) : null;
  const run = RUN.filter((item) => !allowed || allowed.has(item.id));
  const setup = SETUP.filter((item) => !allowed || allowed.has(item.id));
  // The strip scrolls once a CLI carries all eight tabs, so a deep link to
  // e.g. Packages has to bring its own tab into view — otherwise the panel
  // says "Packages …" while the reader sees Launch…Sessions.
  const active = useRef(null);
  useEffect(() => {
    const nav = active.current && active.current.closest(".cli-pane-tabs");
    const frame = requestAnimationFrame(() => revealPaneTab(active.current));
    const onScroll = () => maskPartialTabs(nav);
    if (nav) nav.addEventListener("scroll", onScroll, { passive: true });
    return () => { cancelAnimationFrame(frame); if (nav) nav.removeEventListener("scroll", onScroll); };
  }, [pane, cli]);
  return (
    <div className="cli-pane-bar">
      <nav className="cli-pane-tabs" role="tablist" aria-label="CLI sections">
        {run.map((item) => <Tab key={item.id} cli={cli} pane={pane} workspace={workspace} ctx={ctx} item={item} activeRef={active} />)}
        {run.length && setup.length ? <span className="cli-pane-split" aria-hidden="true" /> : null}
        {setup.map((item) => (
          <Tab
            key={item.id}
            cli={cli}
            pane={pane}
            workspace={workspace}
            ctx={ctx}
            item={item}
            activeRef={active}
            extra={item.id === "packages" && hasPackageUpdates ? <span aria-label="Package updates available"> •</span> : null}
          />
        ))}
      </nav>
      {actions ? <div className="cli-pane-actions" data-align-row>{actions}</div> : null}
    </div>
  );
}
