import { useEffect, useState } from "react";
import { integrationSection } from "@picode/shared/domain/integrations.js";
import PageFrame from "./PageFrame.jsx";
import Mcps from "./Mcps.jsx";
import Webhooks from "./Webhooks.jsx";
import "../styles/integrations.css";

export default function Integrations({ hidden, ...context }) {
  const [section, setSection] = useState(() => integrationSection(location.hash));
  useEffect(() => {
    const change = () => setSection(integrationSection(location.hash));
    window.addEventListener("hashchange", change);
    return () => window.removeEventListener("hashchange", change);
  }, []);
  return <PageFrame id="integrations-view" title="Integrations" hidden={hidden}>
    <nav className="pref-tabs integration-tabs" aria-label="Integrations">
      <a className="pref-tab" href="#/integrations/connectors" aria-current={section === "connectors" ? "page" : undefined}>Connectors</a>
      <a className="pref-tab" href="#/integrations/webhooks" aria-current={section === "webhooks" ? "page" : undefined}>Webhooks</a>
    </nav>
    <Mcps embedded {...context} hidden={hidden || section !== "connectors"} />
    <Webhooks hidden={hidden || section !== "webhooks"} />
  </PageFrame>;
}
