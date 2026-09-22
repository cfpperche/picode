package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/desktop"
	"github.com/cfpperche/picode/internal/install"
	"github.com/cfpperche/picode/internal/version"
)

func TestConfirm(t *testing.T) {
	rows := []struct {
		name  string
		in    string
		want  bool
		wantE bool
	}{
		{"yes", "y\n", true, false},
		{"word yes", "YES\n", true, false},
		{"empty means yes", "\n", true, false},
		{"no", "n\n", false, false},
		{"word no", "no\n", false, false},
		{"closed stdin refuses", "", false, true},
	}
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			got, err := confirm(strings.NewReader(tt.in), "Proceed?")
			if got != tt.want || (err != nil) != tt.wantE {
				t.Errorf("confirm = %v, %v", got, err)
			}
		})
	}
}

func TestPicodeVersionOf(t *testing.T) {
	rows := []struct {
		in   string
		want string
	}{
		{"picode 0.3.1\n", "0.3.1"},
		{"picode 0.3.1+a892ede", "0.3.1"},
		{"garbage", ""},
		{"", ""},
	}
	for _, tt := range rows {
		if got := picodeVersionOf([]byte(tt.in)); got != tt.want {
			t.Errorf("picodeVersionOf(%q) = %q", tt.in, got)
		}
	}
}

var testUbuntu = desktop.Distro{Name: "Ubuntu", State: "Stopped", Version: 2, Default: true}

// releaseServer serves the GitHub release API plus asset bytes and a
// matching SHA256SUMS, and records which release URL was asked for.
func releaseServer(t *testing.T, tag, asset string, body []byte) (*httptest.Server, *string) {
	t.Helper()
	var gotPath string
	sum := sha256.Sum256(body)
	mux := http.NewServeMux()
	release := fmt.Sprintf(`{"tag_name":%q,"html_url":"http://example.test/rel","assets":[{"name":%q,"browser_download_url":"%%URL%%/asset"},{"name":"SHA256SUMS","browser_download_url":"%%URL%%/sums"}]}`,
		tag, asset)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest") || strings.Contains(r.URL.Path, "/releases/tags/"):
			gotPath = r.URL.Path
			base := "http://" + r.Host
			fmt.Fprint(w, strings.ReplaceAll(release, "%URL%", base))
		case r.URL.Path == "/asset":
			w.Write(body)
		case r.URL.Path == "/sums":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), asset)
		default:
			http.NotFound(w, r)
		}
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	oldRoot, oldC := install.APIRoot, install.HTTPClient
	install.APIRoot, install.HTTPClient = ts.URL, ts.Client()
	t.Cleanup(func() { install.APIRoot, install.HTTPClient = oldRoot, oldC })
	return ts, &gotPath
}

func stamp(t *testing.T, v, stamped string) {
	t.Helper()
	oldV, oldS := version.Version, version.Stamped
	version.Version, version.Stamped = v, stamped
	t.Cleanup(func() { version.Version, version.Stamped = oldV, oldS })
}

func TestLinuxReleasePinsThisBuild(t *testing.T) {
	_, gotPath := releaseServer(t, "v0.3.1", "picode-linux-amd64", []byte("bin"))
	stamp(t, "0.3.1", "release")
	rel, err := linuxRelease("picode-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*gotPath, "/releases/tags/v0.3.1") {
		t.Errorf("stamped build asked %q", *gotPath)
	}
	if rel.Tag != "0.3.1" || rel.AssetURL == "" || rel.SumsURL == "" {
		t.Errorf("release = %+v", rel)
	}
}

func TestLinuxReleaseTracksLatestUnstamped(t *testing.T) {
	_, gotPath := releaseServer(t, "v0.9.9", "picode-linux-amd64", []byte("bin"))
	stamp(t, "0.3.1", "")
	rel, err := linuxRelease("picode-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(*gotPath, "/releases/latest") {
		t.Errorf("unstamped build asked %q", *gotPath)
	}
	if rel.Tag != "0.9.9" {
		t.Errorf("tag = %q", rel.Tag)
	}
}

