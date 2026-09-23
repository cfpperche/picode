package desktop

import (
	"strings"
	"testing"
)

func TestParseVolumes(t *testing.T) {
	vols, err := parseVolumes(`[{"name":"C:\\","type":3,"format":"NTFS","free":11202166784,"size":510964789248},{"name":"E:\\","type":3,"format":"NTFS","free":269852311552,"size":500088958976},{"name":"F:\\","type":2,"format":"FAT32","free":1,"size":2}]`)
	if err != nil || len(vols) != 3 || vols[1].Letter != "E" || !vols[1].Fixed || vols[2].Fixed {
		t.Fatalf("vols = %+v, err = %v", vols, err)
	}
	one, err := parseVolumes(`{"name":"C:\\","type":3,"format":"NTFS","free":1,"size":2}`)
	if err != nil || len(one) != 1 {
		t.Errorf("a lone object must parse: %+v %v", one, err)
	}
}

// TestCheckPlace is the decision table for a destination:
//
//	drive exists | fixed NTFS | folder name ok | same folder | fits | → ok / reason
//	no           | —          | —              | —           | —    | "there is no drive"
//	yes          | yes        | no ("..")      | —           | —    | folder-name reason
//	yes          | no (USB)   | yes            | —           | —    | "not a fixed local drive"
//	yes          | FAT32      | yes            | —           | —    | "needs NTFS"
//	yes          | yes        | yes            | yes         | —    | "already there"
//	yes          | yes        | yes            | no          | no   | free vs need, in GB
//	same drive   | yes        | yes            | no          | —    | ok: a rename needs no room
//	yes          | yes        | NUL, "...", "x." | —         | —    | folder-name reason
//	yes          | yes        | yes            | no          | yes  | ok, path joined
func TestCheckPlace(t *testing.T) {
	vols := []WinVolume{
		{Letter: "C", Fixed: true, Format: "NTFS", FreeBytes: 11 << 30},
		{Letter: "E", Fixed: true, Format: "NTFS", FreeBytes: 251 << 30},
		{Letter: "F", Fixed: false, Format: "NTFS", FreeBytes: 900 << 30},
		{Letter: "G", Fixed: true, Format: "FAT32", FreeBytes: 900 << 30},
		{Letter: "H", Fixed: true, Format: "NTFS", FreeBytes: 10 << 30},
	}
	facts := DiskFacts{VHDXPath: `C:\Users\cfpp\AppData\Local\wsl\{x}\ext4.vhdx`, VHDXBytes: 222 << 30}
	cases := []struct {
		letter, folder, wantReason, wantPath string
		ok                                   bool
	}{
		{"D", `WSL`, "no drive D:", "", false},
		{"E", `WSL\..\Windows`, "folder name", "", false},
		{"F", `WSL`, "not a fixed local drive", `F:\WSL`, false},
		{"G", `WSL`, "need NTFS", `G:\WSL`, false},
		{"C", `Users\cfpp\AppData\Local\wsl\{x}`, "folder name", "", false},
		{"H", `WSL\Ubuntu`, "H: has 10 GB free; the disk file needs 222 GB, plus 5.0 GB left free", `H:\WSL\Ubuntu`, false},
		{"C", `WSL\Ubuntu`, "", `C:\WSL\Ubuntu`, true}, // same drive: a rename, no room needed
		{"E", `NUL`, "folder name", "", false},
		{"E", `WSL\...`, "folder name", "", false},
		{"E", `WSL\Ubuntu.`, "folder name", "", false},
		{"e", `WSL\Ubuntu`, "", `E:\WSL\Ubuntu`, true},
	}
	for _, c := range cases {
		pc := CheckPlace(vols, facts, c.letter, c.folder)
		if pc.OK != c.ok || (c.wantReason != "" && !strings.Contains(pc.Reason, c.wantReason)) || (c.wantPath != "" && pc.Path != c.wantPath) {
			t.Errorf("%s:%s → %+v", c.letter, c.folder, pc)
		}
	}
	same := CheckPlace([]WinVolume{{Letter: "C", Fixed: true, Format: "NTFS", FreeBytes: 900 << 30}},
		DiskFacts{VHDXPath: `C:\WSL\Ubuntu\ext4.vhdx`, VHDXBytes: 1}, "C", `WSL\Ubuntu`)
	if same.OK || !same.SameDisk {
		t.Errorf("same folder: %+v", same)
	}
}
