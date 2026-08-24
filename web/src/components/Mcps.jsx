import PageFrame from "./PageFrame.jsx";

export default function Mcps({ hidden, mcp }) {
  return (
    <PageFrame id="mcps-view" title="MCPs" hidden={hidden}>
      <section className="settings-section">
        <dl className="sys-rows">
          <div className="sys-row">
            <dt>Adapter config</dt>
            <dd>{mcp && mcp.configured ? mcp.path : "not configured"}</dd>
          </div>
        </dl>
      </section>
    </PageFrame>
  );
}
