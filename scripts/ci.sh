#!/usr/bin/env bash
# scripts/ci.sh — `make ci`, with the evidence kept.
#
# A full-matrix run failed once (2026-09-12, first `make close` on the flow
# branch) and its output died with the terminal: four targets run in parallel
# with --output-sync, and all that survived in the transcript was a bare FAIL.
# The whole run now lands in var/ci-last.log (git-ignored, overwritten each
# time) and a failure points at it — instrumentation instead of retries, since
# the cause is still unknown.
set -uo pipefail
cd "$(dirname "$0")/.."

# Preserve the command result even when evidence cannot be written (ADR-0170).
delivery_receipt=$(node scripts/delivery-receipts.mjs start full-ci || true)
trap 'delivery_rc=$?; node scripts/delivery-receipts.mjs finish "$delivery_receipt" "$delivery_rc" || true; exit "$delivery_rc"' EXIT

mkdir -p var
log=var/ci-last.log
make --no-print-directory -j4 --output-sync=target ci-gates 2>&1 | tee "$log"
rc=${PIPESTATUS[0]}
if [ "$rc" -ne 0 ]; then
  echo
  echo "ci: FAILED — the whole run is in $log (make ci writes it; the parallel targets interleave nothing)"
fi
exit "$rc"
