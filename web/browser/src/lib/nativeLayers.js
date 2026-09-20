// One trusted HTML document above the native pages (ADR-0161). Regions are
// input as well as paint: a transparent WebView rectangle alone eats clicks.
import { OVERLAY_SELECTORS } from "@picode/shared/domain/overlayAudit.js";
import { rectOf } from "./floatingLayers.js";

const LAYERS = [...OVERLAY_SELECTORS, '.dlg-overlay', '.palette-root', '.share-root', '[role="dialog"]', '[data-radix-dialog-overlay]', '[role="tooltip"]', '.focus-edge', '.native-layer-error'];
const CLEAR = 'picode-native-clear';

export function backgroundRGB(value) {
  const match = /^#([\da-f]{6})$/i.exec(value.trim());
  if (!match) throw new Error('Unsupported desktop background');
  return [0, 2, 4].map(i => parseInt(match[1].slice(i, i + 2), 16));
}

// CSS shadows paint outside layout boxes. Include their finite blur extent
// in the native paint region, otherwise Windows cuts off rounded shadows.
export function paintRect(rect, shadow = '') {
  let left = 0, top = 0, right = 0, bottom = 0;
  for (const part of shadow.replace(/rgba?\([^)]*\)/g, '').split(',')) {
    if (part.includes('inset')) continue;
    const lengths = [...part.matchAll(/(-?[\d.]+)px/g)].map(m => Number(m[1]));
    if (lengths.length < 2) continue;
    const [x, y, blur = 0, spread = 0] = lengths;
    const radius = Math.max(0, blur * 1.5 + spread);
    left = Math.max(left, radius - x); right = Math.max(right, radius + x);
    top = Math.max(top, radius - y); bottom = Math.max(bottom, radius + y);
  }
  return { left: rect.left-left, top: rect.top-top, right: rect.right+right, bottom: rect.bottom+bottom };
}

export function collectNativeLayers(doc, win) {
  const clear = new Set();
  const pages = [];
  for (const host of doc.querySelectorAll('.web-tab-host[data-native-page="true"]')) {
    const rect = rectOf(host, win);
    if (!rect || host.closest('[hidden]')) continue;
    pages.push(rect);
    for (let node = host; node; node = node.parentElement) clear.add(node);
  }
  const overlays = [];
  if (pages.length) {
    for (const el of doc.querySelectorAll(LAYERS.join(','))) {
      const rect = rectOf(el, win);
      if (rect && !el.closest('[hidden]')) overlays.push(paintRect(rect, win.getComputedStyle(el).boxShadow));
    }
  }
  return { pages, overlays, clear };
}

