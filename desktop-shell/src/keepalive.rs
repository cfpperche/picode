// The WSL keepalive. Since ADR-0154 the primary holder is the scheduled
// task PiCodeDistro — `wsl.exe [-d <distro>] --exec /bin/sleep infinity`,
// registered with the Task Scheduler — whose lifetime is independent of
// this process, so a shell crash, upgrade or taskkill cannot strand the
// distro into WSL's idle reclaim. The child below (ADR-0142's shape: the
// retired Go tray's KeepaliveArgs + job_windows.go contract, tied to this
// process with a job object so it cannot outlive the shell) remains only
// as the fallback for a machine where the task machinery fails.

use std::io;
use std::process::{Child, Command};
#[cfg(windows)]
use picode_shell::b64;

/// The scheduled task that holds the distro open (ADR-0154). Its action is
/// `wsl.exe [-d <distro>] --exec /bin/sleep infinity`; at logon it starts
/// itself, it never times out, and a second start while running is a no-op
/// (IgnoreNew) — which makes `schtasks /run` the universal ensure verb.
pub const TASK_NAME: &str = "PiCodeDistro";

/// The task action for one distro: (execute, arguments). The action runs
/// wsl.exe under a headless conhost — a scheduled task with a plain
/// console-app action allocates a visible console window in the user's
/// session (the defect the owner reported on 2026-09-18), and S4U needs
/// admin. Some(d) pins `-d`; None uses WSL's own default — the safe
/// choice before the shell has discovered the distro, and what a foreign
/// ensure (scripts/desktop-swap.sh) registers.
pub fn task_action(distro: Option<&str>) -> (String, String) {
    match distro {
        Some(d) if plain_distro(d) => (
            CONHOST.to_string(),
            format!("--headless wsl.exe -d {d} --exec /bin/sleep infinity"),
        ),
        _ => (
            CONHOST.to_string(),
            "--headless wsl.exe --exec /bin/sleep infinity".to_string(),
        ),
    }
}

const CONHOST: &str = "C:\\Windows\\System32\\conhost.exe";

/// Distro names ride inside a PowerShell registration payload, so only the
/// plain shapes a name can have pass through; anything else degrades to
/// WSL's default distro rather than risk the payload.
fn plain_distro(distro: &str) -> bool {
    !distro.is_empty()
        && distro
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '-' | '_' | '.' | ' '))
}

/// Registers and starts the keepalive task, remembering which action it
/// registered so an unchanged poll only runs the task. A failed run (the
/// task was deleted under us, the machine's task store was reset) forgets
/// the registration; the next ensure rebuilds it.
#[derive(Default)]
pub struct TaskEnsure {
    registered_action: Option<String>,
}

impl TaskEnsure {
    pub fn ensure(&mut self, distro: Option<&str>) -> bool {
        let (execute, arguments) = task_action(distro);
        let action = format!("{execute} {arguments}");
        if self.registered_action.as_deref() != Some(action.as_str()) {
            if !register(&execute, &arguments) {
                return false;
            }
            self.registered_action = Some(action);
        }
        if run() {
            true
        } else {
            self.registered_action = None;
            false
        }
    }
}

#[cfg(windows)]
fn register(execute: &str, arguments: &str) -> bool {
    // Register-ScheduledTask works non-elevated for the current user; the
    // schtasks CLI cannot (onlogon needs admin, /sd is locale-brittle, and
    // S4U — the other way to hide the window — needs admin too). Settings
    // mirror internal/desktop's ApplyPolicy: no time limit, run on
    // battery, IgnoreNew, restart on failure three times a minute apart.
    let script = format!(
        "$ErrorActionPreference='Stop'; try {{ \
          $a = New-ScheduledTaskAction -Execute '{execute}' -Argument '{arguments}'; \
          $t = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME; \
          $s = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries \
            -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew \
            -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -StartWhenAvailable; \
          Register-ScheduledTask -TaskName '{TASK_NAME}' -Action $a -Trigger $t -Settings $s -Force | Out-Null; \
          Start-ScheduledTask -TaskName '{TASK_NAME}' \
        }} catch {{ exit 1 }}"
    );
    let mut cmd = Command::new("powershell");
    cmd.args(["-NoProfile", "-EncodedCommand", &encoded(&script)]);
    hide_console(&mut cmd);
    null_stdio(&mut cmd);
    cmd.status().map(|s| s.success()).unwrap_or(false)
}

#[cfg(not(windows))]
fn register(_action: &str) -> bool {
    false
}

#[cfg(windows)]
fn run() -> bool {
    let mut cmd = Command::new("schtasks");
    cmd.args(["/run", "/tn", TASK_NAME]);
    hide_console(&mut cmd);
    null_stdio(&mut cmd);
    cmd.status().map(|s| s.success()).unwrap_or(false)
}

#[cfg(not(windows))]
fn run() -> bool {
    false
}

// UTF-16LE base64 — the payload shape PowerShell's -EncodedCommand expects,
// quoting-proof where an inline -Command would not be.
#[cfg(windows)]
fn encoded(script: &str) -> String {
    let mut utf16: Vec<u8> = Vec::with_capacity(script.len() * 2);
    for unit in script.encode_utf16() {
        utf16.extend_from_slice(&unit.to_le_bytes());
    }
    b64::encode(&utf16)
}

#[cfg(windows)]
fn null_stdio(cmd: &mut Command) {
    cmd.stdin(std::process::Stdio::null());
    cmd.stdout(std::process::Stdio::null());
    cmd.stderr(std::process::Stdio::null());
}

/// The keepalive command line for one distro, without the launcher itself.
pub fn args(distro: &str) -> Vec<String> {
    vec![
        "-d".to_string(),
        distro.to_string(),
        "--".to_string(),
        "/bin/sleep".to_string(),
        "infinity".to_string(),
    ]
}

