package llama

import "testing"

func TestQuantPat(t *testing.T) {
	m := quantPat.FindStringSubmatch("Llama-3.2-1B-Instruct-Q4_K_M")
	if len(m) < 2 || m[1] != "Q4_K_M" {
		t.Fatalf("%v", m)
	}
}
