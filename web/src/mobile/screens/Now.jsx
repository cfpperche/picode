import NeedsYouCard from "../components/NeedsYouCard.jsx";
import StatStrip from "../components/StatStrip.jsx";
import StateChip, { agentState } from "../components/StateChip.jsx";
import { ProviderFace } from "../../components/ProviderFaces.jsx";
import { IconChevronRight } from "../../components/Icons.jsx";
import { displayAgentName } from "../../lib/tree.js";
import { shortModel } from "../../lib/chip.js";
import { relTime, absTime } from "../../lib/relTime.js";

// The home is a queue of decisions (PagerDuty's "top open incidents"),
// then who is running, then today's numbers, then what finished. Nothing
// here duplicates the Agents tab: an idle agent is one tap away, not a row.
export default function Now({ loaded, entries, running, workingIds, stats, results, onAnswer, onRespond, onOpenAgent, onOpenInbox, onCreate, fleetTotal }) {
  if (loaded && fleetTotal === 0) {
    return (
      <div className="m-screen">
        <div className="m-blank">
          <p className="m-blank-title">No agents yet</p>
          <p className="m-blank-sub">Add a project folder to create your first agent.</p>
          <button type="button" className="btn btn-primary" onClick={() => onCreate("workspace")}>Add workspace</button>
        </div>
      </div>
    );
  }
  return (
    <div className="m-screen">
      <section className="m-section">
        <h2 className="m-section-label">Needs you{entries.length ? <span className="m-count">{entries.length}</span> : null}</h2>
        {!loaded ? <Skel /> : entries.length === 0 ? (
          <p className="m-empty-line">Nothing needs you right now.</p>
        ) : entries.map((e) => (
          <NeedsYouCard key={e.key} entry={e} onAnswer={onAnswer} onRespond={onRespond} onOpen={(en) => (en.kind === "ask" ? onOpenAgent(en.agentId) : onOpenInbox(en.itemId))} />
        ))}
      </section>

      <section className="m-section">
        <h2 className="m-section-label">Running{running.length ? <span className="m-count">{running.length}</span> : null}</h2>
        {!loaded ? <Skel /> : running.length === 0 ? (
          <p className="m-empty-line">Nothing running. <button type="button" className="btn-link" onClick={() => onOpenAgent("")}>Agents</button></p>
        ) : (
          <ul className="m-list">
            {running.map(({ agent, workspace }) => (
              <li key={agent.id} className="m-row">
                <button type="button" className="m-row-main" onClick={() => onOpenAgent(agent.id)}>
                  <span className="m-row-face"><ProviderFace agent={agent} /></span>
                  <span className="m-row-text">
                    <span className="m-row-title">{displayAgentName(agent, workspace)}</span>
                    <span className="m-row-sub">{[shortModel(agent.model || ""), workspace ? workspace.name : "free agent"].filter(Boolean).join(" · ")}</span>
                  </span>
                  <StateChip state={agentState(agent, workingIds)} />
                  <IconChevronRight size={16} className="m-row-chev" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="m-section">
        <h2 className="m-section-label">Today</h2>
        {stats ? <StatStrip stats={stats} /> : <Skel />}
      </section>

      <section className="m-section">
        <h2 className="m-section-label">Recent results</h2>
        {!loaded ? <Skel /> : results.length === 0 ? (
          <p className="m-empty-line">No finished runs yet.</p>
        ) : (
          <ul className="m-list">
            {results.map((it) => (
              <li key={it.id} className={"m-row" + (it.state === "unread" ? " is-unread" : "")}>
                <button type="button" className="m-row-main" onClick={() => onOpenInbox(it.id)}>
                  <span className="m-row-text">
                    <span className="m-row-title">{it.title}</span>
                    <span className="m-row-sub">{it.reason}</span>
                  </span>
                  <span className="m-row-when" title={absTime(it.createdAt)}>{relTime(it.createdAt)}</span>
                  <IconChevronRight size={16} className="m-row-chev" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function Skel() {
  return (
    <div className="m-skel" aria-hidden="true">
      <span className="skel-line w-70" />
      <span className="skel-line w-40" />
    </div>
  );
}
