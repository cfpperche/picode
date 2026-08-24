import PageFrame from "./PageFrame.jsx";

export default function Providers({ hidden, catalog, onSignIn }) {
  const list = catalog && catalog.providers ? catalog.providers : [];
  return (
    <PageFrame id="providers-view" title="Providers" hidden={hidden}>
      <section className="settings-section">
        <ul className="prov-list">
          {list.map((p) => (
            <li key={p.id} className="prov-row">
              <span className="prov-id">{p.id}</span>
              <span className={"prov-auth" + (p.signedIn ? " in" : "")}>{p.signedIn ? "signed in" : "not signed in"}</span>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => onSignIn(p.id)}>Sign in</button>
            </li>
          ))}
        </ul>
      </section>
    </PageFrame>
  );
}
