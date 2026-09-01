package store

import (
	"reflect"
	"testing"
)

func TestCLIFlagsForSpawn(t *testing.T) {
	fresh := Agent{}
	if got, want := fresh.CLIFlagsForSpawn("new-id"), []string{"--session-id", "new-id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fresh start = %v, want %v", got, want)
	}
	if got := fresh.CLIFlagsForSpawn(""); len(got) != 0 {
		t.Fatalf("empty sessionID should be a no-op, got %v", got)
	}

	path := "/some/path.jsonl"
	resuming := Agent{SessionPath: &path}
	got := resuming.CLIFlagsForSpawn("should-be-ignored")
	if want := resuming.CLIFlags(); !reflect.DeepEqual(got, want) {
		t.Fatalf("resuming agent = %v, want CLIFlags() = %v", got, want)
	}
	for _, a := range got {
		if a == "--session-id" {
			t.Fatalf("resuming agent must not also get --session-id: %v", got)
		}
	}
}