func TestFetchLinuxBinaryVerifies(t *testing.T) {
	body := []byte("fake-linux-binary")
	releaseServer(t, "v0.3.1", "picode-linux-amd64", body)
	stamp(t, "0.3.1", "release")
	rel, err := linuxRelease("picode-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	bin, err := fetchLinuxBinary(rel, "picode-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(bin.tmp)
	if !strings.HasSuffix(bin.staged, "picode-linux-amd64") {
		t.Errorf("staged = %q", bin.staged)
	}
}

func TestFetchLinuxBinaryRefusesTamperedSums(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "tampered")
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	rel := install.Release{Tag: "0.3.1", AssetURL: ts.URL + "/a", SumsURL: ts.URL + "/s"}
	if _, err := fetchLinuxBinary(rel, "picode-linux-amd64"); err == nil {
		t.Error("a non-checksum sums file must fail")
	}
}

func TestRunInstallPicodeSkipsMatching(t *testing.T) {
	// runInstallPicode asks for the asset matching the host it runs on
	// (desktop.LinuxAssetName(runtime.GOARCH)), so a fixture that always
	// served picode-linux-amd64 failed on the arm64 macOS runner with
	// "release 0.3.1 has no picode-linux-arm64" — a fixture assumption,
	// not a product fault. The other tests here name the asset themselves
	// and stay arch-independent.
	asset, err := desktop.LinuxAssetName(runtime.GOARCH)
	if err != nil {
		t.Skipf("no Linux picode binary for %s", runtime.GOARCH)
	}
	releaseServer(t, "v0.3.1", asset, []byte("bin"))
	stamp(t, "0.3.1", "release")
	stub := &diskStub{}
	a := &app{runner: stub}
	state := desktop.MachineState{Distros: []desktop.Distro{testUbuntu}, DefaultUser: "goat", TargetUser: "goat", PicodeVersion: "0.3.1"}
	if err := runInstallPicode(a, state, "", ""); err != nil {
		t.Fatal(err)
	}
	if len(stub.calls) != 0 {
		t.Errorf("%d runner calls for a matching binary", len(stub.calls))
	}
}

const probeClean = `@@picode@@
picode 0.3.1
@@tools@@
v22.14.0
@@family@@
ID=ubuntu
@@marker@@
adopted
`

func runtimeState(missing []string, nodeMajor, family string, registered bool) desktop.MachineState {
	return desktop.MachineState{
		Distros: []desktop.Distro{testUbuntu}, DefaultUser: "goat", TargetUser: "goat",
		PicodeVersion: "0.3.1", Missing: missing, NodeMajor: nodeMajor,
		Family: family, RegisteredByDesktop: registered,
	}
}

func TestRunInstallRuntimeNothingMissing(t *testing.T) {
	stub := &diskStub{}
	if err := runInstallRuntime(&app{runner: stub}, runtimeState(nil, "22", "ubuntu", true), "", "", false, nil); err != nil {
		t.Fatal(err)
	}
	if len(stub.calls) != 0 {
		t.Errorf("%d calls for a complete runtime", len(stub.calls))
	}
}

func TestRunInstallRuntimeNonUbuntuStops(t *testing.T) {
	stub := &diskStub{}
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "", "debian", false), "", "", true, nil)
	if err == nil || !strings.Contains(err.Error(), "not Ubuntu") {
		t.Fatalf("err = %v", err)
	}
	if len(stub.calls) != 0 {
		t.Errorf("%d calls before the family gate", len(stub.calls))
	}
}

func TestRunInstallRuntimeRegisteredNeedsNoAsking(t *testing.T) {
	stub := &diskStub{
		replies: [][]byte{nil, nil, nil, []byte(probeClean)},
		errs:    []error{nil, nil, errors.New("mkcert absent"), nil},
	}
	// nil stdin: registered distros must never consult it.
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "22", "ubuntu", true), "", "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(callStrings(stub.calls), "\n")
	for _, want := range []string{"apt-get update", "apt-get install -y tmux"} {
		if !strings.Contains(joined, want) {
			t.Errorf("calls lack %q:\n%s", want, joined)
		}
	}
	for _, never := range []string{"nodesource", "npm install", "pi-coding-agent"} {
		if strings.Contains(joined, never) {
			t.Errorf("node 22 must not reinstall and no CLI is installed here (ADR-0179):\n%s", joined)
		}
	}
}

