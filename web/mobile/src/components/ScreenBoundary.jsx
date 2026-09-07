import { Component } from "react";

export function ScreenLoading() {
  return <section className="m-screen m-route-state" aria-busy="true" aria-label="Loading screen">{[0, 1, 2].map(i => <div className="m-loading-row" key={i} aria-hidden="true"><span /><div><i /><i /></div></div>)}</section>;
}

export function ScreenError({ message = "Couldn’t open this screen.", onRetry }) {
  return <section className="m-screen m-route-state" role="alert"><p>{message}</p><button type="button" className="btn btn-primary" onClick={onRetry}>Try again</button></section>;
}

// A missing chunk after an update or a dropped connection must leave a way
// back in. A new route remounts this boundary; retry fetches fresh HTML/assets.
export default class ScreenBoundary extends Component {
  state = { failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  render() {
    if (this.state.failed) return <section className="m-screen m-route-state" role="alert"><p>Couldn’t open this screen.</p><button type="button" className="btn btn-primary" onClick={() => location.reload()}>Try again</button></section>;
    return this.props.children;
  }
}
