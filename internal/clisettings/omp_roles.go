package clisettings

// Omp's model roles, transcribed from the installed bundle on 2026-09-22
// (omp 18.2.8): `src/config/model-roles.ts`, `MODEL_ROLES` — the id, the tag
// its own hub prints, and the name it gives the role, in the file's own order
// (`MODEL_ROLE_IDS`: the ten chat roles, then the five model-kind roles).
//
// This is vendored knowledge about someone else's software, the same trade
// `pikeys.Catalog` and `LoginMethods` make: a row is worth what its citation is
// worth, and `TestOmpRoleCatalogMatchesTheVendor` pins the list so a rename
// lands as a red test rather than a wrong form. omp renames roles — 18.1.5
// removed `designer`, and 18.2.7 moved image, web, speech, dictation, judge and
// memory *into* this map from settings keys of their own.
//
// The help line on a row says when the CLI reaches for that role. It is the
// vendor's own description where it has one, and otherwise the role's name
// alone: a sentence PiCode invented about someone else's routing would be a
// claim nobody measured.
var ompRoleCatalog = []RoleDef{
	{ID: "default", Tag: "DEFAULT", Name: "Default", Section: "chat", Help: "The model every turn uses unless something else claims it."},
	{ID: "smol", Tag: "SMOL", Name: "Fast", Section: "chat"},
	{ID: "slow", Tag: "SLOW", Name: "Thinking", Section: "chat"},
	{ID: "vision", Tag: "VISION", Name: "Vision", Section: "chat"},
	{ID: "plan", Tag: "PLAN", Name: "Architect", Section: "chat"},
	{ID: "commit", Tag: "COMMIT", Name: "Commit", Section: "chat"},
	{ID: "tiny", Tag: "TINY", Name: "Tiny", Section: "chat"},
	{ID: "memory", Tag: "MEMORY", Name: "Memory", Section: "chat"},
	{ID: "task", Tag: "TASK", Name: "Subtask", Section: "chat"},
	{ID: "advisor", Tag: "ADVISOR", Name: "Advisor", Section: "chat"},
	{ID: "image", Tag: "IMAGE", Name: "Image generation", Section: "kind"},
	{ID: "web", Tag: "WEB", Name: "Web search", Section: "kind"},
	{ID: "speech", Tag: "SPEECH", Name: "Speech", Section: "kind"},
	{ID: "dictation", Tag: "DICTATION", Name: "Dictation", Section: "kind"},
	{ID: "judge", Tag: "JUDGE", Name: "Judge", Section: "kind"},
}

// ompRoles is the declaration the report and the writer read. Every path is
// omp's own key: `modelRoles`, `modelTags`, `cycleOrder` and
// `retry.fallbackChains` (`src/config/settings-schema.ts`, and the grammar in
// `retry.fallbackChains`' own description, read 2026-09-22).
var ompRoles = &rolesSpec{
	path:       []string{"modelRoles"},
	pane:       "models",
	catalog:    ompRoleCatalog,
	group:      groupRoles,
	tagsPath:   []string{"modelTags"},
	cyclePath:  []string{"cycleOrder"},
	cycleLabel: "Quick-switch cycle",
	// Measured, not guessed: omp's hub draws `⟳ N` beside a role from this
	// list's order (`pi-tui/src/overlays/model-hub.ts:2181`, "second stop of
	// the ctrl+p cycle"), and the key it names is the one the Keyboard pane
	// already edits.
	cycleHelp:    "The roles ctrl+p steps through, in order.",
	chainsPath:   []string{"retry", "fallbackChains"},
	chainGroup:   groupFallbacks,
	chainHelp:    "When a model fails, Omp tries the next entry in its chain. @smol means the model the SMOL role uses.",
	chainAddHint: "A role, a provider/model-id, or a provider/*",
	// The suffixes Omp accepts after a selector, in its own display order
	// (`pi-tui/src/thinking.ts`, `CLI_THINKING_LEVELS = ["off",
	// ...THINKING_EFFORTS, "auto"]`, read 2026-09-22).
	levels: []string{"off", "minimal", "low", "medium", "high", "xhigh", "max", "auto"},
}
