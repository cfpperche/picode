package pipkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const npmRegistry = "https://registry.npmjs.org"

var registryHTTP = &http.Client{Timeout: 8 * time.Second}

// Available is one installed package that is behind.
type Available struct {
	Source  string `json:"source"`
	Scope   string `json:"scope"`
	Current string `json:"current,omitempty"`
	Latest  string `json:"latest,omitempty"`
}

// UpdateReport is GET /api/packages/updates.
type UpdateReport struct {
	Updates []Available `json:"updates"`
}

// CheckUpdates compares installed npm packages to the registry.
// Path, git, agent, and pinned npm:@ver sources are skipped. A registry
// miss skips that row; the rest still return.
func CheckUpdates(ctx context.Context, userDir, projectDir string) (UpdateReport, error) {
	return checkUpdates(ctx, registryHTTP, npmRegistry, userDir, projectDir)
}

func checkUpdates(ctx context.Context, client *http.Client, base, userDir, projectDir string) (UpdateReport, error) {
	out := UpdateReport{Updates: []Available{}}
	listed, err := List(userDir, projectDir)
	if err != nil {
		return out, err
	}
	for _, p := range listed.Packages {
		if ctx.Err() != nil {
			break
		}
		if p.Kind != "npm" || NpmPinned(p.Source) || p.InstalledPath == "" {
			continue
		}
		name := NpmName(p.Source)
		if name == "" {
			continue
		}
		current := readPkgVersion(p.InstalledPath)
		if current == "" {
			continue
		}
		latest, err := npmLatest(ctx, client, base, name)
		if err != nil || latest == "" {
			continue
		}
		if Newer(latest, current) {
			out.Updates = append(out.Updates, Available{
				Source:  p.Source,
				Scope:   p.Scope,
				Current: current,
				Latest:  latest,
			})
		}
	}
	return out, nil
}

// UpdateArgs is `pi update --extension <source> --no-approve`.
func UpdateArgs(source string) []string {
	return []string{"update", "--extension", source, "--no-approve"}
}

// Update runs pi update for one source. Project scope uses opts.Cwd.
func Update(ctx context.Context, piCmd, source string, opts MutateOpts) error {
	if err := ValidSource(source); err != nil {
		return err
	}
	if opts.Local && strings.TrimSpace(opts.Cwd) == "" {
		return fmt.Errorf("project scope needs a workspace")
	}
	if piCmd == "" {
		piCmd = "pi"
	}
	cmd := exec.CommandContext(ctx, piCmd, UpdateArgs(source)...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", firstLine(msg))
	}
	return nil
}

// NpmName is the npm package name in a source (no npm: prefix, no @version).
func NpmName(source string) string {
	if KindOf(source) != "npm" {
		return ""
	}
	s := strings.TrimSpace(source)
	s = strings.TrimPrefix(s, "npm:")
	if strings.HasPrefix(s, "@") {
		if i := strings.LastIndex(s, "@"); i > 0 {
			s = s[:i]
		}
		return s
	}
	if i := strings.Index(s, "@"); i > 0 {
		s = s[:i]
	}
	return s
}

// NpmPinned is true when the source pins an exact version (npm:pkg@1.2.3).
func NpmPinned(source string) bool {
	if KindOf(source) != "npm" {
		return false
	}
	s := strings.TrimSpace(source)
	s = strings.TrimPrefix(s, "npm:")
	var ver string
	if strings.HasPrefix(s, "@") {
		if i := strings.LastIndex(s, "@"); i > 0 {
			ver = s[i+1:]
		}
	} else if i := strings.Index(s, "@"); i > 0 {
		ver = s[i+1:]
	}
	if ver == "" || strings.ContainsAny(ver, "^~*<>=") {
		return false
	}
	_, ok := parseSemver(ver)
	return ok
}

// Newer is true when latest is a higher major.minor.patch than current.
func Newer(latest, current string) bool {
	a, ok1 := parseSemver(latest)
	b, ok2 := parseSemver(current)
	if !ok1 || !ok2 {
		return false
	}
	if a[0] != b[0] {
		return a[0] > b[0]
	}
	if a[1] != b[1] {
		return a[1] > b[1]
	}
	return a[2] > b[2]
}

func parseSemver(s string) ([3]int, bool) {
	var zero [3]int
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return zero, false
	}
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return zero, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return zero, false
		}
		out[i] = n
	}
	return out, true
}

func readPkgVersion(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ""
	}
	var raw struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(b, &raw) != nil {
		return ""
	}
	return strings.TrimSpace(raw.Version)
}

func npmLatest(ctx context.Context, client *http.Client, base, name string) (string, error) {
	if client == nil {
		client = registryHTTP
	}
	if base == "" {
		base = npmRegistry
	}
	u := strings.TrimRight(base, "/") + "/" + url.PathEscape(name) + "/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "picode (https://github.com/cfpperche/picode)")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<18))
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry %s", res.Status)
	}
	var raw struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", err
	}
	return strings.TrimSpace(raw.Version), nil
}
