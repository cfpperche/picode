import assert from "node:assert/strict";
import { test } from "node:test";

import {
  OPEN_EXTERNAL_COMMAND,
  SIGNIN_TARGET,
  openWindowsSignIn,
  shellInvoke,
} from "./windowsSettings.js";

test("no shell: the row is not offered and nothing is invoked", () => {
  assert.equal(shellInvoke({}), null);
  assert.equal(shellInvoke(undefined), null);
  assert.equal(shellInvoke({ __TAURI__: { core: {} } }), null);
});

test("the shell is asked for exactly one target", async () => {
  const calls = [];
  const win = {
    __TAURI__: {
      core: {
        invoke: (cmd, args) => {
          calls.push([cmd, args]);
          return Promise.resolve();
        },
      },
    },
  };
  assert.equal(await openWindowsSignIn(win), "asked");
  assert.deepEqual(calls, [[OPEN_EXTERNAL_COMMAND, { url: SIGNIN_TARGET }]]);
  assert.equal(SIGNIN_TARGET, "ms-settings:signinoptions");
});

test("a refusing shell is reported, not swallowed", async () => {
  const win = {
    __TAURI__: { core: { invoke: () => Promise.reject(new Error("refused")) } },
  };
  assert.equal(await openWindowsSignIn(win), "failed");
});
