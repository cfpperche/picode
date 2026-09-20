# Desktop shell — compile gates

Paid 2026-09-20 (`feat/desktop-shell-gate`): `desktop-shell/**` diffs now run
`make desktop-test` (pure half, rustc host tests) **and** `make desktop-shell`
(`cargo xwin build`) in `ci-scoped`; `desktop-test` joined `ci-gates`, so
`make ci` on main compiles and tests the pure crate on every full run. Before
this, no automated gate compiled the shell — `8117fd50`'s `pub(crate)`
`site_of` call from the bin crate sat broken on `main` for a day because only
a manual `cargo xwin build` could see it.

## Debts

- [ ] Remote CI never compiles the shell's COM half: the `desktop` job in
  `.github/workflows/ci.yml` runs `make desktop-test` (plain rustc) only, and
  a `cargo xwin` job was deliberately left out — it needs the Windows SDK
  download (release.yml already pays it, pinned cargo-xwin 0.23.1 + llvm-rc).
  So a Windows-only compile error in `btab.rs`/`capture.rs` still surfaces
  only in the owner's scoped `ci-scoped` run or at release. If that ever
  bites: mirror release.yml's toolchain setup into a cached CI job.
