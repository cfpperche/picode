import NeedsYouCard from "../components/NeedsYouCard.jsx";
import StatStrip from "../components/StatStrip.jsx";
import { IconChevronRight } from "../components/Icons.jsx";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import PullScreen from "../components/PullScreen.jsx";
import "../styles/mobile-lists.css";

// The home is a queue of decisions (PagerDuty's "top open incidents"),
// then today's numbers, then what finished. Who is running lives in the
// Work tab; nothing here duplicates it.
export default function Now({ loaded, error, entries, stats, results, onAnswer, onRespond, onOpenAgent, onOpenInbox, onCreate, fleetTotal, onRefresh }) {
  if (loaded && !error && fleetTotal === 0) {
    return (
      <div className="m-screen m-v2-lists m-now-v2">
        <div className="m-list-empty">
          <p>No agents or terminals yet.</p>
          <button type="button" className="btn btn-primary btn-sm" onClick={() => onCreate("workspace")}>Add workspace</button>
        </div>
      </div>
    );
  }
  return (
    <PullScreen onRefresh={onRefresh} className="m-v2-lists m-now-v2">
      {error ? <div className="m-list-notice" role="alert"><p>{loaded ? "Couldn’t refresh your work." : "Couldn’t load your work."}</p><button type="button" className="btn btn-sm" onClick={onRefresh}>Try again</button></div> : null}
      {!loaded && error ? null : <>
      <section className="m-section">
        <div className="m-section-heading"><h2 className="m-section-label">Needs you{entries.length ? <span className="m-count">{entries.length}</span> : null}</h2><a className="m-section-link" href="#/inbox">Inbox <IconChevronRight size={13} /></a></div>
        {!loaded ? <Skel /> : entries.length === 0 ? (
          <p className="m-empty-line">Nothing needs you right now.</p>
        ) : entries.map((e) => (
          <NeedsYouCard key={e.key} entry={e} onAnswer={onAnswer} onRespond={onRespond} onOpen={(en) => (en.kind === "ask" ? onOpenAgent(en.agentId) : onOpenInbox(en.itemId))} />
        ))}
      </section>

      <section className="m-section">
        <h2 className="m-section-label">Recent results</h2>
        {!loaded ? <Skel /> : results.length === 0 ? (
          <p className="m-empty-line">No finished runs yet.</p>
        ) : (
          <ul className="m-list m-group-list">
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
      <section className="m-section m-today-section">
        <h2 className="m-section-label">Today</h2>
        {stats ? <StatStrip stats={stats} /> : <Skel />}
      </section>
      </>}
    </PullScreen>
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
