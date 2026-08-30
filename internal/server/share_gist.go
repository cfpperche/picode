package server

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

func handleAgentShare(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if agent.SessionPath == nil || *agent.SessionPath == "" {
			writeErr(w, http.StatusNotFound, "no session file yet")
			return
		}
		src := *agent.SessionPath
		if _, err := os.Stat(src); err != nil {
			writeErr(w, http.StatusNotFound, "session file missing")
			return
		}
		if _, err := exec.LookPath("gh"); err != nil {
			writeErr(w, http.StatusBadRequest, "Install GitHub CLI (gh), then run gh auth login.")
			return
		}
		auth := exec.Command("gh", "auth", "status")
		if err := auth.Run(); err != nil {
			writeErr(w, http.StatusBadRequest, "GitHub CLI is not logged in. Run gh auth login.")
			return
		}
		dir, err := os.MkdirTemp("", "picode-share-*")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer os.RemoveAll(dir)
		jsonl := filepath.Join(dir, "session.jsonl")
		raw, err := os.ReadFile(src)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(jsonl, raw, 0o600); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		htmlPath := filepath.Join(dir, "session.html")
		if err := os.WriteFile(htmlPath, []byte(shareHTML(src)), 0o600); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cmd := exec.Command("gh", "gist", "create", "--public=false", "-d", "PiCode session", jsonl, htmlPath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			writeErr(w, http.StatusBadGateway, msg)
			return
		}
		gistURL := strings.TrimSpace(stdout.String())
		if i := strings.LastIndex(gistURL, "http"); i >= 0 {
			gistURL = strings.TrimSpace(gistURL[i:])
			if j := strings.IndexByte(gistURL, '\n'); j >= 0 {
				gistURL = gistURL[:j]
			}
		}
		if gistURL == "" {
			writeErr(w, http.StatusBadGateway, "gh did not return a gist URL")
			return
		}
		id := gistURL[strings.LastIndex(gistURL, "/")+1:]
		writeJSON(w, http.StatusOK, map[string]any{
			"gist":   gistURL,
			"viewer": "https://pi.dev/session/" + id,
		})
	}
}

func shareHTML(path string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>PiCode session</title>`)
	b.WriteString(`<style>body{font:14px/1.5 system-ui,sans-serif;max-width:720px;margin:32px auto;padding:0 16px;color:#111}`)
	b.WriteString(`.u{background:#eef;padding:10px 12px;border-radius:8px;margin:12px 0}`)
	b.WriteString(`.a{background:#f4f4f5;padding:10px 12px;border-radius:8px;margin:12px 0;white-space:pre-wrap}`)
	b.WriteString(`h1{font-size:18px}</style></head><body>`)
	b.WriteString("<h1>PiCode session</h1>")
	evs, err := session.Transcript(path)
	if err != nil {
		b.WriteString("<p>Could not read transcript.</p></body></html>")
		return b.String()
	}
	n := 0
	for _, e := range evs {
		if e.Kind == "compaction" {
			n++
			fmt.Fprintf(&b, `<div class="a"><strong>· Session compacted ·</strong>%s%s</div>`, "\n", html.EscapeString(e.Text))
			continue
		}
		if e.Kind != "user" && e.Kind != "assistant" {
			continue
		}
		n++
		cls := "a"
		if e.Kind == "user" {
			cls = "u"
		}
		fmt.Fprintf(&b, `<div class="%s">%s</div>`, cls, html.EscapeString(e.Text))
	}
	if n == 0 {
		b.WriteString("<p>Empty session.</p>")
	}
	fmt.Fprintf(&b, "<p>%s</p></body></html>", html.EscapeString(time.Now().UTC().Format(time.RFC3339)))
	return b.String()
}
