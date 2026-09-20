# 2026-09-19 — the crop fails silently (feat/annot-crop-why)

Second Send, still `shot: ""`. The id fix was necessary but not sufficient:
`cropFromPreview` ends in a bare `catch { return "" }`, so every give-up —
the invoke rejecting, an empty capture, a decode failure, the crop missing
the frame — looked identical: no picture and no reason.

## What landed
- `cropFromPreview` returns `{ data, why }`; the Send toasts
  "The screenshot did not come through: <reason>" (the shell's own message
  when the invoke is what failed), and says when shots were simply off.
- The capture retries on a ladder (0, 1.5 s, 4 s): a floating layer over the
  tab hydrates a still and hides the native view ([data-sonner-toast] is in
  the shared vocabulary, and a pick toast lives four seconds), which is a
  concrete way to get an empty capture at Send time.
- Guards: JS source scan (15/15 on the annotate file); web build + test-js
  green.

## Next up
- Owner: desktop-restart + deploy, one Send. Either the crop arrives, or the
  toast names the failing step — that name is the next fix.

## Debts
- The capture itself is Windows-only: the reason plumbing is what this branch
  proves; the cause of the empty capture still needs the owner's next Send.
