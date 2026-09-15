// The WSL keepalive (ADR-0142): the child that holds the distro open so the
// VM is not reclaimed while idle. This mirrors the retired Go tray's
// contract (internal/desktop KeepaliveArgs + job_windows.go): `wsl.exe -d
// <distro> -- /bin/sleep infinity`, tied to this process with a job object
// so it cannot outlive the shell however the shell dies.

use std::io;
use std::process::{Child, Command};

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
}
