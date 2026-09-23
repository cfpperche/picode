//! The daemon's origin in the IPC ACL, whatever port it answers on.
//!
//! The capability files list `https://localhost:8445` (and 127.0.0.1) under
//! `remote`, because that is the default port. The daemon may answer
//! anywhere in its range (8445–8455) or on a port the owner chose in
//! Settings / `PICODE_PORT`, and a page served from any other origin gets
//! every command refused — the Management window loaded, then could not
//! scan. Widening the files to `localhost:*` would hand IPC to any local
//! server the webview ever showed; instead, once the shell knows where the
//! daemon answers, it adds the same capability files again with exactly that
//! origin. Same permissions, same windows, one more origin: the daemon's.

use std::collections::HashSet;
use std::sync::Mutex;
use tauri::Manager;

/// The capability files that carry a `remote` origin, as built.
const FILES: [&str; 2] = [
    include_str!("../capabilities/default.json"),
    include_str!("../capabilities/management.json"),
];

/// The origin the files already list (https only); granting it again would
/// be a no-op. `http://…:8445` (an insecure daemon) is not listed and is
/// granted like any other origin.
const STATIC_PORT: u16 = 8445;

static GRANTED: Mutex<Option<HashSet<String>>> = Mutex::new(None);

/// Adds the daemon's origin to the ACL once per origin. Safe to call on
/// every discovery; a failure is logged and leaves the static grant intact.
pub fn grant(app: &tauri::AppHandle, url: &tauri::Url) {
    let Some(port) = url.port_or_known_default() else { return };
    let scheme = url.scheme();
    let host = url.host_str().unwrap_or("localhost");
    let loopback = matches!(host, "localhost" | "127.0.0.1");
    if scheme == "https" && port == STATIC_PORT && loopback {
        return;
    }
    let key = format!("{scheme}://{host}:{port}");
    // Held across the adds: two discoveries racing must not add one origin
    // twice, and an origin is marked granted only once every file went in,
    // so a failure is retried at the next discovery.
    let mut granted = GRANTED.lock().unwrap_or_else(|p| p.into_inner());
    let granted = granted.get_or_insert_with(HashSet::new);
    if granted.contains(&key) {
        return;
    }
    let mut ok = true;
    for file in FILES {
        match for_origin(file, scheme, host, port) {
            Some(cap) => {
                if let Err(e) = app.add_capability(cap) {
                    eprintln!("daemon acl: {key}: {e}");
                    ok = false;
                }
            }
            None => {
                eprintln!("daemon acl: a capability file did not parse");
                ok = false;
            }
        }
    }
    if ok {
        granted.insert(key);
    }
}

/// The capability file re-aimed at one origin: its own identifier suffixed
/// with the port (a runtime capability must not replace the static one), and
/// `remote.urls` set to the daemon on that port under both loopback names —
/// the same pair the file lists for 8445 — plus the host the daemon
/// advertised when it is bound to a specific address instead.
pub(crate) fn for_origin(file: &str, scheme: &str, host: &str, port: u16) -> Option<String> {
    let mut v: serde_json::Value = serde_json::from_str(file).ok()?;
    let obj = v.as_object_mut()?;
    obj.remove("$schema");
    let id = obj.get("identifier")?.as_str()?.to_string();
    obj.insert("identifier".into(), format!("{id}-port-{port}").into());
    let mut urls = vec![
        serde_json::Value::String(format!("{scheme}://localhost:{port}")),
        serde_json::Value::String(format!("{scheme}://127.0.0.1:{port}")),
    ];
    if host != "localhost" && host != "127.0.0.1" {
        // An IPv6 literal needs its brackets back in a URL.
        let h = if host.contains(':') { format!("[{host}]") } else { host.to_string() };
        urls.push(serde_json::Value::String(format!("{scheme}://{h}:{port}")));
    }
    obj.get_mut("remote")?
        .as_object_mut()?
        .insert("urls".into(), serde_json::Value::Array(urls));
    serde_json::to_string(&v).ok()
}

#[cfg(test)]
mod tests {
    use super::{for_origin, FILES};

    #[test]
    fn every_file_is_reaimed_at_exactly_the_daemon() {
        for file in FILES {
            let out = for_origin(file, "https", "localhost", 8447).expect("re-aimed");
            // Tauri's own parse of a runtime capability (what add_capability
            // runs, with an expect) must accept the output.
            out.parse::<tauri::utils::acl::capability::CapabilityFile>()
                .expect("tauri parses the runtime capability");
            let v: serde_json::Value = serde_json::from_str(&out).unwrap();
            let urls: Vec<&str> = v["remote"]["urls"]
                .as_array()
                .unwrap()
                .iter()
                .map(|u| u.as_str().unwrap())
                .collect();
            assert_eq!(urls, ["https://localhost:8447", "https://127.0.0.1:8447"]);
            assert!(v["identifier"].as_str().unwrap().ends_with("-port-8447"));
            // Same grant, not a wider one: the permissions and windows are
            // the file's own, untouched.
            let orig: serde_json::Value = serde_json::from_str(file).unwrap();
            assert_eq!(v["permissions"], orig["permissions"]);
            assert_eq!(v["windows"], orig["windows"]);
            assert_eq!(v["webviews"], orig["webviews"]);
        }
    }

    #[test]
    fn a_file_without_remote_is_not_granted() {
        assert!(for_origin(r#"{"identifier":"x","permissions":[]}"#, "https", "localhost", 8447).is_none());
    }

    #[test]
    fn a_specific_bind_host_is_granted_too() {
        let out = for_origin(FILES[1], "https", "192.168.1.20", 8446).unwrap();
        let v: serde_json::Value = serde_json::from_str(&out).unwrap();
        assert_eq!(v["remote"]["urls"][2], "https://192.168.1.20:8446");
    }
}
