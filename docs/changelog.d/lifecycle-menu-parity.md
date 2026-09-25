### Fixed

- **Omp, Grok, Hermes Agent and Muse Code now show the ••• lifecycle menu** (Check for updates, Reinstall, and Uninstall where the vendor offers one) on Agent CLIs, like the other CLIs. PiCode's own terminal shims for those four were mistaken for the real binary when detecting the install method, so the page reported "no managed lifecycle" and hid the menu. The shim header check now recognizes every shim PiCode writes, not only the intercept ones.
