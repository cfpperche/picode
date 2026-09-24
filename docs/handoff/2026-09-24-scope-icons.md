# 2026-09-24 — scope-icons: an icon on every scope chip

Owner asked for icons on the Skills chips (Global, workspace, agent). Applied to every
scope chip so the setup panes read alike: `web/shared/domain/scopeIcon.js` maps each
pane's words (machine/user/global → globe, workspace/project/local → folder, agent →
bot) and `ScopeIcon.jsx` (both apps) renders it; unknown scopes get none, and non-scope
pills (model kinds, Command/URL, Doctor filters) are untouched. Sites: Skills and its
Add dialog, Packages, Packages roles file, Connectors "Save to", CLI Settings and Models
layers, Pi Settings layers, Memory stores. Mobile Icons gained `IconGlobe`.
The icon is `inline-block` explicitly: Tailwind preflight makes svg `block`, which stacked it
above the label in chips that are not flex (Packages), caught by visual review.
Verified: scopeIcon.test.js; scratch captures of the panes, desktop and phone.
