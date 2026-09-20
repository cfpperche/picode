### Fixed

- Agent CLIs **Update** on an npm-global Pi runs `npm install -g` instead of
  `pi update`, which refuses when the package dir is not writable (a sudo
  npm install into an nvm prefix) and leaves the button failing.
