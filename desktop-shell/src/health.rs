// The health probe (ADR-0142): asks the daemon whether it is up, the way the
// retired Go tray did — `GET {url}/api/health`, answered over loopback with
// no session. Transport is the curl Windows already ships (System32 since
// 1803), not a new TLS dependency: `-k` is a liveness check against a known
// local process, not a trust decision — the browser still gets the mkcert CA.

use serde::Deserialize;
use std::process::Command;

/// What a healthy daemon answers.
#[derive(Debug, PartialEq)]
pub struct Health {
    pub status: String,
    pub boot_id: String,
}

#[derive(Deserialize)]
struct HealthJson {
    #[serde(default)]
    status: String,
    #[serde(rename = "bootId", default)]
    boot_id: String,
}

/// Parses one probe body. The boot id is required: without it a restarted
/// daemon is indistinguishable from a settled one.
pub fn parse(text: &str) -> Result<Health, String> {
    let body: HealthJson =
        serde_json::from_str(text).map_err(|e| format!("health is not JSON: {e}"))?;
    if body.boot_id.is_empty() {
        return Err("health has no bootId".to_string());
    }
    Ok(Health {
        status: body.status,
        boot_id: body.boot_id,
    })
}

/// Probes the daemon at base (no trailing path). Fails closed: any transport
/// or parse failure means "not answering", never "up".
pub fn fetch(base: &str) -> Result<Health, String> {
    let url = format!("{}/api/health", base.trim_end_matches('/'));
    let mut cmd = Command::new(curl_exe());
    cmd.args(["-sk", "--max-time", "5", &url]);
    hide_console(&mut cmd);
    let out = cmd.output().map_err(|e| format!("curl: {e}"))?;
    if !out.status.success() {
        let detail = String::from_utf8_lossy(&out.stderr);
        return Err(format!("curl exit {}: {}", code(&out), detail.trim()));
    }
    parse(&String::from_utf8_lossy(&out.stdout))
}

/// Who is working, for the compact interlock: the same question `picode
/// deploy` asks, because stopping the distro ends the same work. Empty
/// means ready.
pub fn deploy_ready(base: &str) -> Result<Vec<String>, String> {
    let url = format!("{}/api/deploy/readiness", base.trim_end_matches('/'));
    let mut cmd = Command::new(curl_exe());
    cmd.args(["-sk", "--max-time", "5", &url]);
    hide_console(&mut cmd);
    let out = cmd.output().map_err(|e| format!("curl: {e}"))?;
    if !out.status.success() {
        return Err(format!("readiness curl exit {}", code(&out)));
    }
    parse_readiness(&String::from_utf8_lossy(&out.stdout))
}

#[derive(Deserialize)]
struct ReadinessJson {
    #[serde(default)]
    ready: bool,
    #[serde(default)]
    busy: Vec<BusyJson>,
}

#[derive(Deserialize)]
struct BusyJson {
    #[serde(default)]
    kind: String,
    #[serde(default)]
    id: String,
    #[serde(default)]
    name: String,
}

fn parse_readiness(text: &str) -> Result<Vec<String>, String> {
    let body: ReadinessJson =
        serde_json::from_str(text).map_err(|e| format!("readiness is not JSON: {e}"))?;
    if body.ready {
        return Ok(vec![]);
    }
    Ok(body
        .busy
        .iter()
        .map(|b| {
            let name = if b.name.is_empty() { &b.id } else { &b.name };
            if b.kind.is_empty() {
                name.to_string()
            } else {
                format!("{} {name}", b.kind)
            }
        })
        .collect())
}

fn code(out: &std::process::Output) -> String {
    out.status.code().map(|c| c.to_string()).unwrap_or_else(|| "signal".to_string())
}

#[cfg(windows)]
fn curl_exe() -> &'static str {
    "curl.exe"
}

#[cfg(not(windows))]
fn curl_exe() -> &'static str {
    "curl"
}

#[cfg(windows)]
fn hide_console(cmd: &mut Command) {
    use std::os::windows::process::CommandExt;
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    cmd.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(windows))]
fn hide_console(_cmd: &mut Command) {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn healthy_body_parses() {
        let h = parse(r#"{"status":"ok","bootId":"abc-123"}"#).expect("healthy");
        assert_eq!(
            h,
            Health {
                status: "ok".to_string(),
                boot_id: "abc-123".to_string()
            }
        );
    }

    #[test]
    fn missing_boot_id_is_not_healthy() {
        assert!(parse(r#"{"status":"ok"}"#).is_err());
    }

    #[test]
    fn garbage_is_not_healthy() {
        assert!(parse("not json").is_err());
        assert!(parse("").is_err());
    }

    #[test]
    fn extra_fields_are_ignored() {
        let h = parse(r#"{"status":"ok","bootId":"b","port":8445}"#).expect("extra");
        assert_eq!(h.boot_id, "b");
    }

    #[test]
    fn closed_port_fails_closed() {
        // Port 9 (discard) answers nowhere; loopback refuses fast.
        assert!(fetch("https://127.0.0.1:9").is_err());
    }

    #[test]
    fn readiness_names_who_is_working() {
        let busy = parse_readiness(
            r#"{"ready":false,"busy":[{"kind":"agent","id":"a1","name":"Work"},{"kind":"terminal","id":"t2","name":""}]}"#,
        )
        .expect("readiness");
        assert_eq!(busy, vec!["agent Work", "terminal t2"]);
        let ready =
            parse_readiness(r#"{"ready":true,"busy":[]}"#).expect("ready");
        assert!(ready.is_empty());
        assert!(parse_readiness("nope").is_err());
    }
}
