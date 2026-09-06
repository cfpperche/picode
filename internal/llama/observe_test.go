package llama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCapabilityEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, role, build, content string
		events, cancel             bool
	}{
		{"verified", "router", "b10809-test", "text/event-stream", true, true},
		{"unverified build", "router", "b10810-test", "text/event-stream", true, false},
		{"not SSE", "router", "b10809-test", "text/html", false, false},
		{"single model", "model", "b10809-test", "text/event-stream", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Error("capability probe mutated the server")
				}
				if r.Header.Get("Authorization") != "Bearer secret" {
					t.Error("missing authentication")
				}
				if r.URL.Path == "/props" {
					json.NewEncoder(w).Encode(map[string]string{"role": tc.role, "build_info": tc.build})
					return
				}
				w.Header().Set("Content-Type", tc.content)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer upstream.Close()
			c, _ := New(upstream.URL, "secret")
			got := c.Capabilities(context.Background())
			if got.Events != tc.events || got.CancelDownload != tc.cancel {
				t.Fatal(got)
			}
		})
	}
}

func TestProgressRedactsURLsAndPreservesUnknownTotal(t *testing.T) {
	raw := json.RawMessage(`{"https://user:secret@host/a.gguf?token=secret":{"done":101,"total":100},"https://host/b.gguf#secret":{"done":4,"total":0}}`)
	got := parseProgress(raw)
	if len(got) != 2 {
		t.Fatal(got)
	}
	for _, p := range got {
		switch p.File {
		case "a.gguf":
			if p.Done != 100 {
				t.Fatal(p)
			}
		case "b.gguf":
			if p.Total != 0 || p.Done != 4 {
				t.Fatal(p)
			}
		default:
			t.Fatal(p)
		}
	}
	if got := parseProgress(json.RawMessage(`{"stages":["text_model"],"value":0.5}`)); len(got) != 0 {
		t.Fatal(got)
	}
	if _, err := NormalizeURL("http://user:secret@host"); err == nil {
		t.Fatal("URL credentials accepted")
	}
}

func TestEventsAndContextCancellation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models/sse" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"event\":\"model_status\"}\n\n")
			return
		}
		<-r.Context().Done()
	}))
	defer upstream.Close()
	c, _ := New(upstream.URL, "")
	calls := 0
	if err := c.Observe(context.Background(), func(ModelEvent) { calls++ }); err != nil || calls != 1 {
		t.Fatal(calls, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if _, err := c.ListContext(ctx); err == nil {
		t.Fatal("ignored canceled context")
	}
}

func TestProgressEventsSupportWrappedAndDirectPayloads(t *testing.T) {
	for _, data := range []string{`{"https://host/file.gguf?token=secret":{"done":12,"total":50}}`, `{"progress":{"https://host/file.gguf?token=secret":{"done":12,"total":50}}}`} {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"model\":\"m\",\"event\":\"download_progress\",\"data\":%s}\n\n", data)
		}))
		c, _ := New(upstream.URL, "")
		var got ModelEvent
		err := c.Observe(context.Background(), func(e ModelEvent) { got = e })
		upstream.Close()
		if err != nil || len(got.Progress) != 1 || got.Progress[0].File != "file.gguf" || got.Progress[0].Done != 12 || got.Progress[0].Total != 50 {
			t.Fatal(got, err)
		}
	}
	if got := parseProgress(json.RawMessage(`{"progress":{"https://host/file.gguf":{"done":12,"total":50}}}`)); len(got) != 0 {
		t.Fatal("wrapper mistaken for file", got)
	}
}
