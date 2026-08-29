package install

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// ReleaseRepo is the GitHub repo `picode update` checks.
const ReleaseRepo = "cfpperche/picode"

// HTTPClient is the GitHub client. Tests replace it.
var HTTPClient = &http.Client{Timeout: 15 * time.Second}

// Release is a GitHub release we might install.
type Release struct {
	Tag      string
	URL      string
	Asset    string
	AssetURL string
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func stripV(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

// Newer reports whether latest is a greater semver than current (major.minor.patch).
func Newer(current, latest string) bool {
	c := parseVer(stripV(current))
	l := parseVer(stripV(latest))
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseVer(s string) [3]int {
	var out [3]int
	parts := strings.Split(s, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		n := 0
		for _, r := range parts[i] {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}

func assetName() string {
	return fmt.Sprintf("picode-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// APIRoot is the GitHub API prefix. Tests replace it.
var APIRoot = "https://api.github.com"

// LatestRelease fetches GitHub's latest release.
func LatestRelease() (Release, error) {
	url := APIRoot + "/repos/" + ReleaseRepo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "picode")
	res, err := HTTPClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusNotFound {
		return Release{}, fmt.Errorf("no published release")
	}
	if res.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("github: %s", res.Status)
	}
	var g ghRelease
	if err := json.Unmarshal(body, &g); err != nil {
		return Release{}, err
	}
	want := assetName()
	rel := Release{Tag: stripV(g.TagName), URL: g.HTMLURL}
	for _, a := range g.Assets {
		if a.Name == want || a.Name == want+".tar.gz" {
			rel.Asset = a.Name
			rel.AssetURL = a.BrowserDownloadURL
			break
		}
	}
	return rel, nil
}

// Download writes url to dest (0755).
func Download(url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "picode")
	res, err := HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %s", res.Status)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, io.LimitReader(res.Body, 200<<20))
	return err
}
