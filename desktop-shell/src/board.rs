// The tray's single writer (ADR-0142 slice 2): the health thread, the disk
// thread and the menu handlers all report here, and one render composes the
// status line, the disk line, the Give-back label and the tooltip — so no
// timer can erase what another wrote. Process-global behind a OnceLock: the
// shell is single-instance, so there is exactly one tray per machine.

use std::process::Child;
use std::sync::{Mutex, OnceLock};
use tauri::menu::MenuItem;

use crate::{diskline, keepalive, status};

static BOARD: OnceLock<Board> = OnceLock::new();

pub fn init(board: Board) {
    let _ = BOARD.set(board);
}

pub fn get() -> Option<&'static Board> {
    BOARD.get()
}

struct Inner {
    up: bool,
    detail: String,
    url: Option<String>,
    disk_line: String,
    warn: bool,
    facts: Option<diskline::Facts>,
    compacting: bool,
    distro: Option<String>,
    child: Option<Child>,
    task: keepalive::TaskEnsure,
}

impl Default for Inner {
    fn default() -> Self {
        Self {
            up: false,
            detail: "Starting…".to_string(),
            url: None,
            disk_line: String::new(),
            warn: false,
            facts: None,
            compacting: false,
            distro: None,
            child: None,
            task: keepalive::TaskEnsure::default(),
        }
    }
}

// One coherent push to the tray, computed under the lock and applied after
// it is released: the setters round-trip to the main thread, and a lock
// held across that trip would serialize every reporter behind the UI.
struct Render {
    title: String,
    tooltip: String,
    disk_title: String,
    compact_title: String,
    compact_on: bool,
    open_on: bool,
}

pub struct Board {
    app: tauri::AppHandle,
    status_item: MenuItem<tauri::Wry>,
    disk_item: MenuItem<tauri::Wry>,
    compact_item: MenuItem<tauri::Wry>,
    open_item: MenuItem<tauri::Wry>,
    inner: Mutex<Inner>,
}

impl Board {
    pub fn new(
        app: tauri::AppHandle,
        status_item: MenuItem<tauri::Wry>,
        disk_item: MenuItem<tauri::Wry>,
        compact_item: MenuItem<tauri::Wry>,
        open_item: MenuItem<tauri::Wry>,
    ) -> Self {
        Self {
            app,
            status_item,
            disk_item,
            compact_item,
            open_item,
            inner: Mutex::new(Inner::default()),
        }
    }

    /// The health verdict. The URL is stored on success and cleared on
    /// failure, so the compact flow always asks readiness of an address the
    /// probe vouched for.
    pub fn set_health(&self, up: bool, detail: &str, url: Option<&str>) {
        let render = {
            let mut inner = self.inner.lock().expect("board");
            inner.up = up;
            inner.detail = detail.to_string();
            inner.url = url.map(str::to_string);
            Self::compose(&inner)
        };
        self.apply(render);
    }

    /// The disk verdict. A missing report reads as "not read", never as a
    /// zero — the tray is the last place that should call a disk empty.
    pub fn set_disk(&self, facts: Option<diskline::Facts>) {
        let render = {
            let mut inner = self.inner.lock().expect("board");
            match &facts {
                Some(f) => {
                    let (line, warn) = diskline::line(f);
                    inner.disk_line = line;
                    inner.warn = warn;
                }
                None => {
                    inner.disk_line = "Disk: not read".to_string();
                    inner.warn = false;
                }
            }
            inner.facts = facts;
            Self::compose(&inner)
        };
        self.apply(render);
    }

    /// The re-entry guard: one compact at a time, however often the item is
    /// clicked while one runs.
    pub fn begin_compact(&self) -> bool {
        let render = {
            let mut inner = self.inner.lock().expect("board");
            if inner.compacting {
                return false;
            }
            inner.compacting = true;
            Some(Self::compose(&inner))
        };
        if let Some(render) = render {
            self.apply(render);
        }
        true
    }

    pub fn end_compact(&self) {
        let render = {
            let mut inner = self.inner.lock().expect("board");
            inner.compacting = false;
            Self::compose(&inner)
        };
        self.apply(render);
    }

    pub fn note_distro(&self, distro: &str) {
        self.inner.lock().expect("board").distro = Some(distro.to_string());
    }

    pub fn distro(&self) -> Option<String> {
        self.inner.lock().expect("board").distro.clone()
    }

    pub fn url(&self) -> Option<String> {
        self.inner.lock().expect("board").url.clone()
    }

    pub fn facts(&self) -> Option<diskline::Facts> {
        self.inner.lock().expect("board").facts.clone()
    }

    /// Starts the keepalive when the distro is known and no live child
    /// holds it. A dead child (wsl --terminate took it with the distro) is
    /// re-armed, never mourned. The scheduled task (ADR-0154) is the
    /// primary holder — it outlives this process — and the child spawn is
    /// the fallback for a machine where the task machinery fails.
    pub fn ensure_keepalive(&self) {
        let mut inner = self.inner.lock().expect("board");
        let distro = inner.distro.clone();
        if let Some(distro) = distro.as_deref() {
            if inner.task.ensure(Some(distro)) {
                return;
            }
        }
        if let Some(c) = inner.child.as_mut() {
            if matches!(c.try_wait(), Ok(None)) {
                return;
            }
            inner.child = None;
        }
        let Some(distro) = distro else {
            return;
        };
        match keepalive::start(&distro) {
            Ok(c) => inner.child = Some(c),
            Err(e) => eprintln!("keepalive: cannot hold {distro} open: {e}"),
        }
    }

    /// Restarts the keepalive after a compact terminated the distro with it.
    pub fn rearm_keepalive(&self) {
        let mut inner = self.inner.lock().expect("board");
        if let Some(mut c) = inner.child.take() {
            keepalive::stop(&mut c);
        }
        let Some(distro) = inner.distro.clone() else {
            return;
        };
        // A deliberate terminate ended the task's sleep with the distro;
        // forget the registration so the next ensure rebuilds it.
        inner.task.forget();
        if inner.task.ensure(Some(&distro)) {
            return;
        }
        match keepalive::start(&distro) {
            Ok(c) => inner.child = Some(c),
            Err(e) => eprintln!("keepalive: cannot re-arm on {distro}: {e}"),
        }
    }

    fn compose(inner: &Inner) -> Render {
        let disk_title = if inner.disk_line.is_empty() {
            "Disk: reading…".to_string()
        } else {
            inner.disk_line.clone()
        };
        let disk_part = if inner.disk_line.is_empty() || inner.disk_line == "Disk: not read" {
            None
        } else {
            Some(inner.disk_line.as_str())
        };
        let (compact_title, compact_on) =
            diskline::compact_label(inner.facts.as_ref(), inner.compacting);
        Render {
            title: status::title(inner.up, &inner.detail),
            tooltip: status::tooltip(&inner.detail, disk_part, inner.warn),
            disk_title,
            compact_title,
            compact_on,
            open_on: inner.up,
        }
    }

    fn apply(&self, render: Render) {
        let _ = self.status_item.set_text(render.title);
        let _ = self.disk_item.set_text(render.disk_title);
        let _ = self.compact_item.set_text(render.compact_title);
        let _ = self.compact_item.set_enabled(render.compact_on);
        let _ = self.open_item.set_enabled(render.open_on);
        if let Some(tray) = self.app.tray_by_id("picode") {
            let _ = tray.set_tooltip(Some(render.tooltip));
        }
    }
}
