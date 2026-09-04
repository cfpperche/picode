import * as Dialog from "./ResponsiveDialog.jsx";
import { IconAgent, IconExternal, IconFolders, IconInbox, IconPhone, IconSparkles, IconTerminal } from "./Icons.jsx";
import { parseVersion, selectReleaseNotes } from "../lib/whatsNew.js";

const ICONS = {
  agent: IconAgent,
  inbox: IconInbox,
  phone: IconPhone,
  terminal: IconTerminal,
  workspace: IconFolders,
};

function changelogURL(version) {
  const parsed = parseVersion(version);
  return parsed ? "https://github.com/cfpperche/picode/releases/tag/v" + parsed.join(".") : "https://github.com/cfpperche/picode/releases";
}

function releaseLabel(release) {
  return "v" + release.version;
}

export default function WhatsNew({ open, onClose, currentSemver, seenVersion = "", notes = [], unseenOnly = false }) {
  const releases = selectReleaseNotes(notes, currentSemver, unseenOnly ? seenVersion : "");
  const latest = releases[0] || null;
  const changelog = changelogURL(currentSemver);

  return (
    <Dialog.Root open={!!open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay wn-overlay" />
        <Dialog.Content className="dlg dlg-whats-new" onCloseAutoFocus={(e) => e.preventDefault()}>
          <div className="wn-head">
            <span className="wn-mark" aria-hidden="true"><IconSparkles size={17} /></span>
            <div className="wn-heading">
              <Dialog.Title className="wn-title">What’s new in PiCode</Dialog.Title>
              <Dialog.Description className="wn-description">
                {latest ? "Highlights from " + releaseLabel(latest) : "Release highlights and improvements"}
              </Dialog.Description>
            </div>
          </div>

          <div className="wn-scroll" id="whats-new-content">
            {releases.length ? releases.map((release) => (
              <section className="wn-release" key={release.version}>
                <div className="wn-release-head">
                  <h3 className="wn-release-version">{releaseLabel(release)}</h3>
                  {release.date ? <time className="wn-release-date" dateTime={release.date}>{release.date}</time> : null}
                </div>
                <div className="wn-highlights">
                  {release.highlights.map((item, index) => {
                    const Icon = ICONS[item.icon] || IconSparkles;
                    return (
                      <article className="wn-item" key={item.title + "-" + index}>
                        <span className="wn-icon" aria-hidden="true"><Icon size={15} /></span>
                        <div className="wn-item-copy">
                          <h4 className="wn-item-title">{item.title}</h4>
                          {item.summary ? <p className="wn-item-summary">{item.summary}</p> : null}
                        </div>
                      </article>
                    );
                  })}
                </div>
              </section>
            )) : (
              <div className="wn-empty">
                <p>No release notes are available for this build.</p>
                <a className="btn btn-secondary btn-sm" href={changelog} target="_blank" rel="noopener noreferrer">Open full changelog <IconExternal size={13} /></a>
              </div>
            )}
          </div>

          <div className="dlg-actions wn-actions">
            <a className="wn-full-link" href={changelog} target="_blank" rel="noopener noreferrer">Full changelog <IconExternal size={12} /></a>
            <button type="button" className="btn btn-primary btn-sm" onClick={onClose}>Got it</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
