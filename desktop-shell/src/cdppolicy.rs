//! ADR-0128's CDP command catalog — which methods an agent tier may reach.
//!
//! Pure: no tauri, no Windows, so the table is reviewable and testable
//! anywhere (the same split as `lib.rs` / `wslconfig.rs`). The daemon decides
//! an agent's tier — it is the one that knows agent identities; the shell
//! re-checks every method against that tier before the call reaches the
//! controller, so a tier mistake at the daemon cannot become access here.
//!
//! **Deny by default.** A method this table does not name is refused, at
//! every tier: an unnamed method is not "probably fine", it is unknown, and
//! the CDP surface grows with every WebView2 release.
//!
//! The tiers are the ADR's: `read` is screenshots, DOM/CSS/Network/DOM
//! inspection, `Emulation` and no execution path; `act` adds
//! `Runtime.evaluate`, `Input.*` and navigation; `full` adds what leaves the
//! page's own trust — downloads, clipboard permissions, PDF, file chooser and
//! stored credentials.

/// An agent's CDP access tier, ordered: `read` < `act` < `full`.
#[derive(Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Debug)]
pub enum Tier {
    Read,
    Act,
    Full,
}

impl Tier {
    pub fn parse(raw: &str) -> Result<Tier, String> {
        match raw.trim().to_ascii_lowercase().as_str() {
            "read" => Ok(Tier::Read),
            "act" => Ok(Tier::Act),
            "full" => Ok(Tier::Full),
            other => Err(format!(
                "unknown browser tier {:?} — use read, act or full",
                other
            )),
        }
    }

    pub fn name(self) -> &'static str {
        match self {
            Tier::Read => "read",
            Tier::Act => "act",
            Tier::Full => "full",
        }
    }
}

/// Methods only `full` may call. Everything that hands the page's data out of
/// the page's own trust (files, downloads, credentials, storage) or weakens
/// the profile the human shares.
const FULL: &[&str] = &[
    "Browser.grantPermissions",
    "Browser.resetPermissions",
    "Browser.setDownloadBehavior",
    "DOM.setFileInputFiles",
    "Page.handleFileChooser",
    "Page.printToPDF",
    "Page.setDownloadBehavior",
    "Page.setInterceptFileChooserDialog",
    "Storage.clearDataForOrigin",
    "Storage.clearDataForStorageKey",
    "Storage.deleteCookies",
    "Storage.overrideQuotaForOrigin",
    "Storage.setCookies",
    "WebAuthn.addCredential",
    "WebAuthn.addVirtualAuthenticator",
    "WebAuthn.clearCredentials",
    "WebAuthn.disable",
    "WebAuthn.enable",
    "WebAuthn.getCredentials",
    "WebAuthn.removeVirtualAuthenticator",
    "WebAuthn.setAutomaticPresenceSimulation",
    "WebAuthn.setUserVerified",
];

