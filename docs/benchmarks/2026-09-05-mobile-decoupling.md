# Desktop/mobile decoupling references — 2026-09-05

The baseline at `522844a2` has one static React entry importing both shells.
The owner approved independent presentation with a responsive desktop and
a mobile implementation copied initially from its reachable UI. These are
architecture adaptations, not source-code copies from other projects.

| Reference | Evidence | PiCode adaptation |
|---|---|---|
| [T3 Code client runtime](https://github.com/pingdotgg/t3code/blob/761d4bac1c238ea7af4dd36b56719ad5e30771c3/packages/client-runtime/README.md) | Shared web/mobile behavior is exposed through narrow subpaths; no root export. Applications provide platform services and presentation. | Explicit shared exports for schemas, API/feed, domain helpers and tokens; application-owned UI. Verified from the pinned upstream file. |
| [Paseo architecture](https://github.com/getpaseo/paseo/blob/main/docs/architecture.md) | Protocol and daemon client packages have separate responsibilities. Its Expo app itself shares UI across platforms. | Borrow the protocol/client boundary, while making our two browser presentations independent as the owner requested. |
| [Cursor Mobile](https://cursor.com/mobile) | Public product reference for supervising and interacting with agents from a phone. | Keep Now, Inbox, work and conversations as the phone's starting scope. No claim about Cursor's private bundle architecture. |
| [npm workspaces](https://docs.npmjs.com/cli/v11/using-npm/workspaces/) | One root install links local packages and supports scoped workspace commands. | Own package/build/test scripts under each app with one lockfile. |
| [Vite 6 build guide](https://v6.vite.dev/guide/build) | Nested base paths and independent HTML roots are supported; stale dynamic imports need recovery. | Keep the installed Vite major, build each root with its own base/output and provide retry for mobile screen load failure. |
| [Tailwind source detection](https://tailwindcss.com/docs/detecting-classes-in-source-files) | Automatic detection can be disabled and sources declared explicitly. | `source(none)` plus the local `src` tree prevents sibling app classes entering generated CSS. |
| [Manifest identity](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Manifest/Reference/id) | An explicit stable `id` preserves identity when `start_url` changes. | Preserve `/?mobile=1` as identity; launch `/mobile/` and keep the worker scope. |

The package boundary and mobile loading decisions are captured in
[ADR-0072](../decisions/0072-independent-web-applications.md). Future shared
extraction should follow a concrete repeated behavior, not an effort to make
all presentation identical again.
