package auth

import (
	"errors"
	"os"
	"testing"
)

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func errorsIs(err, target error) bool { return errors.Is(err, target) }
