package apps

// The tmux app (ADR-0133, amended 2026-09-14): the tmux Server Inspector as a
// PRIMITIVES app on the Docker mold — one manifest, View/Action over the frozen
// vocabulary, no body of its own.
//
// The first delivery shipped this surface as a native desktop component. Two
// things the owner then showed on a production screenshot moved it back:
//
//  1. a native body is a component the host keeps mounted, and hiding it is
//     the component's own job (`hidden` on its root). This one did not apply
//     it, so after its first open it rendered on top of every other tab —
//     AppSurface has carried that `hidden={!!hidden}` since always, which is
//     the structural answer;
//  2. the hand-rolled chrome was a second UI/UX standard in a shell that
//     already has one. Primitives make drift impossible: the app gets the same
//     head, tabs, filter and cards as the Docker app, and the phone gets it
//     for free.
//
// The security model is ADR-0133's, narrowed by ADR-0141: the name is the
// cheap hint the list uses, the session's own environment markers are the
// receipt read before any removal, and the removal re-verifies identity and
// refuses on any change. Its scope is now one instance: a session stamped by
// another PiCode on this machine (or, before the stamp existed, one carrying
// another instance's port) is named as such and gets no removal door, because
// "no record here" is not proof of garbage while two PiCodes share a machine.
// What changed is only who renders it: the API family /api/tmux/* is gone —
// View and Action replaced it — and with it the custom renderer, its CSS and
// its chunk.

