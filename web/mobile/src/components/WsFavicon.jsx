import { useState } from "react";
import { IconFolder } from "./Icons.jsx";

// Workspace groups wear the project's favicon when one exists, same as the
// desktop sidebar. The list advertises whether one exists (hasFavicon), so a
// normal workspace without an icon never creates an expected 404; failures
// are still remembered per page-load when the file disappears between the
// list and the image request.
const faviconFailed = new Set();
export default function WsFavicon({ ws, size = 16 }) {
  const [failed, setFailed] = useState(faviconFailed.has(ws.id));
  if (failed || ws.hasFavicon === false) return <IconFolder size={size} />;
  return (
    <img
      className="ws-favicon" width={size} height={size} alt="" loading="lazy"
      src={"/api/workspaces/" + encodeURIComponent(ws.id) + "/favicon"}
      onError={() => { faviconFailed.add(ws.id); setFailed(true); }}
    />
  );
}
