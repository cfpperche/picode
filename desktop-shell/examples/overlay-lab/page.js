(() => {
  document.title = 'Live page fixture';
  document.body.innerHTML = `<style>
    body { margin:0; background:#126544; color:white; font:20px system-ui; padding:32px; }
    #marker { width:100px; height:100px; background:#ffd800; animation:slide 2s linear infinite alternate; }
    @keyframes slide { to { transform:translateX(720px) rotate(180deg); } }
    input,button { font:inherit; padding:8px; }
  </style><h1>Live page</h1><p id="ticks"></p><div id="marker"></div>
  <p><input aria-label="Page input" placeholder="Type on the page"><button id="click">Page clicks: 0</button></p>`;
  let frames = 0, clicks = 0;
  function tick() { document.querySelector('#ticks').textContent = `Visible frame: ${++frames}`; requestAnimationFrame(tick); }
  tick();
  document.querySelector('#click').onclick = e => e.target.textContent = `Page clicks: ${++clicks}`;
})();