import (
	"context"
	"fmt"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TmuxServer is the slice of the tmux manager an app may use. Defined here so
// the app never imports internal/server, and as an interface so a test can
// script the server instead of touching the developer's real sessions (the
// leak ADR-0085's neighborhood was written for).
type TmuxServer interface {
	Available() bool
	ServerSessions(ctx context.Context) ([]tmux.ServerSession, error)
	ServerInfo(ctx context.Context) tmux.ServerInfo
	SessionReceipt(ctx context.Context, name string) (tmux.SessionReceipt, error)
	KillSession(ctx context.Context, name string) error
	// MachineSockets lists every tmux server socket this instance can see
	// (ADR-0139 follow-up): the new sessions' socket, the default one, and
	// every -L name in the user's tmux directory.
	MachineSockets(ctx context.Context) []tmux.MachineSocket
	// Instance is this daemon's own identity as stamped into the sessions
	// it creates (ADR-0140) — the value another PiCode reads to tell this
	// instance's leftovers from its own.
	Instance() string
}

// Session scopes — what the UI says about whose session a row is.
const (
	scopeYours     = "yours"     // this daemon's records know the terminal or agent
	scopeUnclaimed = "unclaimed" // PiCode-shaped name, no record behind it
	scopeForeign   = "notPiCode" // the user's own tmux session
)

// Session kinds, from the name's shape only.
const (
	kindAgent    = "agent"
	kindTerminal = "terminal"
	kindOther    = "other"
)

type tmuxApp struct{}

func (tmuxApp) Manifest() Manifest {
	return Manifest{ID: "tmux", Name: "tmux", Icon: "tmux", APIVersion: APIVersion}
}

func (tmuxApp) Badge(context.Context, Host) (Badge, error) { return Badge{}, nil }

// View renders the four screens: the session inventory (""), one session's
// detail ("item/<name>"), the server facts ("server"), and the machine's
// sockets ("sockets").
func (a tmuxApp) View(ctx context.Context, h Host, path string) (View, error) {
	switch {
	case path == "server":
		return a.serverView(ctx, h)
	case path == "sockets":
		return a.socketsView(ctx, h)
	case strings.HasPrefix(path, "item/"):
		return a.sessionView(ctx, h, strings.TrimPrefix(path, "item/"))
	default:
		return a.sessionsView(ctx, h)
	}
}

func (a tmuxApp) Action(ctx context.Context, h Host, req ActionRequest) (ActionResult, error) {
	if req.Action == "refresh" {
		v, err := a.View(ctx, h, req.Path)
		return ActionResult{View: &v}, err
	}
	if req.Action != "reap" {
		return ActionResult{}, fmt.Errorf("%s: no action %q here", a.Manifest().Name, req.Action)
	}
	result, err := a.reap(ctx, h, req.Args)
	if err != nil {
		return ActionResult{}, err
	}
	// The session is gone; the answer is the refreshed inventory, not the
	// detail of something that no longer exists.
	v, err := a.sessionsView(ctx, h)
	if err != nil {
		return ActionResult{Toast: result}, err
	}
	return ActionResult{Toast: result, View: &v}, nil
}

// sessionsView is the inventory: every session on the server in three groups —
// this daemon's own, the ones no record claims, and the user's own tmux.
func (a tmuxApp) sessionsView(ctx context.Context, h Host) (View, error) {
	v := View{
		APIVersion: APIVersion,
		Title:      a.Manifest().Name,
		Tabs: []Tab{
			{ID: "sessions", Label: "Sessions", Path: ""},
			{ID: "server", Label: "Server", Path: "server"},
			{ID: "sockets", Label: "Sockets", Path: "sockets"},
		},
		Empty: "tmux is not available on this machine — agents and terminals need it to keep running after you close the window.",
	}
	if h.Tmux == nil || !h.Tmux.Available() {
		return v, nil
	}
	info := h.Tmux.ServerInfo(ctx)
	sessions, err := h.Tmux.ServerSessions(ctx)
	if err != nil {
		return v, err
	}

	v.Tabs[1].Badge = strconv.Itoa(len(sessions))

	yours := ListBlock("PiCode", "tmux-group:yours", "Every terminal and agent PiCode is running, live in tmux.")
	leftovers := ListBlock("Not in PiCode's records", "tmux-group:unclaimed",
		"PiCode-shaped sessions with no terminal or agent behind them — a leftover, or another PiCode's work on this machine.")
	foreign := ListBlock("Not PiCode's", "tmux-group:foreign", "Sessions this machine's tmux server holds for you, outside PiCode's namespace. Read-only here.")

	unclaimed := 0
	for _, s := range sessions {
		kind, scope, _, ownerName, _ := tmuxAttribution(h.Store, s.Name)
		item := tmuxListItem(s, kind, ownerName)
		switch scope {
		case scopeYours:
			item.Path = "item/" + s.Name
			yours.Items = append(yours.Items, item)
		case scopeUnclaimed:
			unclaimed++
			item.Path = "item/" + s.Name
			item.Badge = "no record"
			item.Tone = "warn"
			leftovers.Items = append(leftovers.Items, item)
		default:
			foreign.Items = append(foreign.Items, item)
		}
	}
	sort.SliceStable(yours.Items, func(i, j int) bool { return tmuxRowTitle(yours.Items[i]) < tmuxRowTitle(yours.Items[j]) })
	sort.SliceStable(leftovers.Items, func(i, j int) bool { return leftovers.Items[i].Title < leftovers.Items[j].Title })
	sort.SliceStable(foreign.Items, func(i, j int) bool { return foreign.Items[i].Title < foreign.Items[j].Title })

	yours.Meta = tmuxGroupMeta(yours.Items)
	leftovers.Meta = tmuxGroupMeta(leftovers.Items)
	foreign.Meta = tmuxGroupMeta(foreign.Items)
	v.Tabs[0].Badge = strconv.Itoa(len(sessions))

	v.Blocks = []Block{}
	if info.Running && len(sessions) == 0 {
		v.Empty = "No tmux sessions on this machine. Start an agent or open a terminal, then refresh."
		return v, nil
	}
	if !info.Running {
		v.Empty = "The tmux server is not running. Start an agent or open a terminal and PiCode brings it up."
		return v, nil
	}
	if len(yours.Items) > 0 {
		v.Blocks = append(v.Blocks, yours)
	}
	if len(leftovers.Items) > 0 {
		v.Blocks = append(v.Blocks, leftovers)
	}
	if len(foreign.Items) > 0 {
		v.Blocks = append(v.Blocks, foreign)
	}
	return v, nil
}

// sessionView is one session's own screen: its identity, the owner sentence it
// has earned, and — for a leftover only — the removal.
func (a tmuxApp) sessionView(ctx context.Context, h Host, name string) (View, error) {
	v := View{APIVersion: APIVersion, Title: name, Tabs: tmuxTabs()}
	if h.Tmux == nil || !h.Tmux.Available() {
		v.Empty = "tmux is not available on this machine."
		return v, nil
	}
	sessions, err := h.Tmux.ServerSessions(ctx)
	if err != nil {
		return v, err
	}
	var row *tmux.ServerSession
	for i := range sessions {
		if sessions[i].Name == name {
			row = &sessions[i]
			break
		}
	}
	if row == nil {
		v.Empty = "This session is no longer running."
		v.Blocks = []Block{{Type: "actions", Actions: []Action{{ID: "refresh", Label: "Back to the inventory"}}}}
		return v, nil
	}

	kind, scope, _, ownerName, _ := tmuxAttribution(h.Store, name)
	title := ownerName
	if title == "" {
		title = name
	}
	state := "running"
	if row.PaneDead {
		state = "exited"
		if row.DeadStatus >= 0 {
			state = fmt.Sprintf("exited (code %d)", row.DeadStatus)
		}
	}
	text := fmt.Sprintf("%s\n%s\nstarted %s\ncurrent command: %s\n%d window(s) · %d pane(s) · %d client(s) attached",
		name, row.SessionID, row.Created.Format(time.RFC3339), row.CurrentCommand, row.Windows, row.Panes, row.Attached)
	if row.CurrentPath != "" {
		text += "\nfolder: " + row.CurrentPath
	}
	v.Blocks = []Block{{
		Type:  "detail",
		Pane:  "detail",
		Title: title,
		Meta:  []string{kindLabel(kind), scope, state},
		Text:  &text,
		At:    row.Created.UTC().Format(time.RFC3339),
	}}

	switch scope {
	case scopeYours:
		// The session belongs to a surface that owns it: open from there.
		if kind == kindAgent {
			v.Blocks = append(v.Blocks, detailMarkdown(fmt.Sprintf("This session is agent **%s**'s live TUI. Open the agent to work in it — removing it from here would be working around its own surface.", ownerName)))
		} else {
			v.Blocks = append(v.Blocks, detailMarkdown(fmt.Sprintf("This session is the terminal **%s**. Open it from Terminals; the server inventory reads it, it does not manage it.", ownerName)))
		}
	case scopeUnclaimed:
		receipt, err := h.Tmux.SessionReceipt(ctx, name)
		if err != nil {
			v.Blocks = append(v.Blocks, detailMarkdown("This session could not be read just now ("+err.Error()+"). Refresh and try again."))
			return v, nil
		}
		if where, elsewhere := tmuxElsewhere(h, receipt); elsewhere {
			// Another instance's work is not this instance's to remove
			// (ADR-0140), and the sentence names it so the operator knows
			// where to go instead.
			v.Blocks = append(v.Blocks, detailMarkdown(fmt.Sprintf(
				"No terminal or agent in PiCode's records here claims this session, because it belongs to **another PiCode instance** — %s. That is live work on the same machine, so this instance leaves it alone: open that PiCode to manage or remove it.", where)))
			return v, nil
		}
		if receipt.Instance != "" && h.Tmux.Instance() != "" {
			v.Blocks = append(v.Blocks, detailMarkdown(
				"No terminal or agent with this name is in PiCode's records any more, and this instance is the one that created the session — a leftover, and nothing else can be relying on it."))
		} else {
			v.Blocks = append(v.Blocks, detailMarkdown(
				"No terminal or agent with this name is in PiCode's records. It may be a leftover from a previous run, or a session belonging to another PiCode on this machine — this session predates the stamp that would tell them apart, so PiCode cannot tell those apart, and nothing removes it on its own."))
		}
		if owner, door := tmuxMarkerOwner(h.Store, receipt); owner != "" {
			// The marker is authoritative and it names live work: refuse the
			// door rather than pretend the name lookup was the whole truth.
			v.Blocks = append(v.Blocks, detailMarkdown(fmt.Sprintf("This session belongs to %s, which still exists in PiCode. Remove the %s instead.", owner, door)))
			return v, nil
		}
		v.Blocks = append(v.Blocks, Block{Type: "actions", Actions: []Action{{
			ID:      "reap",
			Label:   "Remove this session",
			Danger:  true,
			Confirm: fmt.Sprintf("Remove %s? No terminal or agent in PiCode's records claims this session. Removing it stops whatever is still running inside it, and cannot be undone.", name),
			Args: map[string]string{
				"name":      name,
				"sessionId": receipt.SessionID,
				"created":   strconv.FormatInt(receipt.Created.Unix(), 10),
				"panePid":   strconv.Itoa(receipt.PanePID),
			},
		}}})
	default:
		v.Blocks = append(v.Blocks, detailMarkdown("This session is not in PiCode's namespace — it is your own tmux work. It is shown here so the server reads whole; PiCode never acts on it."))
	}
	return v, nil
}

// serverView is the server's own facts plus the absence list: terminals this
// daemon has rows for whose session is not on the server.
func (a tmuxApp) serverView(ctx context.Context, h Host) (View, error) {
	v := View{APIVersion: APIVersion, Title: a.Manifest().Name + " — server", Tabs: tmuxTabs(), Blocks: []Block{}}
	if h.Tmux == nil || !h.Tmux.Available() {
		v.Empty = "tmux is not available on this machine."
		return v, nil
	}
	info := h.Tmux.ServerInfo(ctx)
	sessions, err := h.Tmux.ServerSessions(ctx)
	if err != nil {
		return v, err
	}
	keys := "xterm (Shift+Enter reaches your agents)"
	if info.ExtendedKeysFormat != "" && info.ExtendedKeysFormat != "xterm" {
		keys = info.ExtendedKeysFormat + " — PiCode sets xterm per attach; change the server value in Preferences → Terminal"
	}
	v.Blocks = append(v.Blocks, detailMarkdown(fmt.Sprintf(
		"- **tmux** %s\n- **State** %s\n- **Socket** `%s`\n- **Sessions** %d · %d attached\n- **Clients** %d\n- **Keyboard mode** %s",
		orDash(info.Version), boolWord(info.Running, "running", "not running"), orDash(info.SocketPath),
		len(sessions), tmuxAttached(sessions), info.Clients, keys)))

	absent := tmuxAbsent(h, sessions)
	list := ListBlock("In PiCode's records, not on this server", "tmux-group:absent",
		"Terminals PiCode knows about whose session is gone — \"lost at restart\" marks the ones the flight recorder saw die in a restart window (ADR-0085).")
	for _, t := range absent {
		item := ListItem{ID: t.Session, Title: t.Name, Subtitle: t.Session, Badge: "stopped", Tone: "warn"}
		if t.LostAtRestart {
			item.Badge = "lost at restart"
		}
		list.Items = append(list.Items, item)
	}
	list.Empty = "Every terminal PiCode knows about has its session running."
	if len(list.Items) > 0 {
		v.Blocks = append(v.Blocks, list)
	}
	return v, nil
}

// reap removes ONE leftover, behind the receipt: the args were rendered with
// the view the human confirmed against, and the server re-reads tmux now — a
// session that moved since the page was drawn is refused, and a session whose
// marker names live work is refused with the surface that owns it.
func (a tmuxApp) reap(ctx context.Context, h Host, args map[string]string) (string, error) {
	name := args["name"]
	if name == "" {
		return "", fmt.Errorf("no session named")
	}
	if !tmux.OwnedSessionName(name) {
		return "", fmt.Errorf("this session was not created by PiCode, so it cannot be removed here")
	}
	if h.Tmux == nil || !h.Tmux.Available() {
		return "", fmt.Errorf("tmux is not available on this machine")
	}
	receipt, err := h.Tmux.SessionReceipt(ctx, name)
	if err != nil {
		return "", fmt.Errorf("that session is no longer running")
	}
	created, _ := strconv.ParseInt(args["created"], 10, 64)
	pid, _ := strconv.Atoi(args["panePid"])
	if receipt.SessionID != args["sessionId"] || receipt.Created.Unix() != created || receipt.PanePID != pid {
		return "", fmt.Errorf("that session changed since this page was drawn — refresh and try again")
	}
	if h.Store != nil {
		if owner, door := tmuxMarkerOwner(h.Store, receipt); owner != "" {
			return "", fmt.Errorf("that session belongs to %s, which still exists in PiCode. Remove the %s instead", owner, door)
		}
	}
	if where, elsewhere := tmuxElsewhere(h, receipt); elsewhere {
		return "", fmt.Errorf("that session belongs to another PiCode instance (%s) — remove it from that instance", where)
	}
	if err := h.Tmux.KillSession(ctx, name); err != nil {
		return "", err
	}
	if h.Store != nil {
		if err := h.Store.AppendEvent("tmux.session.reaped", nil, nil, map[string]any{
			"name":      name,
			"sessionId": receipt.SessionID,
			"created":   receipt.Created.Unix(),
			"panePid":   receipt.PanePID,
			"termId":    receipt.TermID,
			"agentId":   receipt.AgentID,
			"actor":     h.Actor,
		}); err != nil {
			// The session is gone; only the record is missing. Say so rather
			// than report a failure that did not happen.
			return "Session removed, but the audit record could not be written.", nil
		}
	}
	return "Session removed.", nil
}

// tmuxAttribution resolves a session name to its kind, scope and owner using
// this daemon's records. The name→id step is lossy (session names are
// sanitized), which is why a miss is scopeUnclaimed — an uncertainty the detail
// states and the removal re-checks against the session's own marker.
func tmuxAttribution(st *store.Store, name string) (kind, scope, ownerID, ownerName, workspaceID string) {
	switch {
	case tmux.IsShellSession(name):
		kind = kindTerminal
		if st == nil {
			return kind, scopeUnclaimed, "", "", ""
		}
		if t, err := st.GetTerminal(strings.TrimPrefix(name, tmux.ShellPrefix)); err == nil {
			return kind, scopeYours, t.ID, t.Name, t.WorkspaceID
		}
		return kind, scopeUnclaimed, "", "", ""
	case tmux.OwnedSessionName(name):
		kind = kindAgent
		if st == nil {
			return kind, scopeUnclaimed, "", "", ""
		}
		if ag, err := st.GetAgent(strings.TrimPrefix(name, tmux.Prefix)); err == nil {
			return kind, scopeYours, ag.ID, ag.Name, ag.WorkspaceID
		}
		return kind, scopeUnclaimed, "", "", ""
	default:
		return kindOther, scopeForeign, "", "", ""
	}
}

// tmuxElsewhere reports whether an unclaimed session was created by ANOTHER
// PiCode instance on this machine (ADR-0140), and names it. That is the
// difference between "a leftover nobody owns" — which this instance may
// remove — and "someone else's live work", which it may not touch: two
// PiCodes share the machine, not their stores, so "no record here" is not
// proof of garbage.
//
// The instance stamp is authoritative. A session created before the stamp
// existed falls back to the loopback address it carries: a different port on
// the same machine is a different instance (nothing else can hold that port).
// When neither can be read the answer is "no" — the session is treated as
// before, and the screen says why it cannot be sure.
func tmuxElsewhere(h Host, receipt tmux.SessionReceipt) (where string, yes bool) {
	if h.Tmux == nil {
		return "", false
	}
	ours := h.Tmux.Instance()
	if receipt.Instance != "" {
		if ours == "" || filepath.Clean(receipt.Instance) == filepath.Clean(ours) {
			return "", false
		}
		if receipt.URL != "" {
			return receipt.URL, true
		}
		return "its data directory is " + receipt.Instance, true
	}
	if ours == "" || receipt.URL == "" || h.LoopbackURL == "" {
		return "", false
	}
	mine, theirs := tmuxPort(h.LoopbackURL), tmuxPort(receipt.URL)
	if mine == "" || theirs == "" || mine == theirs {
		return "", false
	}
	return receipt.URL, true
}

// tmuxPort is the port of a loopback URL, "" when the string does not carry
// one.
func tmuxPort(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Port()
}

// tmuxMarkerOwner answers whether the session's own marker names something this
// daemon still holds, and which surface owns it when it does.
func tmuxMarkerOwner(st *store.Store, receipt tmux.SessionReceipt) (owner, door string) {
	if receipt.TermID != "" {
		if t, err := st.GetTerminal(receipt.TermID); err == nil {
			owner := t.Name
			if owner == "" {
				owner = t.ID
			}
			return "terminal " + owner, "terminal"
		}
	}
	if receipt.AgentID != "" {
		if ag, err := st.GetAgent(receipt.AgentID); err == nil {
			return "agent " + ag.Name, "agent"
		}
	}
	return "", ""
}

// tmuxAbsent lists the terminals whose expected session is not on the server.
// Terminals only: a terminal is created together with its session, so its
// absence is evidence; an agent in managed mode legitimately has none.
func tmuxAbsent(h Host, sessions []tmux.ServerSession) []tmux.AbsentTerminal {
	out := []tmux.AbsentTerminal{}
	if h.Store == nil {
		return out
	}
	live := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		live[s.Name] = true
	}
	terms, err := h.Store.ListTerminals()
	if err != nil {
		return out
	}
	for _, t := range terms {
		session := tmux.ShellSessionName(t.ID)
		if live[session] {
			continue
		}
		out = append(out, tmux.AbsentTerminal{
			ID: t.ID, Name: t.Name, Session: session,
			WorkspaceID: t.WorkspaceID, LostAtRestart: h.LostSessions[session],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LostAtRestart != out[j].LostAtRestart {
			return out[i].LostAtRestart
		}
		return out[i].Session < out[j].Session
	})
	return out
}

func tmuxTabs() []Tab {
	return []Tab{
		{ID: "sessions", Label: "Sessions", Path: ""},
		{ID: "server", Label: "Server", Path: "server"},
		{ID: "sockets", Label: "Sockets", Path: "sockets"},
	}
}

// socketsView answers the operator question ADR-0139 created: which tmux
// servers exist on this machine, what lives on each, and where this
// instance's new sessions land. What can be seen is honest and bounded —
// every socket file in the user's tmux directory plus this instance's own
// -S path; a socket elsewhere is undiscoverable and the block says so.
func (a tmuxApp) socketsView(ctx context.Context, h Host) (View, error) {
	v := View{APIVersion: APIVersion, Title: a.Manifest().Name, Tabs: tmuxTabs()}
	if h.Tmux == nil || !h.Tmux.Available() {
		v.Empty = "tmux is not available on this machine — agents and terminals need it to keep running after you close the window."
		return v, nil
	}
	socks := h.Tmux.MachineSockets(ctx)
	// Not the Collapsible ListBlock helper: this screen has one block and the
	// rows are its whole point, so it renders open (a remembered-collapsed
	// group hid the answer on first visit, seen on the scratch 2026-09-15).
	block := Block{
		Type: "list", ID: "tmux-sockets", Title: "tmux servers on this machine",
		Empty: "No tmux sockets found.",
		Items: []ListItem{},
	}
	running := 0
	for _, s := range socks {
		title := s.Name
		if s.Ours {
			title += " · this instance"
		}
		item := ListItem{ID: s.Path, Title: title, Path: "", Wrap: true, Meta: []string{s.Path}}
		switch {
		case s.Running:
			running++
			item.Badge = "running"
			item.Tone = "ok"
			if s.Sessions > 0 {
				item.Subtitle = fmt.Sprintf("%d session(s) · %d PiCode", s.Sessions, s.PicodeSessions)
			} else {
				item.Subtitle = "running, no sessions"
			}
		default:
			if s.Ours {
				// Not a leftover: this instance's server starts with the first
				// new terminal — say what it is, not what it looks like.
				item.Badge = "idle"
				item.Subtitle = "no server yet — the next terminal this instance opens starts it"
			} else {
				item.Badge = "no server"
				item.Tone = "warn"
				item.Subtitle = "nothing listening — a leftover socket file"
			}
		}
		block.Items = append(block.Items, item)
	}
	block.Meta = []string{fmt.Sprintf("%d socket(s)", len(socks))}
	v.Tabs[2].Badge = strconv.Itoa(running)
	v.Blocks = []Block{block}
	return v, nil
}

func ListBlock(title, id, empty string) Block {
	return Block{Type: "list", ID: id, Title: title, Collapsible: true, Empty: empty, Items: []ListItem{}}
}

func detailMarkdown(md string) Block { return Block{Type: "detail", Markdown: md} }

func tmuxListItem(s tmux.ServerSession, kind, ownerName string) ListItem {
	title := ownerName
	if title == "" {
		title = s.Name
	}
	item := ListItem{
		ID: s.Name, Title: title,
		Subtitle: s.CurrentCommand,
		At:       s.Created.UTC().Format(time.RFC3339),
		Meta:     []string{kindLabel(kind)},
		Wrap:     ownerName == "",
	}
	if s.Attached > 0 {
		item.Meta = append(item.Meta, fmt.Sprintf("%d client(s) attached", s.Attached))
	}
	if s.Panes > 1 {
		item.Meta = append(item.Meta, fmt.Sprintf("%d panes", s.Panes))
	}
	if s.PaneDead {
		item.Badge = "exited"
		item.Tone = "warn"
		if s.DeadStatus >= 0 {
			item.Badge = fmt.Sprintf("exited %d", s.DeadStatus)
		}
	} else {
		item.Badge = "live"
		item.Tone = "ok"
	}
	return item
}

func tmuxRowTitle(i ListItem) string { return i.Title }

func tmuxGroupMeta(items []ListItem) []string {
	if len(items) == 0 {
		return nil
	}
	attached := 0
	for _, i := range items {
		if i.Tone == "ok" {
			attached++
		}
	}
	return []string{fmt.Sprintf("%d session(s) · %d live", len(items), attached)}
}

func tmuxAttached(sessions []tmux.ServerSession) int {
	n := 0
	for _, s := range sessions {
		n += s.Attached
	}
	return n
}

func kindLabel(kind string) string {
	switch kind {
	case kindAgent:
		return "agent"
	case kindTerminal:
		return "terminal"
	}
	return "other"
}

func boolWord(b bool, whenTrue, whenFalse string) string {
	if b {
		return whenTrue
	}
	return whenFalse
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
