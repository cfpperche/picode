import ScreenHeader from "./ScreenHeader.jsx";
import AppIcon from "./AppIcon.jsx";

// A native app's deep link on the phone (ADR-0109). The app's body is a
// component compiled into the desktop shell, so this screen says so in one
// line and offers the way back — the pushed screen's own Back, twice over
// (header and body), never a "needs a newer PiCode" that updating would
// not fix. Reached from #/app/<id> only: the More → Apps tile for such an
// app does not navigate.
export default function NativeAppNotice({ manifest, onBack }) {
  const name = (manifest && manifest.name) || "This app";
  return (
    <div className="m-screen m-native-notice">
      <ScreenHeader title={name} onBack={onBack} />
      <section className="m-route-state" role="status">
        <AppIcon name={manifest ? manifest.icon : ""} label={name} size={24} />
        <p>{name} is a desktop tool.</p>
        <button type="button" className="btn btn-primary" onClick={onBack}>Back</button>
      </section>
    </div>
  );
}