/// Methods `act` may call on top of `read`. `Runtime.evaluate` runs with the
/// page's own trust — that is the line, and it is why `read` has no evaluate
/// and is genuinely read-only.
const ACT: &[&str] = &[
    "CSS.setStyleSheetText",
    "CSS.setStyleTexts",
    "DOM.focus",
    "DOM.insertBefore",
    "DOM.removeAttribute",
    "DOM.removeNode",
    "DOM.setAttributeValue",
    "DOM.setAttributesAsText",
    "DOM.setNodeValue",
    "DOM.setOuterHTML",
    "Debugger.disable",
    "Debugger.enable",
    "Debugger.evaluateOnCallFrame",
    "Debugger.getScriptSource",
    "Debugger.pause",
    "Debugger.removeBreakpoint",
    "Debugger.resume",
    "Debugger.searchInContent",
    "Debugger.setBlackboxPatterns",
    "Debugger.setBreakpoint",
    "Debugger.setBreakpointByUrl",
    "Debugger.setBreakpointsActive",
    "Debugger.setPauseOnExceptions",
    "Debugger.setScriptSource",
    "Debugger.stepInto",
    "Debugger.stepOut",
    "Debugger.stepOver",
    "Fetch.continueRequest",
    "Fetch.continueResponse",
    "Fetch.continueWithAuth",
    "Fetch.disable",
    "Fetch.enable",
    "Fetch.failRequest",
    "Fetch.fulfillRequest",
    "Fetch.getResponseBody",
    "Fetch.takeResponseBodyAsStream",
    "Input.dispatchDragEvent",
    "Input.dispatchKeyEvent",
    "Input.dispatchMouseEvent",
    "Input.dispatchMouseWheelEvent",
    "Input.dispatchTouchEvent",
    "Input.imeSetComposition",
    "Input.insertText",
    "Input.synthesizePinchGesture",
    "Input.synthesizeScrollGesture",
    "Input.synthesizeTapGesture",
    "Network.clearBrowserCache",
    "Network.clearBrowserCookies",
    "Network.deleteCookies",
    "Network.setAcceptedEncodings",
    "Network.setBlockedURLs",
    "Network.setBypassServiceWorker",
    "Network.setCacheDisabled",
    "Network.setCookie",
    "Network.setCookies",
    "Network.setExtraHTTPHeaders",
    "Network.setUserAgentOverride",
    "Page.addScriptToEvaluateOnNewDocument",
    "Page.goBack",
    "Page.goForward",
    "Page.navigate",
    "Page.navigateToHistoryEntry",
    "Page.reload",
    "Page.removeScriptToEvaluateOnNewDocument",
    "Page.setDocumentContent",
    "Page.stopLoading",
    "Runtime.addBinding",
    "Runtime.callFunctionOn",
    "Runtime.evaluate",
    "Runtime.removeBinding",
    "Runtime.runIfWaitingForDebugger",
    "Runtime.runScript",
    "Runtime.terminateExecution",
    "Target.activateTarget",
    "Target.closeTarget",
    "Target.createBrowserContext",
    "Target.createTarget",
    "Target.disposeBrowserContext",
];

/// Methods `read` may call: what the page did and what it is made of, plus
/// the screenshot and viewport emulation. No method here executes script.
const READ: &[&str] = &[
    "Accessibility.disable",
    "Accessibility.enable",
    "Accessibility.getFullAXTree",
    "Accessibility.getPartialAXTree",
    "Accessibility.getRootAXNode",
    "Accessibility.queryAXTree",
    "Browser.getBrowserCommandLine",
    "Browser.getVersion",
    "CSS.collectClassNames",
    "CSS.disable",
    "CSS.enable",
    "CSS.getBackgroundColors",
    "CSS.getComputedStyleForNode",
    "CSS.getInlineStylesForNode",
    "CSS.getMatchedStylesForNode",
    "CSS.getMediaQueries",
    "CSS.getPlatformFontsForNode",
    "CSS.getStyleSheetText",
    "DOM.describeNode",
    "DOM.getAttributes",
    "DOM.getBoxModel",
    "DOM.getContentQuads",
    "DOM.getDocument",
    "DOM.getFlattenedDocument",
    "DOM.getNodeForLocation",
    "DOM.getOuterHTML",
    "DOM.getSearchResults",
    "DOM.hideHighlight",
    "DOM.performSearch",
    "DOM.pushNodesByBackendIdsToFrontend",
    "DOM.querySelector",
    "DOM.querySelectorAll",
    "DOM.requestChildNodes",
    "DOM.resolveNode",
    "DOMDebugger.getEventListeners",
    "Emulation.clearDeviceMetricsOverride",
    "Emulation.setDeviceMetricsOverride",
    "Emulation.setEmulatedMedia",
    "Emulation.setUserAgentOverride",
    "Log.clear",
    "Log.disable",
    "Log.enable",
    "Log.startViolationsReport",
    "Network.disable",
    "Network.emulateNetworkConditions",
    "Network.enable",
    "Network.getAllCookies",
    "Network.getCertificate",
    "Network.getCookies",
    "Network.getRequestPostData",
    "Network.getResponseBody",
    "Overlay.disable",
    "Overlay.enable",
    "Overlay.hideHighlight",
    "Overlay.highlightNode",
    "Page.bringToFront",
    "Page.captureScreenshot",
    "Page.captureSnapshot",
    "Page.createIsolatedWorld",
    "Page.disable",
    "Page.enable",
    "Page.getAppManifest",
    "Page.getFrameTree",
    "Page.getLayoutMetrics",
    "Page.getNavigationHistory",
    "Page.getResourceContent",
    "Page.getResourceTree",
    "Page.searchInResource",
    "Performance.disable",
    "Performance.enable",
    "Performance.getMetrics",
    "Performance.setTimeDomain",
    "Runtime.compileScript",
    "Runtime.disable",
    "Runtime.discardConsoleEntries",
    "Runtime.enable",
    "Runtime.getHeapUsage",
    "Runtime.getIsolateId",
    "Runtime.getProperties",
    "Runtime.globalLexicalScopeNames",
    "Runtime.queryObjects",
    "Runtime.releaseObject",
    "Runtime.releaseObjectGroup",
    "Runtime.setAsyncCallStackDepth",
    "Runtime.setCustomObjectFormatterEnabled",
    "Security.disable",
    "Security.enable",
    "Storage.getCookies",
    "Storage.getStorageKeyForFrame",
    "Storage.getUsageAndQuota",
    "Target.attachToTarget",
    "Target.detachFromTarget",
    "Target.getBrowserContexts",
    "Target.getTargetInfo",
    "Target.getTargets",
    "Target.sendMessageToTarget",
    "Target.setDiscoverTargets",
];

