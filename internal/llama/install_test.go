package llama

import "testing"

func TestPickAsset(t *testing.T) {
	names := []string{
		"llama-b1-bin-ubuntu-vulkan-x64.tar.gz",
		"llama-b1-bin-ubuntu-x64.tar.gz",
		"llama-b1-bin-win-cpu-x64.zip",
		"llama-b1-bin-macos-arm64.tar.gz",
	}
	if g := PickAsset(names, "linux", "amd64"); g != "llama-b1-bin-ubuntu-x64.tar.gz" {
		t.Fatalf("%s", g)
	}
	if g := PickAsset(names, "windows", "amd64"); g != "llama-b1-bin-win-cpu-x64.zip" {
		t.Fatalf("%s", g)
	}
	if g := PickAsset(names, "darwin", "arm64"); g != "llama-b1-bin-macos-arm64.tar.gz" {
		t.Fatalf("%s", g)
	}
}
