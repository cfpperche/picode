package llama

import "testing"

func TestNormalizeURL(t *testing.T) {
	got, err := NormalizeURL("http://127.0.0.1:8080/v1/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:8080" {
		t.Fatalf("%s", got)
	}
	if _, err := NormalizeURL("file:///etc/passwd"); err == nil {
		t.Fatal("expected error")
	}
	got, err = NormalizeURL("")
	if err != nil || got != DefaultURL {
		t.Fatalf("%s %v", got, err)
	}
}
