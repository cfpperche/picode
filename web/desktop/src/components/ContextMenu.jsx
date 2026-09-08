import { useRef } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import * as Tooltip from "@radix-ui/react-tooltip";
import {
  IconChat, IconChevronRight, IconClear, IconClip, IconCopy, IconExternal, IconFile, IconFolders,
  IconMonitor, IconMoon, IconPaste, IconPencil, IconReload, IconScrollEnd, IconSelectAll,
  IconSettings, IconSun, IconTextSize, IconTrash, IconX,
} from "./Icons.jsx";
import { isEditableTarget, insertAtCaret } from "../lib/contextMenuClipboard.js";
import { buildTermMenu } from "../lib/termMenu.js";
import { runTermCommand, focusPane } from "../lib/termActions.js";
import { toast } from "../lib/toast.js";

const THEME_ORDER = ["light", "system", "dark"];
const THEME_ICON = { light: IconSun, system: IconMonitor, dark: IconMoon };

const TERM_ICONS = {
  copy: IconCopy, paste: IconPaste, select: IconSelectAll, ask: IconChat, clip: IconClip,
  file: IconFile, external: IconExternal, end: IconScrollEnd, text: IconTextSize,
  clear: IconClear, pencil: IconPencil, settings: IconSettings, folders: IconFolders,
  x: IconX, trash: IconTrash,
};

// The one PiCode context menu. Anchored to the cursor through a zero-size
// virtual trigger rather than @radix-ui/react-context-menu: that primitive's
// Trigger must wrap a real DOM subtree and unconditionally preventDefaults
// every contextmenu event it sees, which would fight the modifier-bypass and
// pane-detection logic in App.jsx's own listener. DropdownMenu gives what a
// menu of this size needs anyway — roving focus, typeahead, submenus — which
// a Popover never had.
export default function ContextMenu({ state, onClose, themeMode, onTheme, termHandlers }) {
  // Every read of `state` goes through these: the component stays mounted
  // with state === null so Radix keeps owning its own teardown, and the
  // rows below are evaluated on every render, open or not.
  const open = !!state;
  const term = open ? state.term : null;
  const selection = open ? state.selection || "" : "";
  const target = open ? state.target : null;
  const link = open ? state.link : null;
  const NextThemeIcon = THEME_ICON[themeMode] || IconMonitor;
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
    navigator.clipboard.writeText(selection)
      .then(() => toast.ok("Copied."))
      .catch(() => toast.error("Clipboard blocked — copy manually with Ctrl+C."));
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
            {term ? (
              buildTermMenu({ kind: term.kind, selection, cli: term.cli, running: term.running, shell: term.shell, link })
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
                <Item icon={<IconPaste />} label="Paste" disabled={!isEditableTarget(target)} reason="Right-click a text field to paste." onSelect={pasteInto} />
                <div className="um-divider" />
                <Item icon={<IconReload />} label="Reload PiCode" onSelect={() => window.location.reload()} />
                <Item
                  icon={<NextThemeIcon />}
                  label="Toggle theme"
                  onSelect={() => onTheme(THEME_ORDER[(THEME_ORDER.indexOf(themeMode) + 1) % THEME_ORDER.length])}
                />
              </>
            )}
          </Tooltip.Provider>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
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
            {row.sub.map((s) => (
              <DropdownMenu.Item key={s.id} className="um-item" onSelect={onRun(s.id)}>
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

function Item({ icon, label, onSelect, disabled, reason }) {
  const btn = (
    <DropdownMenu.Item
      className="um-item"
      disabled={!!disabled}
      aria-disabled={disabled ? "true" : undefined}
      onSelect={disabled ? undefined : onSelect}
    >
      {icon}
      <span className="um-item-name">{label}</span>
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
