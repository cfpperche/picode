import { useRef } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import * as Tooltip from "@radix-ui/react-tooltip";
import {
  IconChat, IconChevronRight, IconClear, IconClip, IconCollapse, IconCopy, IconExpand, IconExternal, IconFile, IconFolders,
  IconAgent, IconGit, IconGlobe, IconMonitor, IconTerminal, IconMoon, IconPaste, IconPencil, IconReload, IconScrollEnd, IconSelectAll,
  IconSearch, IconSettings, IconSun, IconTextSize, IconTrash, IconX, IconPlus,
} from "./Icons.jsx";
import { isEditableTarget, insertAtCaret } from "../lib/contextMenuClipboard.js";
import { buildTermMenu } from "../lib/termMenu.js";
import { runTermCommand, focusPane } from "../lib/termActions.js";
import { formatChord, primaryChord } from "../lib/appKeys.js";
import { focusRowLabel } from "@picode/shared/domain/focusMode.js";
import { toast } from "../lib/toast.js";

const THEME_ORDER = ["light", "system", "dark"];
const THEME_ICON = { light: IconSun, system: IconMonitor, dark: IconMoon };

const TERM_ICONS = {
  copy: IconCopy, paste: IconPaste, select: IconSelectAll, ask: IconChat, clip: IconClip,
  file: IconFile, external: IconExternal, end: IconScrollEnd, text: IconTextSize,
  clear: IconClear, search: IconSearch, pencil: IconPencil, settings: IconSettings, folders: IconFolders,
  x: IconX, trash: IconTrash, expand: IconExpand, collapse: IconCollapse, globe: IconGlobe,
};

