// uia.rs — the accessibility tree for the computer tool's `snapshot`
// (ADR-0148): UI Automation's control view of one window, walked depth-first
// with a cache request so each element costs one cross-process call. The
// client object lives on the desk thread (an MTA thread of its own): UIA
// clients must never run on the thread that owns windows, and ours never
// does. Values come through the Value pattern's property, which is how an
// edit control tells its text.
use windows::core::BSTR;
use windows::Win32::System::Com::{CoCreateInstance, CLSCTX_INPROC_SERVER};
use windows::Win32::System::Variant::{VARIANT, VT_BSTR};
use windows::Win32::UI::Accessibility::{
    CUIAutomation, IUIAutomation, IUIAutomationCacheRequest, IUIAutomationElement,
    IUIAutomationTreeWalker, UIA_AutomationIdPropertyId, UIA_BoundingRectanglePropertyId,
    UIA_ControlTypePropertyId, UIA_IsEnabledPropertyId, UIA_IsOffscreenPropertyId,
    UIA_IsPasswordPropertyId, UIA_NamePropertyId, UIA_ValueValuePropertyId,
};

use picode_shell::axfmt::Node;

pub struct Uia {
    auto: IUIAutomation,
    walker: IUIAutomationTreeWalker,
    cache: IUIAutomationCacheRequest,
}

/// Runaway trees (a document with a million cells) stop here, well past
/// what a snapshot renders.
const MAX_VISITED: usize = 4_000;
/// Siblings under one parent past this count are not walked.
const MAX_SIBLINGS: usize = 1_500;

impl Uia {
    pub fn new() -> Result<Self, String> {
        unsafe {
            let auto: IUIAutomation = CoCreateInstance(&CUIAutomation, None, CLSCTX_INPROC_SERVER)
                .map_err(|e| format!("snapshot: UI Automation is not available ({e})"))?;
            let walker = auto.ControlViewWalker().map_err(|e| format!("snapshot: {e}"))?;
            let cache = auto.CreateCacheRequest().map_err(|e| format!("snapshot: {e}"))?;
            for p in [
                UIA_NamePropertyId,
                UIA_ControlTypePropertyId,
                UIA_AutomationIdPropertyId,
                UIA_BoundingRectanglePropertyId,
                UIA_IsEnabledPropertyId,
                UIA_IsOffscreenPropertyId,
                UIA_IsPasswordPropertyId,
                UIA_ValueValuePropertyId,
            ] {
                cache.AddProperty(p).map_err(|e| format!("snapshot: {e}"))?;
            }
            Ok(Uia { auto, walker, cache })
        }
    }

    /// The window's tree, depth-first in control-view order, at most
    /// `max_nodes` nodes and `max_depth` levels.
    pub fn snapshot(&self, hwnd: isize, max_nodes: usize, max_depth: u16) -> Result<Vec<Node>, String> {
        unsafe {
            let root = self
                .auto
                .ElementFromHandle(crate::desktop::hwnd(hwnd))
                .map_err(|e| format!("snapshot: the window has no accessibility tree ({e})"))?;
            let root = root.BuildUpdatedCache(&self.cache).map_err(|e| format!("snapshot: {e}"))?;
            let mut out = Vec::new();
            let mut stack: Vec<(IUIAutomationElement, u16)> = vec![(root, 0)];
            let mut visited = 0usize;
            while let Some((el, depth)) = stack.pop() {
                visited += 1;
                if visited > MAX_VISITED {
                    break;
                }
                out.push(node_of(&el, depth));
                if out.len() >= max_nodes {
                    break;
                }
                if depth >= max_depth {
                    continue;
                }
                let mut kids = Vec::new();
                if let Ok(first) = self.walker.GetFirstChildElementBuildCache(&el, &self.cache) {
                    let mut cur = Some(first);
                    while let Some(c) = cur {
                        cur = self.walker.GetNextSiblingElementBuildCache(&c, &self.cache).ok();
                        kids.push(c);
                        if kids.len() >= MAX_SIBLINGS {
                            break;
                        }
                    }
                }
                for k in kids.into_iter().rev() {
                    stack.push((k, depth + 1));
                }
            }
            Ok(out)
        }
    }
}

unsafe fn node_of(el: &IUIAutomationElement, depth: u16) -> Node {
    let text = |r: windows::core::Result<BSTR>| r.map(|b| b.to_string()).unwrap_or_default();
    let flag = |r: windows::core::Result<windows::core::BOOL>, default: bool| r.map(|b| b.as_bool()).unwrap_or(default);
    let rect = el.CachedBoundingRectangle().map(|r| (r.left, r.top, r.right, r.bottom)).unwrap_or((0, 0, 0, 0));
    let value = el
        .GetCachedPropertyValue(UIA_ValueValuePropertyId)
        .ok()
        .and_then(|v| variant_text(&v))
        .unwrap_or_default();
    Node {
        depth,
        control_type: el.CachedControlType().map(|c| c.0).unwrap_or(0),
        name: text(el.CachedName()),
        value,
        automation_id: text(el.CachedAutomationId()),
        rect,
        enabled: flag(el.CachedIsEnabled(), true),
        offscreen: flag(el.CachedIsOffscreen(), false),
        password: flag(el.CachedIsPassword(), false),
    }
}

// A VARIANT's string, read from the union the way the C header lays it out:
// the crate offers no conversion, and the Value property is always a BSTR.
fn variant_text(v: &VARIANT) -> Option<String> {
    unsafe {
        let inner = &v.Anonymous.Anonymous;
        if inner.vt != VT_BSTR {
            return None;
        }
        let b: &BSTR = &inner.Anonymous.bstrVal;
        let s = b.to_string();
        if s.is_empty() {
            None
        } else {
            Some(s)
        }
    }
}
