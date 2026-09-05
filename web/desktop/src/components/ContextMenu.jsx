import * as Popover from "@radix-ui/react-popover";
import * as Tooltip from "@radix-ui/react-tooltip";
import { IconCopy, IconPaste, IconReload, IconSun, IconMonitor, IconMoon } from "./Icons.jsx";
import { isEditableTarget, insertAtCaret } from "../lib/contextMenuClipboard.js";
import { toast } from "../lib/toast.js";

const THEME_ORDER = ["light", "system", "dark"];
const THEME_ICON = { light: IconSun, system: IconMonitor, dark: IconMoon };

// The one generic PiCode context menu (MVP). Anchored to the cursor via a
// zero-size virtual Popover.Anchor rather than @radix-ui/react-context-menu:
// that primitive's Trigger must wrap a real DOM subtree and unconditionally
// preventDefaults every contextmenu event it sees, which would fight the
// modifier-bypass/terminal-exclusion logic in App.jsx's own listener.
export default function ContextMenu({ state, onClose, themeMode, onTheme }) {
  const open = !!state;
  const canCopy = open && !!(state.selection && state.selection.trim());
  const canPaste = open && isEditableTarget(state.target);
  const NextThemeIcon = THEME_ICON[themeMode] || IconMonitor;

  function run(fn) {
    return () => { fn(); onClose(); };
  }

  function copySelection() {
    navigator.clipboard.writeText(state.selection)
      .then(() => toast.ok("Copied."))
      .catch(() => toast.error("Clipboard blocked — copy manually with Ctrl+C."));
  }

  function pasteInto() {
    navigator.clipboard.readText()
      .then((text) => { if (text) insertAtCaret(state.target, text); })
      .catch(() => toast.error("Clipboard blocked — paste manually with Ctrl+V."));
  }

  return (
    <Popover.Root open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <Popover.Anchor asChild>
        <span style={{ position: "fixed", left: open ? state.x : 0, top: open ? state.y : 0, width: 0, height: 0 }} />
      </Popover.Anchor>
      <Popover.Portal>
        <Popover.Content className="um-popover" side="bottom" align="start" sideOffset={2} collisionPadding={8} onOpenAutoFocus={(e) => e.preventDefault()}>
          <Tooltip.Provider delayDuration={300}>
            <Item icon={<IconCopy />} label="Copy" disabled={!canCopy} reason="No text selected." onSelect={run(copySelection)} />
            <Item icon={<IconPaste />} label="Paste" disabled={!canPaste} reason="Right-click a text field to paste." onSelect={run(pasteInto)} />
            <div className="um-divider" />
            <Item icon={<IconReload />} label="Reload PiCode" onSelect={run(() => window.location.reload())} />
            <Item
              icon={<NextThemeIcon />}
              label="Toggle theme"
              onSelect={run(() => onTheme(THEME_ORDER[(THEME_ORDER.indexOf(themeMode) + 1) % THEME_ORDER.length]))}
            />
          </Tooltip.Provider>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}

function Item({ icon, label, onSelect, disabled, reason }) {
  const btn = (
    <button type="button" className="um-item" aria-disabled={disabled ? "true" : undefined} onClick={disabled ? undefined : onSelect}>
      {icon}
      <span className="um-item-name">{label}</span>
    </button>
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
