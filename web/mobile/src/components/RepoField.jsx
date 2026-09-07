import { useEffect, useMemo, useState } from "react";
import SearchCombo from "./SearchCombo.jsx";
import { fetchGithubRepos } from "@picode/shared/client/githubRepos.js";
import { IconLock } from "./Icons.jsx";

// Mobile half of the clone-form repo picker (the desktop half is an inline
// combobox on the URL field): the input keeps paste-anything-first, and a
// tap-to-open picker lists the repositories behind the machine's gh login.
// Blocked setups (no gh / not logged in) degrade to one line with the one
// action that fixes it — the same lines the desktop shows.
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
  const options = useMemo(
    () =>
      repos.map((r) => ({
        id: r.url || r.nameWithOwner,
        label: r.nameWithOwner,
        hint: r.description || "",
        icon: r.isPrivate ? <IconLock /> : null,
      })),
    [repos],
  );

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
      <input
        name={name}
        type="text"
        autoComplete="off"
        autoFocus={autoFocus}
        spellCheck={false}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onValue(e.target.value)}
      />
      {state.phase === "ready" && options.length ? (
        <SearchCombo
          value={null}
          onChange={(url) => onValue(url)}
          options={options}
          label="Choose from your GitHub"
          searchPlaceholder="Filter repositories"
        />
      ) : state.phase === "ready" ? (
        <p className="repo-status" role="status">
          <span>No repositories on this account — paste a URL above.</span>
        </p>
      ) : state.phase === "blocked" ? (
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
