import PageFrame from "./PageFrame.jsx";
import Webhooks from "./Webhooks.jsx";
import "../styles/integrations.css";

export default function Integrations({ hidden }) {
  return <PageFrame id="integrations-view" title="Webhooks" hidden={hidden}>
    <Webhooks hidden={hidden} />
  </PageFrame>;
}
