package communication

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalTrustDecisionTable(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	data := t.TempDir()
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(filepath.Join(data, "cert.pem"), cert, 0600); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(data, "existing-ca.pem")
	if err := os.WriteFile(extra, cert, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, url string
		extras    []string
		ok, empty bool
	}{
		{"HTTP", "http://localhost:1234", nil, true, true},
		{"self signed localhost IP", server.URL, nil, true, false},
		{"preserve existing CA", server.URL, []string{extra}, true, false},
		{"missing existing CA", server.URL, []string{filepath.Join(data, "missing")}, false, false},
		{"hostname mismatch", "https://localhost:1234", nil, false, false},
		{"remote refused", "https://example.com", nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LocalTrust(data, tc.url, tc.extras)
			if (err == nil) != tc.ok {
				t.Fatalf("trust: %v", err)
			}
			if tc.ok && (got == "") != tc.empty {
				t.Fatal("unexpected trust material")
			}
			if len(tc.extras) > 0 && tc.ok && strings.Count(got, "BEGIN CERTIFICATE") != 2 {
				t.Fatal("lost existing trust")
			}
		})
	}
}