/// Spawns the keepalive for the distro. A job-object failure is a warning,
/// not an error: a stray sleep is better than no keepalive at all.
pub fn start(distro: &str) -> io::Result<Child> {
    let mut cmd = Command::new(wsl_exe());
    cmd.args(args(distro));
    hide_console(&mut cmd);
    // No pipes: the child must never block on a buffer nobody drains.
    cmd.stdin(std::process::Stdio::null());
    cmd.stdout(std::process::Stdio::null());
    cmd.stderr(std::process::Stdio::null());
    let child = cmd.spawn()?;
    if let Err(e) = supervise(&child) {
        eprintln!("keepalive: supervision failed ({e}); the child still holds the distro");
    }
    Ok(child)
}

fn wsl_exe() -> &'static str {
    "wsl.exe"
}

#[cfg(windows)]
fn hide_console(cmd: &mut Command) {
    use std::os::windows::process::CommandExt;
    const CREATE_NO_WINDOW: u32 = 0x0800_0000;
    cmd.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(not(windows))]
fn hide_console(_cmd: &mut Command) {}

// A job object terminates its processes when the last handle closes, and the
// handle closes when this process dies — taskkill /F, crash, Task Manager.
// The handle is deliberately never closed; closing it early would kill the
// children immediately.
#[cfg(windows)]
fn supervise(child: &Child) -> Result<(), String> {
    use std::sync::OnceLock;
    use windows::Win32::Foundation::{CloseHandle, HANDLE};
    use windows::Win32::System::JobObjects::{
        AssignProcessToJobObject, CreateJobObjectW, SetInformationJobObject,
        JOBOBJECT_EXTENDED_LIMIT_INFORMATION, JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
        JobObjectExtendedLimitInformation,
    };
    use windows::Win32::System::Threading::{OpenProcess, PROCESS_SET_QUOTA, PROCESS_TERMINATE};

    // The job handle crosses into a static, so it needs Send+Sync it does
    // not have. Safe: the handle is written once, never closed, used only
    // for AssignProcessToJobObject, and the kernel object outlives us.
    struct JobHandle(HANDLE);
    unsafe impl Send for JobHandle {}
    unsafe impl Sync for JobHandle {}
    static JOB: OnceLock<JobHandle> = OnceLock::new();

    let job = &JOB.get_or_init(|| unsafe {
        let handle = CreateJobObjectW(None, None).unwrap_or_default();
        if handle.is_invalid() {
            return JobHandle(HANDLE::default());
        }
        let mut info = JOBOBJECT_EXTENDED_LIMIT_INFORMATION::default();
        info.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
        let ok = SetInformationJobObject(
            handle,
            JobObjectExtendedLimitInformation,
            &info as *const _ as *const std::ffi::c_void,
            std::mem::size_of_val(&info) as u32,
        );
        if ok.is_err() {
            let _ = CloseHandle(handle);
            return JobHandle(HANDLE::default());
        }
        JobHandle(handle)
    });
    if job.0.is_invalid() {
        return Err("could not create the job object".to_string());
    }
    unsafe {
        // The pid cannot have been reused: the child is still running and
        // this process holds a handle to it.
        let proc = OpenProcess(PROCESS_SET_QUOTA | PROCESS_TERMINATE, false, child.id())
            .map_err(|e| format!("open child {}: {e}", child.id()))?;
        let assigned = AssignProcessToJobObject(job.0, proc).map_err(|e| {
            let _ = CloseHandle(proc);
            format!("assign child {} to the job: {e}", child.id())
        });
        let _ = CloseHandle(proc);
        assigned.map_err(|e| e.to_string())
    }
}

#[cfg(not(windows))]
fn supervise(_child: &Child) -> Result<(), String> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn keepalive_is_a_sleep_without_a_window_or_shell() {
        assert_eq!(
            args("Ubuntu"),
            vec!["-d", "Ubuntu", "--", "/bin/sleep", "infinity"]
        );
    }

    #[test]
    fn distro_names_with_spaces_stay_one_argument() {
        let argv = args("my distro");
        assert_eq!(argv[1], "my distro");
        assert_eq!(argv.len(), 5);
    }

    #[test]
    fn task_action_wraps_wsl_in_a_headless_conhost() {
        // A plain console-app action allocates a visible console window in
        // the user's session; the headless conhost is what keeps the task
        // windowless (S4U needs admin).
        assert_eq!(
            task_action(Some("Ubuntu")),
            (
                "C:\\Windows\\System32\\conhost.exe".to_string(),
                "--headless wsl.exe -d Ubuntu --exec /bin/sleep infinity".to_string()
            )
        );
    }

    #[test]
    fn task_action_falls_back_to_the_default_distro() {
        assert_eq!(
            task_action(None),
            (
                "C:\\Windows\\System32\\conhost.exe".to_string(),
                "--headless wsl.exe --exec /bin/sleep infinity".to_string()
            )
        );
    }

    #[test]
    fn task_action_never_carries_an_unplain_name_into_the_payload() {
        // A name shaped like this would be a quoting or injection risk
        // inside the registration payload; it degrades to WSL's default
        // distro instead.
        assert_eq!(
            task_action(Some("Ubuntu'; Remove-Item C:\\")),
            (
                "C:\\Windows\\System32\\conhost.exe".to_string(),
                "--headless wsl.exe --exec /bin/sleep infinity".to_string()
            )
        );
        assert_eq!(
            task_action(Some("")),
            (
                "C:\\Windows\\System32\\conhost.exe".to_string(),
                "--headless wsl.exe --exec /bin/sleep infinity".to_string()
            )
        );
    }
}
