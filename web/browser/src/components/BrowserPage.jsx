import PageFrame from "./PageFrame.jsx";

// Browser (ADR-0128): the work browser's own settings surface, reached
// from the user menu. Desktop-only by nature — the menu entry renders
// only inside the desktop shell; the web /browser/ app has link-handoff
// settings of its own instead (v2).
export default function BrowserPage({ hidden }) {
  return (
    <PageFrame id="browser-view" title="Browser" hidden={hidden}>
      <h4 className="settings-sub">Work browser</h4>
      <div className="settings-card">
        <p className="settings-hint">Browser tabs open inside the app and share one sign-in profile, so site logins persist on this machine. Site permissions (camera, location, downloads) are asked per site.</p>
      </div>
      <h4 className="settings-sub">Developer access</h4>
      <div className="settings-card">
        <p className="settings-hint">Agents read pages through the app&rsquo;s authenticated channel. An external debugging port stays off unless enabled for development &mdash; this lab build ships it enabled.</p>
      </div>
    </PageFrame>
  );
}
