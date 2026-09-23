import { useEffect, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api } from "@picode/shared/client/api.js";
import { workspaceRowMenu, wantsPullRequest } from "@picode/shared/domain/workspaceRowMenu.js";
import { IconChevronRight, IconCommunication, IconCopy, IconExternal, IconFolder, IconFolderOpen, IconGit, IconMoveDown, IconMoveUp, IconPullRequest, IconSession, IconSettings, IconX } from "./Icons.jsx";
import { RowMenu, RowMenuItem, RowMenuSep } from "./WorkspaceRows.jsx";
import WorkspaceSettings from "./WorkspaceSettings.jsx";
import { OPEN_WORKSPACE_SETTINGS } from "./LandingWork.jsx";
import { toast, toastError } from "../lib/toast.js";

const ICONS = {
  communication: <IconCommunication size={13} />,
  files: <IconFolder size={13} />,
  "git-graph": <IconGit size={13} />,
  sessions: <IconSession size={13} />,
  reveal: <IconFolderOpen size={13} />,
  remote: <IconExternal size={13} />,
  pr: <IconPullRequest size={13} />,
  "copy-path": <IconCopy size={13} />,
  settings: <IconSettings size={13} />,
  "move-up": <IconMoveUp size={13} />,
  "move-down": <IconMoveDown size={13} />,
  remove: <IconX size={13} />,
};

// copyText falls back to a hidden textarea: a PiCode reached by IP over plain
// HTTP is not a secure context, and navigator.clipboard is undefined there.
async function copyText(value) {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
      return true;
    }
  } catch { /* fall through to the textarea */ }
  const ta = document.createElement("textarea");
  ta.value = value;
  ta.setAttribute("readonly", "");
  ta.style.cssText = "position:fixed;opacity:0;pointer-events:none";
  document.body.appendChild(ta);
  ta.select();
  let ok = false;
  try { ok = document.execCommand("copy"); } catch { ok = false; }
  ta.remove();
  return ok;
}

// The workspace card's "…" menu. The rows are workspaceRowMenu.js; this
// component renders them and asks for the pull request when it opens, so a
// closed menu never costs a gh call. The last answer stays on screen while a
// reopen asks again (the server caches it for a minute anyway).
export default function WorkspaceMenu({ ws, hasAgents, onMoveUp, onMoveDown, onFileTree, onGitGraph, onSessions, onRemove }) {
  const branch = (ws.git && ws.git.branch) || "";
  const [pr, setPr] = useState({ branch: null, page: undefined });
  const [settingsOpen, setSettingsOpen] = useState(false);
  // Set by the Settings item: the closing menu must not hand focus back to
  // its trigger, or on the phone sheet (which never focuses a field) focus
  // stays behind the sheet and Escape closes the drawer under it too.
  const toSettings = useRef(false);
  const triggerRef = useRef(null);
  const returnTo = useRef(null);

  // Preferences → Landing work's Edit opens this card's dialog.
  useEffect(() => {
    const onOpen = (e) => {
      if (e.detail !== ws.id) return;
      returnTo.current = document.activeElement; // the Edit button: focus goes back there
      setSettingsOpen(true);
    };
    window.addEventListener(OPEN_WORKSPACE_SETTINGS, onOpen);
    return () => window.removeEventListener(OPEN_WORKSPACE_SETTINGS, onOpen);
  }, [ws.id]);

  function onOpenChange(open) {
    if (!open || !wantsPullRequest(ws)) return;
    if (pr.branch !== branch) setPr({ branch, page: undefined });
    api("/api/workspaces/" + encodeURIComponent(ws.id) + "/pr")
      .then((page) => setPr({ branch, page }))
      .catch(() => setPr({ branch, page: null }));
  }

  async function reveal() {
    try {
      await api("/api/workspaces/" + encodeURIComponent(ws.id) + "/reveal", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: "{}",
      });
    } catch (e) { toastError(e); }
  }

  async function copy(value, what) {
    if (await copyText(value)) toast.ok(what + " copied.");
    else toast.error("Clipboard blocked — select the path in Files instead.");
  }

  function select(r) {
    if (r.url) { window.open(r.url, "_blank", "noopener,noreferrer"); return; }
    switch (r.id) {
      case "communication": location.hash = "#/clis/messages/" + encodeURIComponent("workspace:" + ws.id); break;
      case "files": onFileTree && onFileTree("workspace", ws.id, ws.name); break;
      case "git-graph": onGitGraph && onGitGraph("workspace", ws.id, ws.name); break;
      case "sessions": onSessions && onSessions(ws.id); break;
      case "reveal": void reveal(); break;
      case "copy-path": void copy(r.value, "Path"); break;
      case "settings": toSettings.current = true; setSettingsOpen(true); break;
      case "move-up": onMoveUp && onMoveUp(); break;
      case "move-down": onMoveDown && onMoveDown(); break;
      case "remove": onRemove(ws); break;
      default:
    }
  }

  const rows = workspaceRowMenu(ws, {
    hasAgents,
    canMoveUp: !!onMoveUp,
    canMoveDown: !!onMoveDown,
    pr: pr.branch === branch ? pr.page : undefined,
  });

  return (
    <>
    <RowMenu label={ws.name} triggerRef={triggerRef} onOpenChange={onOpenChange} onCloseAutoFocus={(e) => { if (toSettings.current) { e.preventDefault(); toSettings.current = false; } }}>
      {rows.map((r, i) => {
        if (r.sep) return <RowMenuSep key={"sep" + i} />;
        if (r.sub) return (
          <DropdownMenu.Sub key={r.id}>
            <DropdownMenu.SubTrigger className="ws-row-menu-item">
              {ICONS[r.id]} {r.label}
              <IconChevronRight size={13} className="um-chev" />
            </DropdownMenu.SubTrigger>
            <DropdownMenu.Portal>
              <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                {r.sub.map((s) => (
                  <DropdownMenu.Item key={s.id} className="ws-row-menu-item" title={s.title} onSelect={() => void copy(s.value, s.label)}>
                    {s.label}
                  </DropdownMenu.Item>
                ))}
              </DropdownMenu.SubContent>
            </DropdownMenu.Portal>
          </DropdownMenu.Sub>
        );
        if (r.pending) return (
          <DropdownMenu.Item key={r.id} className="ws-row-menu-item" disabled>
            {ICONS[r.id]} {r.label}
          </DropdownMenu.Item>
        );
        return (
          <RowMenuItem key={r.id} title={r.title} danger={r.danger} onSelect={() => select(r)}>
            {ICONS[r.id] || null} {r.label}
          </RowMenuItem>
        );
      })}
    </RowMenu>
    <WorkspaceSettings ws={ws} open={settingsOpen} onClose={() => setSettingsOpen(false)} returnFocus={() => { const to = returnTo.current; returnTo.current = null; (to && to.isConnected ? to : triggerRef.current)?.focus(); }} />
    </>
  );
}
