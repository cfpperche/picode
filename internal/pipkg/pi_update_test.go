package pipkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPiUpdateCheck(t *testing.T) {
	orig := piRegistryBase
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/"+PiPackage+"/latest") {
			_, _ = w.Write([]byte(`{"version":"9.9.9"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()
	piRegistryBase = ts.URL
	t.Cleanup(func() { piRegistryBase = orig })
	ResetPiUpdateCache()

	// Outdated: latest 9.9.9 beats 1.0.0.
	info := PiUpdateCheck(context.Background(), "1.0.0")
	if !info.Outdated || info.Latest != "9.9.9" || info.Current != "1.0.0" {
		t.Fatalf("info = %+v", info)
	}

	// Fresh: same version means no alert.
	if info := PiUpdateCheck(context.Background(), "9.9.9"); info.Outdated {
		t.Fatalf("9.9.9 reported outdated: %+v", info)
	}

	// Empty current is a no-op.
	if info := PiUpdateCheck(context.Background(), ""); info.Outdated || info.Latest != "" {
		t.Fatalf("empty current = %+v", info)
	}
}

func TestPiUpdateCheckCache(t *testing.T) {
	var mu sync.Mutex
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		_, _ = w.Write([]byte(`{"version":"2.0.0"}`))
	}))
	defer ts.Close()
	orig := piRegistryBase
	piRegistryBase = ts.URL
	t.Cleanup(func() { piRegistryBase = orig })
	ResetPiUpdateCache()

	for i := 0; i < 3; i++ {
		if info := PiUpdateCheck(context.Background(), "1.0.0"); !info.Outdated {
			t.Fatalf("iter %d = %+v", i, info)
		}
	}
	mu.Lock()
	if hits != 1 {
		t.Fatalf("registry hits = %d, want 1 (cache)", hits)
	}
	mu.Unlock()

	// Reset drops the cache.
	ResetPiUpdateCache()
	PiUpdateCheck(context.Background(), "1.0.0")
	mu.Lock()
	if hits != 2 {
		t.Fatalf("registry hits after reset = %d, want 2", hits)
	}
	mu.Unlock()
}

func TestPiUpdateCheckFallsBackToCache(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"3.0.0"}`))
	}))
	orig := piRegistryBase
	piRegistryBase = ts.URL
	ResetPiUpdateCache()
	PiUpdateCheck(context.Background(), "1.0.0")
	ts.Close() // registry gone

	if info := PiUpdateCheck(context.Background(), "1.0.0"); !info.Outdated || info.Latest != "3.0.0" {
		t.Fatalf("hiccup lost the cached answer: %+v", info)
	}
	piRegistryBase = orig
	ResetPiUpdateCache()

	// A different current with no cache and no registry stays calm.
	ts404 := httptest.NewServer(http.HandlerFunc(http.NotFound))
	defer ts404.Close()
	piRegistryBase = ts404.URL
	t.Cleanup(func() { piRegistryBase = orig })
	if info := PiUpdateCheck(context.Background(), "5.0.0"); info.Outdated || info.Latest != "" {
		t.Fatalf("no-registry = %+v", info)
	}
	_ = time.Second
}
