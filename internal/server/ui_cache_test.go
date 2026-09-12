package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUICachePolicy(t *testing.T) {
	for _, tc := range []struct {
		path      string
		immutable bool
	}{
		{"/", false},
		{"/?mobile=1", false},
		{"/desktop/", false},
		{"/browser/", false},
		{"/mobile/", false},
		{"/mobile/index.html", false},
		{"/sw.js", false},
		{"/manifest.json", false},
		{"/api/agents", false},
		{"/assets/launcher-abcd.js", true},
		{"/browser/assets/index-abcd.js", true},
		{"/desktop/assets/index-abcd.js", true},
		{"/mobile/assets/index-abcd.css", true},
		{"/mobile/assets-old/file.js", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			cacheControl(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			want := "no-cache"
			if tc.immutable {
				want = "public, max-age=31536000, immutable"
			}
			if got := rec.Header().Get("Cache-Control"); got != want {
				t.Fatalf("Cache-Control = %q, want %q", got, want)
			}
		})
	}
}
