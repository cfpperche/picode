// The tray's single writer (ADR-0142 slice 2): the health thread and the
// menu handlers all report here, and one render composes the status line
// and the tooltip — so no timer can erase what another wrote. Process-global
// behind a OnceLock: the shell is single-instance, so there is exactly one
// tray per machine.

use std::process::Child;
use std::sync::{Mutex, OnceLock};
use tauri::menu::MenuItem;

use crate::{keepalive, status};

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
    distro: Option<String>,
    // The daemon's address while the health loop gets answers from it; None
    // when it stops answering, so no reader opens a dead port.
    url: Option<String>,
    child: Option<Child>,
    task: keepalive::TaskEnsure,
}

impl Default for Inner {
    fn default() -> Self {
        Self {
            up: false,
            detail: "Starting…".to_string(),
            distro: None,
            url: None,
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
    open_on: bool,
}

pub struct Board {
    app: tauri::AppHandle,
    status_item: MenuItem<tauri::Wry>,
    open_item: MenuItem<tauri::Wry>,
    inner: Mutex<Inner>,
}

impl Board {
    pub fn new(
        app: tauri::AppHandle,
        status_item: MenuItem<tauri::Wry>,
        open_item: MenuItem<tauri::Wry>,
    ) -> Self {
        Self {
            app,
            status_item,
            open_item,
            inner: Mutex::new(Inner::default()),
        }
    }

    /// The health verdict, shown on the status line and in the tooltip.
    pub fn set_health(&self, up: bool, detail: &str) {
        let render = {
            let mut inner = self.inner.lock().expect("board");
            inner.up = up;
            inner.detail = detail.to_string();
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

    /// The health loop's current answer to "where is PiCode": set while the
    /// daemon answers, cleared the moment it does not. Windows opened from
    /// the tray read this instead of asking wsl.exe again on the event
    /// thread.
    pub fn note_url(&self, url: Option<String>) {
        self.inner.lock().expect("board").url = url;
    }

    pub fn app(&self) -> &tauri::AppHandle {
        &self.app
    }

    pub fn url(&self) -> Option<String> {
        self.inner.lock().expect("board").url.clone()
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

    fn compose(inner: &Inner) -> Render {
        Render {
            title: status::title(inner.up, &inner.detail),
            tooltip: status::tooltip(&inner.detail),
            open_on: inner.up,
        }
    }

    fn apply(&self, render: Render) {
        let _ = self.status_item.set_text(render.title);
        let _ = self.open_item.set_enabled(render.open_on);
        if let Some(tray) = self.app.tray_by_id("picode") {
            let _ = tray.set_tooltip(Some(render.tooltip));
        }
    }
}