export function installNativeLayers({ window: win, document: doc }) {
  const invoke = win.__TAURI__?.core?.invoke;
  // Injected only by the shell version implementing this protocol. An older
  // shell must be upgraded together with this UI; it keeps its current path.
  if (!invoke || !win.__PICODE_LIVE_LAYERS__) return () => {};
  let frame = 0, dirty = false, sending = false, stopped = false, last = '';
  let cleared = new Set();
  let measured = new Set();
  let error = null;
  let acknowledged = new Set();
  const reportError = () => {
    if (error) return;
    error = doc.createElement('div');
    error.className = 'native-layer-error';
    error.setAttribute('role', 'alert');
    error.append(doc.createTextNode('Update failed. '));
    const retry = doc.createElement('button');
    retry.type = 'button'; retry.textContent = 'Retry';
    retry.onclick = () => { last = ''; schedule(); };
    error.append(retry);
    const address = doc.querySelector('.web-tab-surface:not([hidden]) .web-tab-address');
    const input = address?.querySelector('input');
    input?.dispatchEvent(new win.KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    if (address) {
      error.classList.add('native-layer-error-address');
      address.append(error);
    } else {
      // No address field (for example an app-only surface): keep recovery
      // in the always-painted host header, outside native page holes.
      doc.body.append(error);
    }
  };
  const send = async () => {
    frame = 0;
    if (stopped) return;
    dirty = true;
    if (sending) return;
    sending = true;
    try {
      while (dirty && !stopped) {
        dirty = false;
        const { pages, overlays, clear } = collectNativeLayers(doc, win);
        const nextMeasured = new Set(doc.querySelectorAll('.web-tab-host[data-native-page="true"],'+LAYERS.join(',')));
        for (const node of measured) if (!nextMeasured.has(node)) size.unobserve(node);
        for (const node of nextMeasured) if (!measured.has(node)) size.observe(node);
        measured = nextMeasured;
        // Follow finite opening/closing motion; an unrelated infinite spinner
        // must not turn region updates into a permanent animation loop.
        if (doc.getAnimations?.().some(a => a.playState === 'running'
          && a.effect?.getTiming().iterations !== Infinity
          && a.effect?.target?.matches?.(LAYERS.join(',')))) schedule();
        for (const node of cleared) if (!clear.has(node)) node.classList.remove(CLEAR);
        for (const node of clear) if (!cleared.has(node)) node.classList.add(CLEAR);
        cleared = clear;
        const background = backgroundRGB(win.getComputedStyle(doc.documentElement).getPropertyValue('--bg-base'));
        const payload = { pages, overlays, background };
        const fingerprint = JSON.stringify(payload);
        if (fingerprint === last) continue;
        try {
          for (const host of acknowledged) host.removeAttribute('data-native-layers-ready');
          await invoke('chrome_layers', payload);
          if (stopped) break;
          for (const host of acknowledged) host.removeAttribute('data-native-layers-ready');
          acknowledged = new Set(doc.querySelectorAll('.web-tab-host[data-native-page="true"]'));
          for (const host of acknowledged) host.setAttribute('data-native-layers-ready', 'true');
          last = fingerprint;
          if (error) { error.remove(); error = null; }
        } catch (e) {
          if (stopped) break;
          for (const host of acknowledged) host.removeAttribute('data-native-layers-ready');
          acknowledged.clear();
          console.error('native layers:', e);
          reportError();
          break;
        }
      }
    } finally {
      sending = false;
    }
  };
  function schedule() {
    if (stopped || frame) return;
    frame = win.requestAnimationFrame(send);
  }
  const observer = new win.MutationObserver(schedule);
  observer.observe(doc.documentElement, { childList: true, subtree: true, attributes: true,
    attributeFilter: ['class', 'style', 'hidden', 'data-state', 'data-native-page', 'data-theme'] });
  const size = new win.ResizeObserver(schedule);
  size.observe(doc.documentElement);
  const dismissTransient = () => {
    if (doc.querySelector('[role="dialog"][data-state="open"], [role="alertdialog"][data-state="open"]')) return;
    const { pages, overlays } = collectNativeLayers(doc, win);
    if (!pages.length || !overlays.length) return;
    const host = doc.querySelector('.web-tab-host[data-native-page="true"]');
    // Native sibling clicks never bubble through this document. Tell Radix's
    // existing outside-pointer handler where focus went; do not synthesize a
    // click or an action on a host control.
    host?.dispatchEvent(new win.PointerEvent('pointerdown', { bubbles: true, pointerType: 'mouse' }));
  };
  win.addEventListener('blur', dismissTransient);
  win.addEventListener('resize', schedule);
  doc.addEventListener('transitionend', schedule, true);
  doc.addEventListener('animationend', schedule, true);
  win.addEventListener('scroll', schedule, true);
  schedule();
  return () => {
    stopped = true; observer.disconnect(); size.disconnect();
    win.cancelAnimationFrame(frame);
    win.removeEventListener('blur', dismissTransient);
    win.removeEventListener('resize', schedule);
    doc.removeEventListener('transitionend', schedule, true);
    doc.removeEventListener('animationend', schedule, true);
    for (const host of acknowledged) host.removeAttribute('data-native-layers-ready');
    win.removeEventListener('scroll', schedule, true);
    for (const node of cleared) node.classList.remove(CLEAR);
    error?.remove();
  };
}
