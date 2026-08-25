package llama

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func picodeLlamaDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return "llama"
	}
	return filepath.Join(home, ".picode", "llama")
}

func bundledBinary() string {
	dir := picodeLlamaDir()
	for _, n := range []string{"llama-server", "llama-server.exe"} {
		p := filepath.Join(dir, n)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// PickAsset chooses a portable CPU/OS build from llama.cpp release names.
func PickAsset(names []string, goos, goarch string) string {
	var needle string
	switch goos + "/" + goarch {
	case "linux/amd64":
		needle = "bin-ubuntu-x64.tar.gz"
	case "linux/arm64":
		needle = "bin-ubuntu-arm64.tar.gz"
	case "darwin/arm64":
		needle = "bin-macos-arm64.tar.gz"
	case "darwin/amd64":
		needle = "bin-macos-x64.tar.gz"
	case "windows/amd64":
		needle = "bin-win-cpu-x64.zip"
	case "windows/arm64":
		needle = "bin-win-cpu-arm64.zip"
	default:
		return ""
	}
	for _, n := range names {
		low := strings.ToLower(n)
		if strings.Contains(low, needle) && !strings.Contains(low, "vulkan") && !strings.Contains(low, "cuda") {
			return n
		}
	}
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), needle) {
			return n
		}
	}
	return ""
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease() (ghRelease, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/ggml-org/llama.cpp/releases?per_page=20", nil)
	if err != nil {
		return ghRelease{}, err
	}
	req.Header.Set("User-Agent", "picode")
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return ghRelease{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return ghRelease{}, fmt.Errorf("GitHub HTTP %d", res.StatusCode)
	}
	var list []ghRelease
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		return ghRelease{}, err
	}
	for _, r := range list {
		if strings.HasPrefix(r.TagName, "b") && len(r.Assets) > 0 {
			return r, nil
		}
	}
	return ghRelease{}, fmt.Errorf("no llama.cpp build found")
}

// InstallBinary downloads a matching llama.cpp release into ~/.picode/llama/.
func InstallBinary() (string, error) {
	rel, err := latestRelease()
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(rel.Assets))
	byName := map[string]string{}
	for _, a := range rel.Assets {
		names = append(names, a.Name)
		byName[a.Name] = a.URL
	}
	pick := PickAsset(names, runtime.GOOS, runtime.GOARCH)
	if pick == "" {
		return "", fmt.Errorf("no llama.cpp build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	dlURL := byName[pick]
	tmp, err := os.CreateTemp("", "picode-llama-*"+filepath.Ext(pick))
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := downloadFile(dlURL, tmp); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	dest := picodeLlamaDir()
	if err := os.MkdirAll(dest, 0o700); err != nil {
		return "", err
	}
	if err := extractArchive(tmpName, dest); err != nil {
		return "", err
	}
	bin, err := promoteServer(dest)
	if err != nil {
		return "", err
	}
	return bin, nil
}

func downloadFile(rawURL string, dst *os.File) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "picode")
	res, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("download HTTP %d", res.StatusCode)
	}
	_, err = io.Copy(dst, res.Body)
	return err
}

func extractArchive(path, dest string) error {
	low := strings.ToLower(path)
	switch {
	case strings.HasSuffix(low, ".zip"):
		return unzip(path, dest)
	case strings.HasSuffix(low, ".tar.gz") || strings.HasSuffix(low, ".tgz"):
		return untarGz(path, dest)
	default:
		return fmt.Errorf("unknown archive")
	}
}

func unzip(path, dest string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if err := writeZipFile(dest, f); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(dest string, f *zip.File) error {
	name := filepath.Clean(f.Name)
	if strings.HasPrefix(name, "..") {
		return nil
	}
	out := filepath.Join(dest, name)
	if !strings.HasPrefix(out, dest) {
		return nil
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(out, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	w, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rc)
	_ = w.Close()
	return err
}

func untarGz(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if strings.HasPrefix(name, "..") {
			continue
		}
		out := filepath.Join(dest, name)
		if !strings.HasPrefix(out, dest) {
			continue
		}
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		w, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, tr)
		_ = w.Close()
		if err != nil {
			return err
		}
	}
}

func promoteServer(dest string) (string, error) {
	var found string
	_ = filepath.Walk(dest, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		n := strings.ToLower(info.Name())
		if n == "llama-server" || n == "llama-server.exe" {
			found = p
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("archive had no llama-server")
	}
	target := filepath.Join(dest, filepath.Base(found))
	if found != target {
		_ = os.Remove(target)
		if err := os.Link(found, target); err != nil {
			b, rerr := os.ReadFile(found)
			if rerr != nil {
				return "", rerr
			}
			if err := os.WriteFile(target, b, 0o755); err != nil {
				return "", err
			}
		}
		_ = os.Chmod(target, 0o755)
		found = target
	}
	return found, nil
}
