fn main() {
    // The ACL needs an explicit application manifest: without it no app
    // command is permissionable, and every invoke from the remote UI dies
    // with "not allowed" (work-browser-tabs handoff). Listing the commands
    // here autogenerates `allow-<name>` / `deny-<name>` permissions which
    // the capability files can then reference.
    tauri_build::try_build(
        tauri_build::Attributes::new().app_manifest(tauri_build::AppManifest::new().commands(&[
            "btab_navigate",
            "btab_bounds",
            "btab_visibility",
            "btab_back",
            "btab_forward",
            "btab_reload",
            "btab_meta",
            "btab_screenshot",
            "btab_preview",
            "btab_print",
            "btab_zoom",
            "btab_set_zoom",
            "btab_find",
            "btab_cdp_call",
            "btab_cdp_events",
            "btab_close",
            "btab_open_external",
            "btab_set_prefs",
            "btab_clear_data",
            "btab_download_dir",
            "btab_set_download_dir",
            "btab_set_ask_download",
            "btab_open_path",
            "btab_reveal_path",
            "btab_set_permission_policy",
            "btab_permission_answer",
            "btab_set_scripts",
            "lab_open",
            "lab_navigate",
            "lab_back",
            "lab_forward",
            "lab_reload",
            "lab_current_url",
            "disk_report",
            "disk_compact",
            "disk_compact_dry_run",
            "clean_list",
            "clean_apply",
            "wslconfig_read",
            "wslconfig_write",
        ])),
    )
    .expect("failed to run tauri-build");
}
