package provision

import "testing"

// ownerConf is the real /etc/wsl.conf this feature was designed against. It
// already carries systemd=true, so provisioning must not touch it: the
// comment, the key order and the spaced "generateResolvConf = false" all have
// to survive. Rewriting this file is how a provisioner breaks someone's DNS.
const ownerConf = `[boot]
systemd=true

[user]
default=goat

# Optional: remove windows from PATH (autocompletion)
[interop]
appendWindowsPath=false

[network]
generateResolvConf = false
`

func TestEnsureSystemdLeavesASatisfiedFileAlone(t *testing.T) {
	got, changed := EnsureSystemd(ownerConf)
	if changed {
		t.Error("changed = true, want false: systemd=true is already set")
	}
	if got != ownerConf {
		t.Errorf("file was rewritten\n--- got ---\n%s\n--- want ---\n%s", got, ownerConf)
	}
}

func TestEnsureKey(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		section     string
		key         string
		value       string
		want        string
		wantChanged bool
	}{
		{
			name:    "key missing lands after the section's last pair",
			in:      "[boot]\nprotectBinfmt=true\n\n[user]\ndefault=goat\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nprotectBinfmt=true\nsystemd=true\n\n[user]\ndefault=goat\n",
			wantChanged: true,
		},
		{
			name:    "empty section takes the key right after the header",
			in:      "[boot]\n\n[user]\ndefault=goat\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nsystemd=true\n\n[user]\ndefault=goat\n",
			wantChanged: true,
		},
		{
			name:    "a comment-only section keeps its comment trailing",
			in:      "[boot]\n# nothing here yet\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nsystemd=true\n# nothing here yet\n",
			wantChanged: true,
		},
		{
			name:    "missing section is appended after one blank line",
			in:      "[user]\ndefault=goat\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[user]\ndefault=goat\n\n[boot]\nsystemd=true\n",
			wantChanged: true,
		},
		{
			name:    "empty file becomes just the section",
			in:      "",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nsystemd=true\n",
			wantChanged: true,
		},
		{
			name:    "wrong value is swapped and the spacing kept",
			in:      "[network]\ngenerateResolvConf = false\n",
			section: "network", key: "generateResolvConf", value: "true",
			want:        "[network]\ngenerateResolvConf = true\n",
			wantChanged: true,
		},
		{
			name:    "section, key and value all match case-insensitively",
			in:      "[Boot]\nSystemd = True\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[Boot]\nSystemd = True\n",
			wantChanged: false,
		},
		{
			name:    "a commented-out key is not a key",
			in:      "[boot]\n#systemd=true\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nsystemd=true\n#systemd=true\n",
			wantChanged: true,
		},
		{
			name:    "the same key in another section is not this key",
			in:      "[other]\nsystemd=true\n\n[boot]\nprotectBinfmt=true\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[other]\nsystemd=true\n\n[boot]\nprotectBinfmt=true\nsystemd=true\n",
			wantChanged: true,
		},
		{
			name:    "CRLF line endings survive",
			in:      "[boot]\r\nprotectBinfmt=true\r\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\r\nprotectBinfmt=true\r\nsystemd=true\r\n",
			wantChanged: true,
		},
		{
			name:    "a file with no trailing newline keeps none",
			in:      "[boot]\nsystemd=false",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nsystemd=true",
			wantChanged: true,
		},
		{
			name:    "trailing blank lines stay at the end",
			in:      "[boot]\nprotectBinfmt=true\n\n",
			section: "boot", key: "systemd", value: "true",
			want:        "[boot]\nprotectBinfmt=true\nsystemd=true\n\n",
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := EnsureKey(tt.in, tt.section, tt.key, tt.value)
			if got != tt.want {
				t.Errorf("content\n--- got ---\n%q\n--- want ---\n%q", got, tt.want)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

// A fix that is not idempotent is a fix that corrupts the file on the second
// run, which is exactly what a converging installer does every logon.
func TestEnsureKeyIsIdempotent(t *testing.T) {
	for _, in := range []string{
		"",
		"[user]\ndefault=goat\n",
		"[boot]\nsystemd=false\n",
		"[boot]\n# nothing here yet\n",
		"[boot]\r\nprotectBinfmt=true\r\n",
		ownerConf,
	} {
		once, _ := EnsureSystemd(in)
		twice, changed := EnsureSystemd(once)
		if changed {
			t.Errorf("second pass reported a change for %q", in)
		}
		if twice != once {
			t.Errorf("second pass rewrote the file\n--- once ---\n%q\n--- twice ---\n%q", once, twice)
		}
	}
}