func TestRunInstallRuntimeAdoptedAsks(t *testing.T) {
	stub := &diskStub{}
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "22", "ubuntu", false), "", "", false, strings.NewReader("n\n"))
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
	if len(stub.calls) != 0 {
		t.Errorf("%d calls after a decline", len(stub.calls))
	}
}

func TestRunInstallRuntimeAdoptedEOFRefuses(t *testing.T) {
	stub := &diskStub{}
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "22", "ubuntu", false), "", "", false, strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInstallRuntimeYesSkipsTheQuestion(t *testing.T) {
	stub := &diskStub{replies: [][]byte{nil, nil, nil, []byte(probeClean)}}
	if err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "22", "ubuntu", false), "", "", true, nil); err != nil {
		t.Fatal(err)
	}
	if len(stub.calls) == 0 {
		t.Error("no calls after --yes")
	}
}

func TestRunInstallRuntimeInstallsNodeAtTheCIMajor(t *testing.T) {
	stub := &diskStub{replies: [][]byte{nil, nil, nil, []byte(probeClean)}}
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"node", "npm"}, "", "ubuntu", true), "", "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(callStrings(stub.calls), "\n"); !strings.Contains(joined, "node_22.x") {
		t.Errorf("no nodejs 22 install:\n%s", joined)
	}
}

// A node already present, whatever its major, is left alone: PiCode does
// not need it, and the CLIs the user installs later say what they need.
func TestRunInstallRuntimeLeavesAnOldNodeAlone(t *testing.T) {
	stub := &diskStub{replies: [][]byte{nil, nil, nil, []byte(probeClean)}}
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "18", "ubuntu", true), "", "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(callStrings(stub.calls), "\n"); strings.Contains(joined, "nodesource") {
		t.Errorf("node 18 was replaced:\n%s", joined)
	}
}

func TestRunInstallRuntimeReportsLeftovers(t *testing.T) {
	stillMissing := "@@tools@@\nmissing:tmux\nv22.1.0\n"
	stub := &diskStub{replies: [][]byte{nil, nil, nil, []byte(stillMissing)}}
	err := runInstallRuntime(&app{runner: stub}, runtimeState([]string{"tmux"}, "22", "ubuntu", true), "", "", false, nil)
	if err == nil || !strings.Contains(err.Error(), "still missing") {
		t.Fatalf("err = %v", err)
	}
}

func callStrings(calls [][]string) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, strings.Join(c, " "))
	}
	return out
}

func TestRunInstallRuntimeUnknownUserStops(t *testing.T) {
	stub := &diskStub{errs: []error{errors.New("exit status 1")}}
	st := runtimeState([]string{"pi"}, "22", "ubuntu", true)
	st.TargetUser = "ghost"
	err := runInstallRuntime(&app{runner: stub}, st, "", "ghost", false, nil)
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("err = %v, want the unknown account named", err)
	}
	if len(stub.calls) != 1 {
		t.Errorf("%d calls, want only the account check", len(stub.calls))
	}
}

func TestRunInstallPicodeUnknownUserStops(t *testing.T) {
	stub := &diskStub{errs: []error{errors.New("exit status 1")}}
	a := &app{runner: stub}
	state := desktop.MachineState{Distros: []desktop.Distro{testUbuntu}, DefaultUser: "goat", TargetUser: "ghost", PicodeVersion: "0.2.0"}
	err := runInstallPicode(a, state, "", "ghost")
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("err = %v, want the unknown account named", err)
	}
	if len(stub.calls) != 1 {
		t.Errorf("%d calls, want only the account check", len(stub.calls))
	}
}

// --user still names the account the probe runs as, but the runtime stage
// installs no CLI for it any more (ADR-0179): every call is root's apt/node
// work or the probe as the user, never an npm install as either.
func TestRunInstallRuntimeUserModeInstallsNoCLI(t *testing.T) {
	stub := &diskStub{replies: [][]byte{[]byte("1000\n"), nil, nil, nil, []byte(probeClean)}}
	st := runtimeState([]string{"tmux"}, "22", "ubuntu", true)
	st.TargetUser = "cfpp"
	if err := runInstallRuntime(&app{runner: stub}, st, "", "cfpp", false, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range callStrings(stub.calls) {
		if strings.Contains(c, "npm install") {
			t.Fatalf("a CLI was installed by the runtime stage: %s", c)
		}
	}
}
