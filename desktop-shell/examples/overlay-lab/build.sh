#!/usr/bin/env bash
# Build only the isolated lab; never launch or replace the resident shell.
set -euo pipefail
cd "$(dirname "$0")/../.."
lab_target_dir=${CARGO_TARGET_DIR:-"$PWD/target"}
mkdir -p "$lab_target_dir/overlay-lab"
lab_target_dir=$(cd "$lab_target_dir" && pwd)
printf '1 24 "%s"\n' "$PWD/examples/overlay-lab/app.manifest" > "$lab_target_dir/overlay-lab/app.rc"
llvm-rc /fo "$lab_target_dir/overlay-lab/app.res" "$lab_target_dir/overlay-lab/app.rc"
cargo xwin rustc --release --example live_overlay_lab \
  --target x86_64-pc-windows-msvc --target-dir "$lab_target_dir" -- \
  -C lto=off -C "link-arg=$lab_target_dir/overlay-lab/app.res"
