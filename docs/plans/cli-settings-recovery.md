# Native settings recovery

Status: implemented; scoped gates and browser validation passed. Follow-up to the adversarial review of
`3b49ceaa`; navigation and native persistence remain as defined by ADR-0101.

## Decision table

| Conditions | Required behavior | Validation |
|---|---|---|
| First context read fails | Show retry; no editable target | Existing browser suite |
| Loaded context, transient refresh failure | Keep draft and editor; block writes until revalidated | Browser, both apps |
| Retry still pending or fails again | Keep draft and write block | Browser, both apps |
| Key capture open before refresh failure | Pause the window listener; no write while blocked | Browser, both apps |
| Retry succeeds | Restore controls with draft intact | Browser, both apps |
| Agent/workspace is missing | Remove editable target; never fall back | Context unit tests and browser deletion |
| Native defaults fail, explicit agent settings available | Show defaults error; allow agent settings without inventing inherited values | Real malformed-file browser case |
| Agent settings PATCH fails | Show error; do not stop/start | Desktop browser matrix, existing mobile unit matrix |
| Tool mode unchanged, or agent stopped | Save without restart | Existing browser and mobile unit coverage |
| Tool mode changes, stop fails | Show partial success; do not start | Desktop browser matrix: managed and interactive |
| Tool mode changes, start fails | Show partial success; do not claim Saved | Desktop browser matrix: managed and interactive |
| Tool mode changes, restart succeeds | Show success after the full sequence | Desktop browser matrix: managed and interactive |
| Desktop Tools/Checklist popover near an edge | Radix owns placement; all options remain visible | Screenshots and overlay audit |
| Controls wrap onto multiple lines | Same control height; align within each line | Six alignment unit cases and narrow screenshots |

UI verification uses a disposable docs fixture. Runtime outcomes are injected
at the browser HTTP boundary; no real agent starts or model turns are needed.
Physical-device and real-process restart acceptance remain external.

## Evidence

`make ci-scoped` passed. `qa-cli-settings-recovery.mjs` passed 13 scenarios,
including both desktop restart modes at four outcome boundaries. The existing
native settings and mobile quick-sheet browser scripts also passed. The new
missing-context case and six wrapped-alignment cases pass with the existing
unit tests. Screenshots were inspected at verified 1365 x 1000, 560 x 1000 and
390 x 844 viewports; popover audits pass. Evidence is retained in
`var/screenshots/cli-settings-regressions/`. Integration is recorded in the
session handoff.
