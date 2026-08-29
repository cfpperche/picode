package main

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

// The tray icon is committed as bytes, so a truncated or mangled file would
// only show up as a blank notification area on someone's machine.
func TestTrayIconIsValidICO(t *testing.T) {
	if len(trayIcon) < 6 {
		t.Fatalf("icon is %d bytes — not an ICO", len(trayIcon))
	}
	if reserved := binary.LittleEndian.Uint16(trayIcon[0:]); reserved != 0 {
		t.Errorf("reserved = %d, want 0", reserved)
	}
	if kind := binary.LittleEndian.Uint16(trayIcon[2:]); kind != 1 {
		t.Errorf("type = %d, want 1 (icon)", kind)
	}
	count := int(binary.LittleEndian.Uint16(trayIcon[4:]))
	if count == 0 {
		t.Fatal("the icon holds no images")
	}

	var sawSmall bool
	for i := 0; i < count; i++ {
		e := 6 + 16*i
		if e+16 > len(trayIcon) {
			t.Fatalf("directory entry %d runs past the end of the file", i)
		}
		width := trayIcon[e]
		size := binary.LittleEndian.Uint32(trayIcon[e+8:])
		offset := binary.LittleEndian.Uint32(trayIcon[e+12:])
		if int(offset)+int(size) > len(trayIcon) {
			t.Errorf("image %d (%d bytes at %d) runs past the end of the file", i, size, offset)
		}
		if width > 0 && width <= 16 {
			sawSmall = true
		}
	}
	// Windows asks for 16px in the notification area; without one it scales a
	// larger image and the glyph turns to mush.
	if !sawSmall {
		t.Error("no 16px image — the tray would have to scale a bigger one")
	}
}

// The asset name is a contract between this program and the release workflow.
// Nothing fails loudly if they drift: `update` would simply never find a
// download, on every machine, forever.
func TestReleaseWorkflowPublishesTheAssetUpdateLooksFor(t *testing.T) {
	b, err := os.ReadFile("../../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(b)

	if !strings.Contains(workflow, DesktopAsset) {
		t.Errorf("the release workflow does not build %q", DesktopAsset)
	}
	// internal/install.assetName() builds this shape for picode itself.
	if !strings.Contains(workflow, "picode-${goos}-${goarch}") {
		t.Error("the release workflow does not build picode-<goos>-<goarch>, which `picode update` looks for")
	}
	// A release that does not stamp the version ships a binary claiming to be
	// the source default, so `update` would offer the same upgrade forever.
	if !strings.Contains(workflow, "internal/version.Version=") {
		t.Error("the release workflow does not stamp the version")
	}
}
