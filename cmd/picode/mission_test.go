package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcptool"
)

func TestMissionCLIHelpAndIdentity(t *testing.T) {
	var out, errs bytes.Buffer
	if code := missionMain(t.Context(), []string{"--help"}, &mcptool.Caller{}, &out, &errs); code != 0 || !strings.Contains(out.String(), "--request-id") {
		t.Fatal(code, out.String())
	}
	out.Reset()
	if code := missionMain(t.Context(), []string{"accept", "--id", "m"}, &mcptool.Caller{}, &out, &errs); code != 2 {
		t.Fatal("agent CLI exposed owner action", code)
	}
	if code := missionMain(t.Context(), []string{"show", "--id", "m"}, &mcptool.Caller{}, &out, &errs); code != 1 {
		t.Fatal("unattributed CLI read mission", code)
	}
}
