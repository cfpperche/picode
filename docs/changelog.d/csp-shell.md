### Fixed

- The app shell now really runs under the Content-Security-Policy: `/browser/`, `/desktop/` and `/mobile/` — the URLs the launcher and the desktop shell actually load — carried no policy at all, because the header matched only `/`, `/index.html` and `*.html`. The script hash is computed from the file each path serves, so each shell's inline theme bootstrap stays allowed.

### Security

- `frame-src` (with the dev-server preview) allows PiCode's own browser surface to frame this machine's `localhost`/`127.0.0.1` pages over http(s) and nothing else; the framed page remains a separate origin with its own policy, and `frame-ancestors 'self'` keeps PiCode itself unframed by others.
