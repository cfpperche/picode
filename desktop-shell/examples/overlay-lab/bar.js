(() => {
  document.body.innerHTML = `<style>body { margin:0; background:#eef1f7; padding:14px; display:flex; gap:8px; font:16px system-ui; } input,button { height:36px; box-sizing:border-box; font:inherit; } input { width:430px; }</style>
    <input aria-label="Address" placeholder="Type an address"><button data-mode="suggest">Suggestions</button><button data-mode="menu">Menu</button><button data-mode="modal">Modal</button><button data-mode="close">Close</button>`;
  let seq = 0;
  window.openOverlay = mode => document.title = `lab:${mode}:${++seq}`;
  document.querySelector('input').onfocus = () => openOverlay('suggest');
  for(const button of document.querySelectorAll('button')) button.onclick = () => openOverlay(button.dataset.mode);
  addEventListener('keydown', e => { if(e.key === 'Escape') openOverlay('close'); });
})();