// The one PiCode context menu. Anchored to the cursor through a zero-size
// virtual trigger rather than @radix-ui/react-context-menu: that primitive's
// Trigger must wrap a real DOM subtree and unconditionally preventDefaults
// every contextmenu event it sees, which would fight the modifier-bypass and
// pane-detection logic in App.jsx's own listener. DropdownMenu gives what a
// menu of this size needs anyway — roving focus, typeahead, submenus — which
// a Popover never had.
export default function ContextMenu({ state, onClose, themeMode, onTheme, termHandlers, onOpenAgent, onOpenTerminal, onGraphAction, focusOn, focusable, onFullscreen, clis, onSaveSnippet }) {
  // Every read of `state` goes through these: the component stays mounted
  // with state === null so Radix keeps owning its own teardown, and the
  // rows below are evaluated on every render, open or not.
  const open = !!state;
  const term = open ? state.term : null;
  const selection = open ? state.selection || "" : "";
  const target = open ? state.target : null;
  const link = open ? state.link : null;
  const graph = open ? state.graph : null;
  // Read once per render, like `graph` above: a row's handler must not reach
  // back into `state` at click time, when the menu may already be closing.
  const graphCtx = open ? state.graphCtx : null;
  const NextThemeIcon = THEME_ICON[themeMode] || IconMonitor;
  const fullscreenChord = formatChord(primaryChord("app.fullscreen.toggle"));
  const ran = useRef(false);

  // Radix closes the menu itself after a row is chosen, and would hand
  // focus back to the 0×0 trigger — nowhere. A menu simply dismissed puts
  // the caret back in the terminal; a menu that ran something leaves focus
  // where that action put it (the message bar, a dialog, another tab).
  function dismiss() {
    const pane = term;
    const acted = ran.current;
    ran.current = false;
    onClose();
    if (pane && !acted) requestAnimationFrame(() => focusPane(pane.id));
  }

  function runTerm(id) {
    return () => {
      ran.current = true;
      runTermCommand(id, { ...term, selection, link }, termHandlers || {});
    };
  }

  function copySelection() {
    copyText(selection);
  }

  function copyText(text) {
    navigator.clipboard.writeText(text)
      .then(() => toast.ok("Copied."))
      .catch(() => toast.error("Clipboard blocked — copy manually with Ctrl+C."));
  }

  // A graph row acts on the repository the reader is looking at: copying is
  // done here, and opening an agent belongs to the app that owns the tabs.
  function runGraph(item) {
    return () => {
      ran.current = true;
      if (item.kind === "copy") copyText(item.value);
      else if (item.kind === "open-agent" && onOpenAgent) onOpenAgent(item.agentId);
      else if (item.kind === "open-terminal" && onOpenTerminal) onOpenTerminal(item.terminalId);
      // A write row opens the form; nothing is composed or sent from a menu.
      else if (item.kind === "action" && onGraphAction) onGraphAction(item, graphCtx);
    };
  }

  function pasteInto() {
    navigator.clipboard.readText()
      .then((text) => { if (text) insertAtCaret(target, text); })
      .catch(() => toast.error("Clipboard blocked — paste manually with Ctrl+V."));
  }

  return (
    <DropdownMenu.Root open={open} modal={false} onOpenChange={(v) => { if (!v) dismiss(); }}>
      <DropdownMenu.Trigger asChild>
        <span style={{ position: "fixed", left: open ? state.x : 0, top: open ? state.y : 0, width: 0, height: 0 }} />
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          className="um-popover"
          side="bottom"
          align="start"
          sideOffset={2}
          collisionPadding={8}
          onCloseAutoFocus={(e) => e.preventDefault()}
        >
          <Tooltip.Provider delayDuration={300}>
            {graph ? (
              <GraphMenu menu={graph} onRun={runGraph} />
            ) : term ? (
              buildTermMenu({ kind: term.kind, selection, cli: term.cli, running: term.running, shell: term.shell, link, findKey: formatChord(primaryChord("app.terminal.find")), focus: !!focusOn, focusable: focusable !== false, focusKey: fullscreenChord, record: term.record, clis })
                .map((row, i) => (row.sep
                  ? <div key={"s" + i} className="um-divider" />
                  : <TermRow key={row.id} row={row} onRun={runTerm} />))
            ) : (
              <>
                <Item
                  icon={<IconCopy />}
                  label="Copy"
                  disabled={!selection.trim()}
                  reason="No text selected."
                  onSelect={copySelection}
                />
                {onSaveSnippet ? (
                  <Item
                    icon={<IconPlus />}
                    label="Save selection as snippet"
                    disabled={!selection.trim()}
                    reason="Select text first."
                    onSelect={() => { ran.current = true; onSaveSnippet(selection); }}
                  />
                ) : null}
                <Item icon={<IconPaste />} label="Paste" disabled={!isEditableTarget(target)} reason="Right-click a text field to paste." onSelect={pasteInto} />
                <div className="um-divider" />
                <Item icon={<IconReload />} label="Reload PiCode" onSelect={() => window.location.reload()} />
                <Item
                  icon={<NextThemeIcon />}
                  label="Toggle theme"
                  onSelect={() => onTheme(THEME_ORDER[(THEME_ORDER.indexOf(themeMode) + 1) % THEME_ORDER.length])}
                />
                {focusable !== false && onFullscreen ? (
                  <Item
                    icon={focusOn ? <IconCollapse /> : <IconExpand />}
                    label={focusRowLabel(!!focusOn)}
                    chord={fullscreenChord}
                    onSelect={() => { ran.current = true; onFullscreen(); }}
                  />
                ) : null}
              </>
            )}
          </Tooltip.Provider>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

// The git graph's menu (ADR-0096). The header is what you pointed at plus
// the facts that decide what may be done to it; phase 1 carries tier 0 rows
// only, so nothing here runs git. An empty item list renders as the header
// alone — the reader still learns which checkout holds the branch.
// A row's icon says what kind of act it is before the label does: copying,
// opening a tab, or a git command — and a tier C command wears the app's
// danger colour, as every other destructive row does.
function graphIcon(item) {
  if (item.kind === "copy") return <IconCopy />;
  if (item.kind === "open-agent") return <IconAgent />;
  if (item.kind === "open-terminal") return <IconTerminal />;
  return <IconGit />;
}

function GraphMenu({ menu, onRun }) {
  return (
    <>
      <div className="um-head">
        <span className="um-head-name">{menu.title}</span>
        {menu.state ? <span className="um-note">{menu.state}</span> : null}
        {menu.busy ? <span className="um-note um-note-busy">{menu.busy}</span> : null}
      </div>
      {menu.items.length ? <div className="um-divider" /> : null}
      {menu.items.map((item) => (item.kind === "section"
        ? (
          <DropdownMenu.Label key={item.id} className="um-label um-label-mid">
            {item.label}
            {item.state ? <span className="um-note">{item.state}</span> : null}
          </DropdownMenu.Label>
        )
        : (
          <DropdownMenu.Item
            key={item.id}
            className={"um-item" + (item.tier === "C" ? " um-danger" : "")}
            onSelect={onRun(item)}
          >
            <span className="um-item-name">{graphIcon(item)}{item.label}</span>
          </DropdownMenu.Item>
        )))}
    </>
  );
}

function TermRow({ row, onRun }) {
  const Icon = TERM_ICONS[row.icon];
  const face = <><span className="um-item-name">{Icon ? <Icon /> : null}{row.label}</span>{row.key ? <kbd className="um-key">{row.key}</kbd> : null}</>;
  if (row.sub) {
    return (
      <DropdownMenu.Sub>
        <DropdownMenu.SubTrigger className="um-item">
          <span className="um-item-name">{Icon ? <Icon /> : null}{row.label}</span>
          <IconChevronRight size={13} className="um-chev" />
        </DropdownMenu.SubTrigger>
        <DropdownMenu.Portal>
          <DropdownMenu.SubContent className="um-popover" sideOffset={2} collisionPadding={8}>
            {row.sub.map((s) => s.sub ? (
              <DropdownMenu.Sub key={s.id}>
                <DropdownMenu.SubTrigger className="um-item">
                  <span className="um-item-name">{s.label}</span>
                  <IconChevronRight size={13} className="um-chev" />
                </DropdownMenu.SubTrigger>
                <DropdownMenu.Portal>
                  <DropdownMenu.SubContent className="um-popover" sideOffset={2} collisionPadding={8}>
                    {s.sub.map((s2) => (
                      <DropdownMenu.Item
                        key={s2.id}
                        className="um-item"
                        disabled={!!s2.disabled}
                        title={s2.title || undefined}
                        onSelect={s2.disabled ? undefined : onRun(s2.id)}
                      >
                        <span className="um-item-name">{s2.label}</span>
                      </DropdownMenu.Item>
                    ))}
                  </DropdownMenu.SubContent>
                </DropdownMenu.Portal>
              </DropdownMenu.Sub>
            ) : (
              <DropdownMenu.Item
                key={s.id}
                className="um-item"
                disabled={!!s.disabled}
                title={s.title || undefined}
                onSelect={s.disabled ? undefined : onRun(s.id)}
              >
                <span className="um-item-name">{s.label}</span>
                {s.key ? <kbd className="um-key">{s.key}</kbd> : null}
              </DropdownMenu.Item>
            ))}
          </DropdownMenu.SubContent>
        </DropdownMenu.Portal>
      </DropdownMenu.Sub>
    );
  }
  const item = (
    <DropdownMenu.Item
      className={"um-item" + (row.danger ? " um-danger" : "")}
      disabled={!!row.disabled}
      aria-disabled={row.disabled ? "true" : undefined}
      title={row.hint || undefined}
      onSelect={row.disabled ? undefined : onRun(row.id)}
    >
      {face}
    </DropdownMenu.Item>
  );
  if (!row.disabled || !row.reason) return item;
  return (
    <Tooltip.Root>
      <Tooltip.Trigger asChild>{item}</Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Content className="ctx-tip" side="right" sideOffset={6} collisionPadding={8}>
          {row.reason}
        </Tooltip.Content>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}

function Item({ icon, label, onSelect, disabled, reason, chord }) {
  // Icon rides inside `um-item-name`, exactly like TermRow: `.um-item` is
  // space-between, so an icon left as a direct child shoves the label to the
  // far edge instead of sitting beside it.
  const btn = (
    <DropdownMenu.Item
      className="um-item"
      disabled={!!disabled}
      aria-disabled={disabled ? "true" : undefined}
      onSelect={disabled ? undefined : onSelect}
    >
      <span className="um-item-name">{icon}{label}</span>
      {chord ? <kbd className="um-key">{chord}</kbd> : null}
    </DropdownMenu.Item>
  );
  if (!disabled || !reason) return btn;
  return (
    <Tooltip.Root>
      <Tooltip.Trigger asChild>{btn}</Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Content className="ctx-tip" side="right" sideOffset={6} collisionPadding={8}>
          {reason}
        </Tooltip.Content>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}
