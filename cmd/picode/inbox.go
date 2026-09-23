package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/mcptool"
)

// `picode inbox notify|ask` is the Inbox door for any guest CLI (ADR-0037,
// ADR-0154): the same two moves the pi-inbox package gives pi and
// `picode mcp` gives MCP clients, as one plain command a shell can call.
// notify files an FYI; ask files a blocking question and, with --wait,
// polls the item until the human answers in the Inbox and prints the
// answer on stdout — so a CLI with no MCP config and no pi packages can
// still reach the human instead of stalling in its transcript.

const inboxUsage = `usage: picode inbox notify --title T [--body B] [--reason R]
                file a non-blocking FYI into the Inbox; prints the item id
                picode inbox ask --question Q [--context C] [--wait] [--timeout D]
                file a blocking question. With --wait: polls every 2 s until
                you answer in the Inbox, then prints the answer on stdout.
                No default timeout — an ask-human may take hours.

Daemon discovery: --url, else PICODE_URL, else <data dir>/server.json
(the same order the pi packages and ` + "`picode mcp`" + ` use).
`

func runInbox(args []string) {
	if code := inboxMain(args, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

// inboxMain is runInbox without the exit, for tests.
func inboxMain(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(errOut, inboxUsage)
		return 2
	}
	switch args[0] {
	case "notify":
		return runInboxNotify(args[1:], out, errOut)
	case "ask":
		return runInboxAsk(args[1:], out, errOut, time.Sleep, time.Now)
	default:
		fmt.Fprintf(errOut, "picode inbox: unknown command %q\n\n%s", args[0], inboxUsage)
		return 2
	}
}

// inboxDial finds the running daemon: --url wins, then PICODE_URL /
// PICODE_TERM_URL (what PiCode's terminals carry, ADR-0056), then
// server.json in the data dir — re-read on every call, because the port
// can rebind while a guest CLI lives (ADR-0050).
func inboxDial(urlFlag string) (mcptool.Daemon, error) {
	env := mcptool.Env(os.Getenv)
	dataDir := browserhost.DataDir()
	if base := strings.TrimSpace(urlFlag); base != "" {
		// Route the flag through ResolveURL so an origin typo is rejected
		// with the same message an environment typo gets.
		b, err := mcptool.ResolveURL(mcptool.MapEnv(map[string]string{"PICODE_URL": base}), "")
		if err != nil {
			return nil, err
		}
		base = b
		return mcptool.NewHTTPDaemon(base, func() string { return mcptool.ReadToken(env, dataDir) }), nil
	}
	base, err := mcptool.ResolveURL(env, dataDir)
	if err != nil {
		return nil, err
	}
	return mcptool.NewHTTPDaemon(base, func() string { return mcptool.ReadToken(env, dataDir) }), nil
}

// inboxIdentity is pi-inbox's agentIdentity (ADR-0134's assertion model):
// PiCode names managed agents and its own terminals from the environment a
// guest CLI inherited; anything else files as system with an honest
// user@host, so the Inbox still shows where the question came from.
func inboxIdentity(env func(string) string) (kind, source string) {
	if id := strings.TrimSpace(env("PICODE_AGENT_ID")); id != "" {
		return "agent", id
	}
	if id := strings.TrimSpace(env("PICODE_TERM_ID")); id != "" {
		return "terminal", id
	}
	return "system", inboxWho()
}

// inboxWho names an unmanaged caller the way a shell prompt would.
// Errors degrade to the same shape the pi packages use for a caller
// nobody can name ("pi (unmanaged)", "mcp (unmanaged)").
func inboxWho() string {
	usr, uerr := user.Current()
	host, herr := os.Hostname()
	switch {
	case uerr == nil && herr == nil && usr.Username != "":
		return usr.Username + "@" + host
	case uerr == nil && usr.Username != "":
		return usr.Username
	case herr == nil:
		return host
	}
	return "picode (unmanaged)"
}

// inboxPayload is the wire shape every Inbox filer uses — packages/pi-inbox
// buildNotifyPayload / buildAskPayload and internal/mcptool's twins, copied
// exactly (the daemon defaults the allowed verbs from the kind).
type inboxPayload struct {
	Kind       string `json:"kind"`
	SourceKind string `json:"sourceKind"`
	SourceID   string `json:"sourceId"`
	Reason     string `json:"reason"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Blocking   bool   `json:"blocking"`
}

const (
	inboxMaxTitle = 200
	inboxMaxBody  = 100_000
)

func inboxClip(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

func inboxNotifyPayload(kind, source, title, body, reason string) (inboxPayload, error) {
	title = inboxClip(strings.TrimSpace(title), inboxMaxTitle)
	if title == "" {
		return inboxPayload{}, fmt.Errorf("notify needs a --title")
	}
	if reason = strings.TrimSpace(reason); reason == "" {
		reason = "agent notification"
	}
	return inboxPayload{
		Kind: "fyi", SourceKind: kind, SourceID: source,
		Reason: inboxClip(reason, inboxMaxTitle), Title: title,
		Body: inboxClip(strings.TrimSpace(body), inboxMaxBody), Blocking: false,
	}, nil
}

func inboxAskPayload(kind, source, question, context string) (inboxPayload, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return inboxPayload{}, fmt.Errorf("ask needs a --question")
	}
	body := question
	if context = strings.TrimSpace(context); context != "" {
		body = question + "\n\n" + context
	}
	return inboxPayload{
		Kind: "question", SourceKind: kind, SourceID: source,
		Reason: "agent needs your input", Title: inboxClip(question, inboxMaxTitle),
		Body: inboxClip(body, inboxMaxBody), Blocking: true,
	}, nil
}

// inboxFile POSTs one payload and returns the item id the daemon minted.
func inboxFile(d mcptool.Daemon, p inboxPayload) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	status, body, err := d.Post(context.Background(), "/api/inbox", raw)
	if err != nil {
		return "", fmt.Errorf("PiCode is not reachable (%v)", err)
	}
	if status < 200 || status >= 300 {
		var parsed struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &parsed) == nil && parsed.Error != "" {
			return "", fmt.Errorf("Inbox refused the item: %s", parsed.Error)
		}
		return "", fmt.Errorf("Inbox refused the item: HTTP %d", status)
	}
	var item struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &item) != nil || item.ID == "" {
		return "", fmt.Errorf("filed, but PiCode did not name the item")
	}
	return item.ID, nil
}

func runInboxNotify(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("inbox notify", flag.ContinueOnError)
	fs.SetOutput(errOut)
	url := fs.String("url", "", "PiCode origin (default: PICODE_URL, else server.json)")
	title := fs.String("title", "", "short headline — what happened (required)")
	body := fs.String("body", "", "details, markdown")
	reason := fs.String("reason", "", "why the human is seeing this")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	// What the caller typed is judged before the daemon is looked for, the
	// way `inbox ask` already does it. The other order told someone who
	// forgot --title that "PiCode is not running", which is both the wrong
	// diagnosis and the wrong exit code — and it made the usage test pass
	// on a developer's machine, where a daemon answers, while failing on
	// every clean runner.
	kind, source := inboxIdentity(mcptool.Env(os.Getenv))
	p, err := inboxNotifyPayload(kind, source, *title, *body, *reason)
	if err != nil {
		fmt.Fprintln(errOut, "picode inbox:", err)
		return 2
	}
	d, err := inboxDial(*url)
	if err != nil {
		fmt.Fprintf(errOut, "picode inbox: PiCode is not running (%v) — start it, or point --url / PICODE_URL at it\n", err)
		return 1
	}
	id, err := inboxFile(d, p)
	if err != nil {
		fmt.Fprintln(errOut, "picode inbox:", err)
		return 1
	}
	fmt.Fprintln(out, id)
	return 0
}

// runInboxAsk files a blocking question. Without --wait it prints the item
// id and returns — the answer lands in the Inbox app. With --wait it polls
// the item every 2 s (re-reading server.json each poll: the daemon can
// rebind) and prints the answer on stdout, keeping stderr for the waiting
// note so the calling CLI's tool output stays machine-clean.
func runInboxAsk(args []string, out, errOut io.Writer, sleep func(time.Duration), now func() time.Time) int {
	fs := flag.NewFlagSet("inbox ask", flag.ContinueOnError)
	fs.SetOutput(errOut)
	url := fs.String("url", "", "PiCode origin (default: PICODE_URL, else server.json)")
	question := fs.String("question", "", "the question (required)")
	context := fs.String("context", "", "what the human needs to answer well, markdown")
	wait := fs.Bool("wait", false, "poll until the human answers, then print the answer")
	timeout := fs.Duration("timeout", 0, "with --wait: give up after this long (no default — an ask-human may take hours)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *timeout != 0 && !*wait {
		fmt.Fprintln(errOut, "picode inbox: --timeout applies only with --wait")
	}
	if *timeout < 0 {
		fmt.Fprintln(errOut, "picode inbox: --timeout must be positive")
		return 2
	}
	kind, source := inboxIdentity(mcptool.Env(os.Getenv))
	p, err := inboxAskPayload(kind, source, *question, *context)
	if err != nil {
		fmt.Fprintln(errOut, "picode inbox:", err)
		return 2
	}
	first, err := inboxDial(*url)
	if err != nil {
		fmt.Fprintf(errOut, "picode inbox: PiCode is not running (%v) — start it, or point --url / PICODE_URL at it\n", err)
		return 1
	}
	id, err := inboxFile(first, p)
	if err != nil {
		fmt.Fprintln(errOut, "picode inbox:", err)
		return 1
	}
	if !*wait {
		fmt.Fprintln(out, id)
		fmt.Fprintln(out, "Answer in the Inbox app.")
		return 0
	}
	fmt.Fprintf(errOut, "waiting for your answer in the Inbox (item %s)…\n", id)
	var deadline time.Time
	if *timeout > 0 {
		deadline = now().Add(*timeout)
	}
	for {
		answered, gone := inboxPollAnswer(*url, id)
		switch {
		case answered != nil:
			fmt.Fprintln(out, mcptool.AnswerLine(answered))
			return 0
		case gone:
			fmt.Fprintln(out, "answered/removed elsewhere — the question is no longer in the Inbox")
			return 0
		}
		if !deadline.IsZero() && now().After(deadline) {
			fmt.Fprintf(errOut, "picode inbox: no answer in %s — the question is still open in the Inbox (item %s)\n", *timeout, id)
			return 1
		}
		sleep(2 * time.Second)
	}
}

// inboxPollAnswer reads the item once and reports its answer state:
// a recorded response when the item is done, gone after a delete.
// A transient dial or server error reads as "not yet".
func inboxPollAnswer(urlFlag, id string) (answered *string, gone bool) {
	d, err := inboxDial(urlFlag)
	if err != nil {
		return nil, false
	}
	status, body, err := d.Get(context.Background(), "/api/inbox/"+id+"?wait=1")
	if err != nil {
		return nil, false
	}
	if status == 404 {
		return nil, true
	}
	if status != 200 {
		return nil, false
	}
	var it struct {
		State    string  `json:"state"`
		Response *string `json:"response"`
	}
	if json.Unmarshal(body, &it) != nil {
		return nil, false
	}
	if it.State == "done" {
		return it.Response, false
	}
	return nil, false
}
