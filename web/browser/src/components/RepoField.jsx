import { useEffect, useMemo, useRef, useState } from "react";
import * as Popover from "@radix-ui/react-popover";
import { Command } from "cmdk";
import { fetchGithubRepos } from "@picode/shared/client/githubRepos.js";
import { deriveRepo } from "@picode/shared/domain/cloneUrl.js";
import { IconLock } from "./Icons.jsx";

// The clone-form URL field, upgraded into a filterable combobox (ADR-0034's
// "paste a URL" stays first-class): typing filters the repositories behind
// the machine's gh login; a pasted URL of any host still works — when it
// parses, a "Use this URL" row leads the list and Enter submits it as
// before. Pick a row and the same derivation as before fills name and
// destination. Blocked setups (no gh / not logged in) degrade to one line
// with the one action that fixes it.
export default function RepoField({
  name,
  value,
  onValue,
  placeholder,
  autoFocus,
}) {
  const [state, setState] = useState({ phase: "loading" });
  const [attempt, setAttempt] = useState(0);
  const [copied, setCopied] = useState(false);
  const listRef = useRef(null);

  useEffect(() => {
    let alive = true;
    setState({ phase: "loading" });
    fetchGithubRepos(attempt > 0).then((s) => {
      if (alive) setState(s);
    });
    return () => {
      alive = false;
    };
  }, [attempt]);

  const repos = state.phase === "ready" ? state.repos : [];
  const urlParses = !!deriveRepo(value).name;
  const q = value.trim().toLowerCase();
  const filtered = useMemo(() => {
    if (!q) return repos;
    return repos.filter((r) =>
      (r.nameWithOwner + " " + (r.description || "")).toLowerCase().includes(q),
    );
  }, [repos, q]);
  const groups = useMemo(() => groupByOwner(filtered), [filtered]);

  // Open is derived, but "focused" lands a frame AFTER the focus event:
  // opening synchronously inside focusin makes Radix's own guards see that
  // same event as "focus moved outside the layer" and close us at once.
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const ready = state.phase === "ready";
  const open = focused && ready && !dismissed;
  const openTimer = useRef(null);
  const inputRef = useRef(null);
  // Was the field already focused when this click's pointerdown landed? If
  // not, the click IS the focusing interaction and must end open — the same
  // click otherwise toggles shut what its own focus just opened.
  const pdWasFocused = useRef(false);

  function scheduleOpen() {
    cancelAnimationFrame(openTimer.current);
    openTimer.current = requestAnimationFrame(() => {
      setFocused(true);
    });
  }
  function cancelOpen() {
    cancelAnimationFrame(openTimer.current);
    setFocused(false);
    setDismissed(false);
  }

  function pick(url) {
    onValue(url);
    setDismissed(true);
  }
  function focusItem() {
    // Hand focus to cmdk's active (or first) row so arrows work from the input.
    const scope = listRef.current;
    if (!scope) return;
    const el =
      scope.querySelector('[cmdk-item][data-selected="true"]') ||
      scope.querySelector("[cmdk-item]");
    if (el) el.focus();
  }
  function onInputKeyDown(e) {
    if (e.key === "ArrowDown" && !e.altKey && !e.metaKey && !e.ctrlKey) {
      if (ready) {
        e.preventDefault();
        setFocused(true);
        setDismissed(false);
        requestAnimationFrame(focusItem);
      }
    } else if (e.key === "Escape" && open) {
      setDismissed(true);
    } else if (e.key === "Enter" && open) {
      const scope = listRef.current;
      if (!scope) return;
      const el =
        scope.querySelector('[cmdk-item][data-selected="true"]') ||
        scope.querySelector("[cmdk-item]");
      if (el) {
        e.preventDefault();
        el.click();
      }
    }
  }
  function copyLogin() {
    try {
      navigator.clipboard.writeText("gh auth login");
    } catch {
      /* reason text already shows the command */
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="repo-field">
      <Popover.Root
        open={open}
        onOpenChange={(o) => {
          if (!o) setDismissed(true);
        }}
      >
        <Popover.Anchor asChild>
          <input
            ref={inputRef}
            name={name}
            type="text"
            autoComplete="off"
            autoFocus={autoFocus}
            spellCheck={false}
            placeholder={placeholder}
            role="combobox"
            aria-expanded={open}
            aria-controls="repo-picker-list"
            aria-autocomplete="list"
            value={value}
            onChange={(e) => {
              onValue(e.target.value);
              setFocused(true);
              setDismissed(false);
            }}
            onFocus={() => {
              scheduleOpen();
            }}
            onBlur={(e) => {
              // Focus moving into the list (arrow-down) is not a close.
              const rt = e.relatedTarget;
              if (rt && listRef.current && listRef.current.contains(rt)) return;
              cancelOpen();
            }}
            onPointerDown={() => {
              pdWasFocused.current =
                document.activeElement === inputRef.current;
            }}
            onClick={() => {
              if (pdWasFocused.current && open) setDismissed(true);
              else if (ready) {
                setDismissed(false);
                scheduleOpen();
              }
            }}
            onKeyDown={onInputKeyDown}
          />
        </Popover.Anchor>
        <Popover.Portal>
          <Popover.Content
            className="cockpit-pop cockpit-combo-pop repo-pop"
            side="bottom"
            align="start"
            sideOffset={6}
            collisionPadding={8}
            onOpenAutoFocus={(e) => e.preventDefault()}
            onCloseAutoFocus={(e) => e.preventDefault()}
            onPointerDownOutside={(e) => {
              if (inputRef.current && e.target === inputRef.current)
                e.preventDefault();
            }}
            onFocusOutside={(e) => {
              if (inputRef.current && e.target === inputRef.current)
                e.preventDefault();
            }}
          >
            <Command label="Your repositories" loop shouldFilter={false}>
              <Command.List
                className="combo-list repo-list"
                id="repo-picker-list"
                ref={listRef}
                // Keep the input's focus while a row is clicked — a blur here
                // would unmount the list before the item's click lands.
                onMouseDown={(e) => e.preventDefault()}
              >
                {state.phase === "loading" ? (
                  <div className="file-skel" aria-hidden="true">
                    <div className="skel-line w-80" />
                    <div className="skel-line w-50" />
                  </div>
                ) : filtered.length === 0 && !urlParses ? (
                  <Command.Empty className="combo-empty">
                    {q
                      ? "No repositories match — paste a full repository URL."
                      : "No repositories on this account — paste a URL above."}
                  </Command.Empty>
                ) : (
                  <>
                    {urlParses ? (
                      <Command.Item
                        value={"use " + value}
                        onSelect={() => pick(value.trim())}
                        className="cockpit-opt repo-use-url"
                      >
                        Use this URL
                      </Command.Item>
                    ) : null}
                    {groups.map(([owner, rows]) => (
                      <Command.Group key={owner} heading={owner}>
                        {rows.map((r) => (
                          <Command.Item
                            key={r.url || r.nameWithOwner}
                            value={
                              r.nameWithOwner + " " + (r.description || "")
                            }
                            onSelect={() => pick(r.url || r.nameWithOwner)}
                            className="cockpit-opt repo-opt"
                          >
                            <span className="repo-opt-name">
                              {r.nameWithOwner}
                            </span>
                            {r.isPrivate ? <IconLock /> : null}
                            {r.description ? (
                              <span className="combo-hint repo-opt-desc">
                                {r.description}
                              </span>
                            ) : null}
                          </Command.Item>
                        ))}
                      </Command.Group>
                    ))}
                  </>
                )}
              </Command.List>
            </Command>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
      {state.phase === "blocked" ? (
        <p className="repo-status" role="status">
          <span>{state.reason}</span>
          {state.kind === "install" ? (
            <a
              className="btn btn-ghost btn-sm"
              href="https://cli.github.com"
              target="_blank"
              rel="noreferrer"
            >
              How to install
            </a>
          ) : state.kind === "login" ? (
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={copyLogin}
            >
              {copied ? "Copied" : "Copy sign-in command"}
            </button>
          ) : null}
        </p>
      ) : state.phase === "error" ? (
        <p className="repo-status" role="status">
          <span>{state.reason}</span>
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={() => setAttempt(attempt + 1)}
          >
            Retry
          </button>
        </p>
      ) : null}
    </div>
  );
}

function groupByOwner(repos) {
  const byOwner = new Map();
  for (const r of repos) {
    const i = r.nameWithOwner.indexOf("/");
    const owner = i > 0 ? r.nameWithOwner.slice(0, i) : "Repositories";
    if (!byOwner.has(owner)) byOwner.set(owner, []);
    byOwner.get(owner).push(r);
  }
  const owners = [...byOwner.keys()].sort((a, b) => a.localeCompare(b));
  return owners.map((owner) => [
    owner,
    byOwner
      .get(owner)
      .sort((a, b) => (b.updatedAt || "").localeCompare(a.updatedAt || "")),
  ]);
}
