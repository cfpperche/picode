package rpc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplyNeedleNormalizesAndTrims(t *testing.T) {
	long := "Human reply to your question \"title\": " +
		"palavra   \n com   acentos e   texto suficientemente longo para estourar o limite de sessenta e quatro runes"
	got := replyNeedle(long)
	r := []rune(got)
	if len(r) > 64 {
		t.Fatalf("needle too long: %d runes", len(r))
	}
	if strings.Join(strings.Fields(got), " ") != got {
		t.Fatalf("needle not whitespace-normalized: %q", got)
	}
	if !strings.HasSuffix(got, "quatro runes") {
		t.Fatalf("needle lost the reply tail: %q", got)
	}
	if replyNeedle("  ") != "" {
		t.Fatalf("blank payload should normalize to empty")
	}
}

func TestDirHasUserText(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A real-shaped pi session line: JSON with escaped quotes inside the text.
	write("s1.jsonl", "{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":[{\"type\":\"text\",\"text\":\"Human reply to your question \\\"[Teste burst]\\\": reply resolvido por mim :D\"}]}}\n")
	write("s2.jsonl", "{\"type\":\"message\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"Working in the terminal — output lands there, not here.\"}]}}\n")

	if !dirHasUserText(dir, replyNeedle("Human reply to your question \"[Teste burst]\": reply resolvido por mim :D")) {
		t.Fatal("needle present in a user message: not found")
	}
	// An assistant line containing the same words is not a user reply.
	if dirHasUserText(dir, replyNeedle("Working in the terminal — output lands there, not here.")) {
		t.Fatal("assistant line matched as user reply")
	}
	if dirHasUserText(dir, replyNeedle("resposta que nunca foi escrita em arquivo nenhum")) {
		t.Fatal("absent needle matched")
	}
	// Empty directory: nothing matches.
	empty := t.TempDir()
	if dirHasUserText(empty, "qualquer coisa") {
		t.Fatal("empty dir matched")
	}
}
