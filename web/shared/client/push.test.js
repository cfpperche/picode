import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { urlBase64ToUint8Array, pushBlockedReason } from "./push.js";

function env({ https = true, host = "picode.local", sw = true, pm = true, notif = true, ios = false, standalone = false, permission = "default" } = {}) {
  const win = {
    isSecureContext: https || host === "localhost",
    location: { protocol: https ? "https:" : "http:", hostname: host },
  };
  if (pm) win.PushManager = function () {};
  if (notif) win.Notification = { permission };
  const nav = { userAgent: ios ? "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)" : "Mozilla/5.0 (Linux; Android 14) Chrome/128", platform: ios ? "iPhone" : "Linux", maxTouchPoints: 5 };
  if (sw) nav.serviceWorker = {};
  return { win, nav, standalone };
}

describe("urlBase64ToUint8Array", () => {
  it("decodes url-safe base64 without padding", () => {
    const bytes = urlBase64ToUint8Array("BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4");
    assert.equal(bytes.length, 65);
    assert.equal(bytes[0], 0x04);
  });
});

describe("pushBlockedReason", () => {
  it("is empty when everything is in place", () => {
    assert.equal(pushBlockedReason(env()), "");
    assert.equal(pushBlockedReason(env({ https: false, host: "localhost" })), "");
  });
  it("names HTTPS, browser support, the iOS home-screen rule and a denied permission", () => {
    assert.match(pushBlockedReason(env({ https: false })), /HTTPS/);
    assert.match(pushBlockedReason(env({ pm: false })), /does not support/);
    assert.match(pushBlockedReason(env({ ios: true, pm: false })), /Home Screen/);
    assert.match(pushBlockedReason(env({ ios: true, standalone: false })), /Home Screen/);
    assert.equal(pushBlockedReason(env({ ios: true, standalone: true })), "");
    assert.match(pushBlockedReason(env({ permission: "denied" })), /blocked/);
  });
});