/// The tier a method needs, or `None` when the catalog does not name it.
/// `full` is checked first so a method that appears in two lists resolves to
/// the stricter one instead of the looser.
pub fn required_tier(method: &str) -> Option<Tier> {
    let method = method.trim();
    if FULL.contains(&method) {
        return Some(Tier::Full);
    }
    if ACT.contains(&method) {
        return Some(Tier::Act);
    }
    if READ.contains(&method) {
        return Some(Tier::Read);
    }
    None
}

/// The gate itself: does a caller with `tier` reach `method`? The error is
/// the copy the UI and the agent see, and it says what would have been
/// needed — a refusal with no reason is a bug report waiting to happen.
pub fn allows(tier: Tier, method: &str) -> Result<(), String> {
    match required_tier(method) {
        Some(required) if required <= tier => Ok(()),
        Some(required) => Err(format!(
            "{} needs the {} tier; this agent has {} — nothing was sent to the page",
            method.trim(),
            required.name(),
            tier.name()
        )),
        None => Err(format!(
            "{} is not in the browser command catalog — refused",
            method.trim()
        )),
    }
}

/// Every named method and its tier, for the policy editor and the API that
/// will publish the catalog (slice 2's daemon endpoint).
pub fn catalog() -> Vec<(&'static str, Tier)> {
    let mut all: Vec<(&'static str, Tier)> = Vec::new();
    all.extend(READ.iter().map(|m| (*m, Tier::Read)));
    all.extend(ACT.iter().map(|m| (*m, Tier::Act)));
    all.extend(FULL.iter().map(|m| (*m, Tier::Full)));
    all.sort();
    all
}

#[cfg(test)]
mod tests {
    use super::*;

    fn tiers() -> [&'static [&'static str]; 3] {
        [READ, ACT, FULL]
    }

    #[test]
    fn every_entry_is_a_well_formed_method_and_sorted() {
        for list in tiers() {
            for method in list {
                let (domain, name) = method.split_once('.').expect("Domain.method");
                assert!(!domain.is_empty() && !name.is_empty(), "{method}");
                assert!(
                    domain.chars().next().unwrap().is_ascii_uppercase(),
                    "{method} has a lower-case domain"
                );
                assert!(
                    name.chars().next().unwrap().is_ascii_lowercase(),
                    "{method} has an upper-case method name"
                );
            }
            let mut sorted: Vec<&str> = (*list).to_vec();
            sorted.sort();
            assert!(list.iter().eq(sorted.iter()), "keep the table sorted");
        }
    }

    #[test]
    fn the_three_tiers_are_disjoint() {
        let [read, act, full] = tiers();
        for (a, b) in [(read, act), (read, full), (act, full)] {
            for method in a {
                assert!(!b.contains(method), "{method} is listed twice");
            }
        }
    }

    #[test]
    fn read_is_read_only() {
        // The line the ADR draws: read has no execute path, no input, no
        // navigation.
        for method in ["Runtime.evaluate", "Runtime.callFunctionOn", "Input.dispatchMouseEvent", "Page.navigate"] {
            assert_eq!(required_tier(method), Some(Tier::Act), "{method}");
            assert!(allows(Tier::Read, method).is_err(), "{method}");
        }
        // Its own allow list still works, screenshots included.
        for method in ["Page.captureScreenshot", "DOM.getDocument", "Network.getResponseBody", "Emulation.setDeviceMetricsOverride"] {
            assert_eq!(required_tier(method), Some(Tier::Read), "{method}");
            assert!(allows(Tier::Read, method).is_ok(), "{method}");
        }
    }

    #[test]
    fn the_tier_matrix_holds_on_every_row() {
        // caller tier x the tier the catalog demands x whether the method is
        // named at all. Every row, so the table is the test.
        let rows: &[(Tier, &str, bool)] = &[
            (Tier::Read, "Page.captureScreenshot", true),
            (Tier::Read, "Runtime.evaluate", false),
            (Tier::Read, "Page.printToPDF", false),
            (Tier::Read, "Nope.nope", false),
            (Tier::Act, "Page.captureScreenshot", true),
            (Tier::Act, "Runtime.evaluate", true),
            (Tier::Act, "Page.printToPDF", false),
            (Tier::Act, "Nope.nope", false),
            (Tier::Full, "Page.captureScreenshot", true),
            (Tier::Full, "Runtime.evaluate", true),
            (Tier::Full, "Page.printToPDF", true),
            (Tier::Full, "Nope.nope", false),
        ];
        for (tier, method, allow) in rows {
            assert_eq!(
                allows(*tier, method).is_ok(),
                *allow,
                "{} + {method}",
                tier.name()
            );
        }
    }

    #[test]
    fn act_reaches_read_and_act_but_not_full() {
        assert!(allows(Tier::Act, "Page.captureScreenshot").is_ok());
        assert!(allows(Tier::Act, "Runtime.evaluate").is_ok());
        for method in ["Page.printToPDF", "DOM.setFileInputFiles", "Browser.setDownloadBehavior"] {
            assert_eq!(required_tier(method), Some(Tier::Full), "{method}");
            assert!(allows(Tier::Act, method).is_err(), "{method}");
        }
    }

    #[test]
    fn full_reaches_everything_named_and_nothing_else() {
        for (method, _) in catalog() {
            assert!(allows(Tier::Full, method).is_ok(), "{method}");
        }
        // Deny by default: a method nobody named is refused at every tier,
        // including full.
        for method in ["Page.setBypassCSP", "Network.setRequestInterception", "HeapProfiler.takeHeapSnapshot", "NotADomain.notAMethod", ""] {
            assert_eq!(required_tier(method), None, "{method}");
            assert!(allows(Tier::Full, method).is_err(), "{method}");
        }
    }

    #[test]
    fn refusals_name_the_method_and_the_tier() {
        let e = allows(Tier::Read, "Runtime.evaluate").unwrap_err();
        assert!(e.contains("Runtime.evaluate") && e.contains("act") && e.contains("read"), "{e}");
        let e = allows(Tier::Act, "Page.printToPDF").unwrap_err();
        assert!(e.contains("Page.printToPDF") && e.contains("full") && e.contains("act"), "{e}");
        let e = allows(Tier::Full, "Nope.nope").unwrap_err();
        assert!(e.contains("Nope.nope") && e.contains("catalog"), "{e}");
    }

    #[test]
    fn tier_parses_what_the_daemon_will_send() {
        assert_eq!(Tier::parse("read"), Ok(Tier::Read));
        assert_eq!(Tier::parse(" Act "), Ok(Tier::Act));
        assert_eq!(Tier::parse("FULL"), Ok(Tier::Full));
        assert!(Tier::parse("admin").is_err());
        assert!(Tier::parse("").is_err());
        assert!(Tier::Read < Tier::Act && Tier::Act < Tier::Full);
    }

    #[test]
    fn catalog_is_sorted_and_complete() {
        let all = catalog();
        assert_eq!(all.len(), READ.len() + ACT.len() + FULL.len());
        let mut sorted = all.clone();
        sorted.sort();
        assert_eq!(all, sorted);
    }
}
