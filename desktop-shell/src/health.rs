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

/// Whether one daemon path answers HTTP 200. This is the readiness gate
/// for the shell's first navigation: a restart window can hold the port
/// with a listener that 404s, and navigating into that window leaves the
/// resident parked on a dead body forever (2026-09-18, back-to-back
/// deploys bricked the shell until a manual restart).
pub fn status_ok(base: &str, path: &str) -> bool {
    let url = format!("{}{}", base.trim_end_matches('/'), path);
    let mut cmd = Command::new(curl_exe());
    cmd.args(["-sk", "-o", "NUL", "-w", "%{http_code}", "--max-time", "5", &url]);
    hide_console(&mut cmd);
    matches!(cmd.output(), Ok(out) if String::from_utf8_lossy(&out.stdout).trim() == "200")
}

/// Whether Windows trusts the daemon's certificate: the same probe without
/// `-k`. curl.exe verifies through schannel, the store WebView2 reads, so
/// "curl refuses" means "the window would land on a certificate error".
/// `--ssl-no-revoke`: an mkcert root publishes no revocation list.
pub fn cert(base: &str) -> picode_shell::waitstate::Cert {
    let url = format!("{}/api/health", base.trim_end_matches('/'));
    if !url.starts_with("https://") {
        return picode_shell::waitstate::Cert::Trusted;
    }
    let mut cmd = Command::new(curl_exe());
    cmd.args(["-s", "-S", "--ssl-no-revoke", "-o", "NUL", "--max-time", "5", &url]);
    hide_console(&mut cmd);
    match cmd.output() {
        Ok(out) => picode_shell::waitstate::cert_verdict(
            out.status.code(),
            &String::from_utf8_lossy(&out.stderr),
        ),
        Err(_) => picode_shell::waitstate::Cert::Unknown,
    }
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
}

#[cfg(test)]
mod acl_tests {
    // The webview refuses a command that is registered but not allowed by
    // any capability with "not allowed by ACL" — an error no user should
    // ever see (2026-09-18: btab_clear_app_data shipped exactly so). The
    // sources are embedded, so this runs wherever the tests run.
    //
    // Permissions are kebab-case versions of the snake_case commands
    // (btab_clear_data -> allow-btab-clear-data).

    const MAIN_RS: &str = include_str!("main.rs");
    const BUILD_RS: &str = include_str!("../build.rs");
    const CAPABILITIES: &str = concat!(
        include_str!("../capabilities/default.json"),
        include_str!("../capabilities/lab.json"),
        include_str!("../capabilities/computerlab.json"),
        include_str!("../capabilities/management.json"),
        include_str!("../capabilities/waiting.json"),
    );

    // Registered but only ever called from the tray menu's own Rust handler,
    // never from a webview — so no ACL entry exists for it, and none should.
    // Every other command must be allowed by a capability (the test below
    // enforces it): the Management window's seven commands live in
    // capabilities/management.json.
    const ACL_EXCEPTIONS: [&str; 1] = [
        "computerlab_open",
    ];

    #[test]
    fn every_registered_command_is_allowed_by_the_acl() {
        let start = MAIN_RS
            .find("generate_handler!")
            .expect("the invoke_handler block must exist");
        let open = start + MAIN_RS[start..].find('[').expect("the list opens");
        let end = open + MAIN_RS[open..].find(']').expect("the list closes");
        for entry in MAIN_RS[open + 1..end].split(',') {
            let name = entry.trim().rsplit("::").next().unwrap_or("").trim();
            if name.is_empty() || ACL_EXCEPTIONS.contains(&name) {
                continue;
            }
            let permission = format!("\"allow-{}\"", name.replace('_', "-"));
            assert!(
                BUILD_RS.contains(&format!("\"{name}\"")),
                "command `{name}` is registered in generate_handler! but build.rs does not generate its permission — the ACL entry would dangle"
            );
            assert!(
                CAPABILITIES.contains(&permission),
                "command `{name}` is registered in generate_handler! but no capability allows `{permission}` — the webview refuses it with 'not allowed by ACL'"
            );
        }
    }
}
