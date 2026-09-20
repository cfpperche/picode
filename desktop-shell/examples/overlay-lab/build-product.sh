#!/usr/bin/env bash
# Build the real shell against one scratch origin, without resident duties.
set -euo pipefail
cd "$(dirname "$0")/../.."
qa_port=${1:?Usage: build-product.sh SCRATCH_PORT}
[[ "$qa_port" =~ ^[0-9]+$ ]] && ((qa_port > 0 && qa_port <= 65535 && qa_port != 8445)) || {
  echo 'Use an explicit scratch port, never 8445.' >&2; exit 1;
}
qa_target_dir=${CARGO_TARGET_DIR:-"$PWD/target"}
mkdir -p "$qa_target_dir/overlay-lab"
qa_target_dir=$(cd "$qa_target_dir" && pwd)
printf '1 24 "%s"\n' "$PWD/examples/overlay-lab/app.manifest" > "$qa_target_dir/overlay-lab/app.rc"
llvm-rc /fo "$qa_target_dir/overlay-lab/app.res" "$qa_target_dir/overlay-lab/app.rc"
qa_config=$(node --input-type=commonjs - "$qa_port" <<'JS'
const fs = require('node:fs');
const cap = JSON.parse(fs.readFileSync('capabilities/default.json','utf8'));
cap.identifier = 'overlay-scratch-only';
cap.windows = ['main']; cap.webviews = ['main-content'];
cap.remote = {urls: [`http://localhost:${process.argv[2]}`]};
console.log(JSON.stringify({app:{security:{capabilities:[cap]}}}));
JS
)
# Keep the normal release/LTO profile: overriding LTO on this example alone
# produced a Windows Tokio startup access violation during the native gate.
TAURI_CONFIG="$qa_config" cargo xwin rustc --release --features overlay-qa \
  --example overlay_product_qa --target x86_64-pc-windows-msvc \
  --target-dir "$qa_target_dir" -- -C "link-arg=$qa_target_dir/overlay-lab/app.res"
# Restore generated production ACL schemas after the temporary QA build.
cargo xwin check --target x86_64-pc-windows-msvc --target-dir "$qa_target_dir"
echo "QA example only: run overlay_product_qa.exe --overlay-qa=http://localhost:$qa_port/desktop/"
