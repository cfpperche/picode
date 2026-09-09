import PageFrame from "./PageFrame.jsx";
import "./agent-clis.css";

// Every Agent CLIs route keeps the same page geometry when tabs change.
export default function AgentClisFrame({ id = "agent-clis-view", hidden, children }) {
  return <PageFrame id={id} title="Agent CLIs" className="cli-page" hidden={hidden} wide>{children}</PageFrame>;
}
