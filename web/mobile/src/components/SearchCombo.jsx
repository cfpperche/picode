import { useState } from "react";
import * as Popover from "@radix-ui/react-popover";
import { Command } from "cmdk";

export default function SearchCombo({
  id,
  value,
  onChange,
  options,
  label,
  searchPlaceholder,
  disabled,
  footer,
  icon,
  triggerClassName,
  side = "top",
  align = "start",
  ariaLabel,
  onOpen,
}) {
  const [open, setOpen] = useState(false);
  // onOpen lets an owner fetch its options the first time the list is asked
  // for, so a picker backed by a subprocess costs nothing until it is used.
  const openChanged = (next) => { if (next && onOpen) onOpen(); setOpen(next); };
  const searchable = searchPlaceholder !== false;

  return (
    <Popover.Root open={open} onOpenChange={openChanged}>
      <Popover.Trigger asChild>
        <button
          type="button"
          id={id}
          className={triggerClassName || "cockpit-chip"}
          disabled={disabled}
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
        >
          {icon ? <span className="cockpit-chip-icon">{icon}</span> : null}
          <span className="cockpit-chip-label">{label}</span>
          <svg
            width="10"
            height="10"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.4"
            aria-hidden="true"
          >
            <path d="m6 9 6 6 6-6" />
          </svg>
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className="cockpit-pop cockpit-combo-pop search-combo-pop"
          side={side}
          align={align}
          sideOffset={6}
          collisionPadding={8}
        >
          <Command label={searchable && searchPlaceholder ? searchPlaceholder : (ariaLabel || label || "Search")} loop>
            {searchable ? (
              <Command.Input
                className="combo-input"
                placeholder={searchPlaceholder || "Search"}
              />
            ) : null}
            <Command.List className="combo-list">
              <Command.Empty className="combo-empty">No matches</Command.Empty>
              {(options || []).map((o) => (
                <Command.Item
                  key={o.id === "" ? "__default" : o.id}
                  value={(o.label || "") + " " + (o.hint || "") + " " + o.id}
                  disabled={!!o.disabled}
                  onSelect={() => {
                    if (o.id !== value) onChange(o.id);
                    setOpen(false);
                  }}
                  className={
                    "cockpit-opt" + (o.id === value ? " selected" : "")
                  }
                >
                  {o.icon ? (
                    <span className="combo-opt-icon repo-opt-lock">{o.icon}</span>
                  ) : null}
                  <span>{o.label}</span>
                  {o.hint ? <span className="combo-hint">{o.hint}</span> : null}
                </Command.Item>
              ))}
            </Command.List>
          </Command>
          {footer}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
