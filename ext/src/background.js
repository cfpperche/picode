const HOST = "com.picode.browser";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: "picode-send",
      title: "Send to PiCode",
      contexts: ["page", "selection", "link"],
    });
  });
});

chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true }).catch(() => {});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId !== "picode-send" || !tab?.windowId) return;
  try {
    await chrome.sidePanel.open({ windowId: tab.windowId });
  } catch (_) {
    /* older Chrome */
  }
});

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (!msg || msg.channel !== "picode") return;
  native(msg.payload)
    .then(sendResponse)
    .catch((err) => sendResponse({ ok: false, code: "host_missing", error: String(err) }));
  return true;
});

function native(payload) {
  return new Promise((resolve) => {
    chrome.runtime.sendNativeMessage(HOST, payload, (res) => {
      if (chrome.runtime.lastError) {
        resolve({
          ok: false,
          code: "host_missing",
          error: "Install the PiCode host.",
        });
        return;
      }
      resolve(res || { ok: false, code: "picode_down", error: "PiCode is not running." });
    });
  });
}
