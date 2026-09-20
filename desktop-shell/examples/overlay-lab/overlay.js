(() => {
  document.title = 'Overlay fixture';
  document.body.innerHTML = `<style>
    html,body { margin:0; background:transparent; height:100%; font:16px system-ui; }
    body.modal { background:rgba(0,0,0,.35); display:grid; place-items:center; }
    #card { box-sizing:border-box; background:#fff; color:#111; padding:20px; border:1px solid #aab; border-radius:12px; }
    .modal #card { width:440px; }
    button,input { font:inherit; min-height:36px; }
  </style><div id="card"></div>`;
  window.mode = mode => {
    document.body.className = mode;
    document.querySelector('#card').innerHTML = mode === 'modal'
      ? '<h2>Live modal</h2><p>The green page and moving yellow marker must remain visible through the backdrop.</p><input aria-label="Modal input" placeholder="Type here"><button>Confirm</button>'
      : mode === 'menu' ? '<button>Menu item</button><details><summary>Submenu</summary><button>Nested item</button></details>'
      : '<div role="listbox"><button role="option">First suggestion</button><button role="option">Second suggestion</button></div>';
  };
  mode('suggest');
})();
