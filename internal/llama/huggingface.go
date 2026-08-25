package llama

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// HFHit is one search row (no token).
type HFHit struct {
	ID        string `json:"id"`
	Downloads int    `json:"downloads"`
}

// HFQuant is one GGUF quantization.
type HFQuant struct {
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
}

// HFDetails is repo metadata for download.
type HFDetails struct {
	ID            string    `json:"id"`
	Gated         string    `json:"gated,omitempty"` // auto | manual | empty
	Quantizations []HFQuant `json:"quantizations"`
}

var (
	quantPat = regexp.MustCompile(`(?i)(?:^|[-_.])((?:UD-)?(?:IQ\d(?:_[A-Z0-9]+)+|Q\d(?:_[A-Z0-9]+)+|BF16|F16|F32|MXFP\d(?:_[A-Z0-9]+)*))$`)
	shardPat = regexp.MustCompile(`-\d{5}-of-\d{5}$`)
)

func hfToken() string {
	if t := strings.TrimSpace(os.Getenv("HF_TOKEN")); t != "" {
		return t
	}
	home, _ := os.UserHomeDir()
	paths := []string{}
	if p := os.Getenv("HF_TOKEN_PATH"); p != "" {
		paths = append(paths, p)
	}
	if p := os.Getenv("HF_HOME"); p != "" {
		paths = append(paths, filepath.Join(p, "token"))
	}
	if home != "" {
		paths = append(paths, filepath.Join(home, ".cache", "huggingface", "token"))
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err == nil {
			if t := strings.TrimSpace(string(b)); t != "" {
				return t
			}
		}
	}
	return ""
}

func hfGet(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, "https://huggingface.co"+path, nil)
	if err != nil {
		return nil, err
	}
	if t := hfToken(); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	cli := &http.Client{Timeout: 15 * time.Second}
	res, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == 429 {
		return nil, fmt.Errorf("Hugging Face rate limit")
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("Hugging Face HTTP %d", res.StatusCode)
	}
	return raw, nil
}

// HFSearch finds GGUF repos. Never returns the token.
func HFSearch(q string) ([]HFHit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []HFHit{}, nil
	}
	v := url.Values{}
	v.Set("search", q)
	v.Set("filter", "gguf")
	v.Set("sort", "downloads")
	v.Set("direction", "-1")
	v.Set("limit", "20")
	raw, err := hfGet("/api/models?" + v.Encode())
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID        string `json:"id"`
		Downloads int    `json:"downloads"`
	}
	if json.Unmarshal(raw, &rows) != nil {
		return nil, fmt.Errorf("invalid Hugging Face search")
	}
	out := make([]HFHit, 0, len(rows))
	for _, r := range rows {
		if r.ID != "" {
			out = append(out, HFHit{ID: r.ID, Downloads: r.Downloads})
		}
	}
	return out, nil
}

// HFInfo loads quantizations and gated flag.
func HFInfo(id string) (HFDetails, error) {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "..") {
		return HFDetails{}, fmt.Errorf("model id required")
	}
	parts := strings.Split(id, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	raw, err := hfGet("/api/models/" + strings.Join(parts, "/") + "?blobs=true")
	if err != nil {
		return HFDetails{}, err
	}
	var payload struct {
		ID       string `json:"id"`
		Gated    any    `json:"gated"`
		Siblings []struct {
			RFilename string `json:"rfilename"`
			Size      int64  `json:"size"`
		} `json:"siblings"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return HFDetails{}, fmt.Errorf("invalid Hugging Face details")
	}
	out := HFDetails{ID: payload.ID}
	if out.ID == "" {
		out.ID = id
	}
	switch g := payload.Gated.(type) {
	case string:
		if g == "auto" || g == "manual" {
			out.Gated = g
		}
	case bool:
		if g {
			out.Gated = "auto"
		}
	}
	sizes := map[string]int64{}
	okSize := map[string]bool{}
	for _, s := range payload.Siblings {
		fn := s.RFilename
		if i := strings.LastIndex(fn, "/"); i >= 0 {
			fn = fn[i+1:]
		}
		low := strings.ToLower(fn)
		if !strings.HasSuffix(low, ".gguf") || strings.HasPrefix(low, "mmproj") {
			continue
		}
		stem := strings.TrimSuffix(fn, ".gguf")
		stem = strings.TrimSuffix(stem, ".GGUF")
		stem = shardPat.ReplaceAllString(stem, "")
		m := quantPat.FindStringSubmatch(stem)
		if len(m) < 2 {
			continue
		}
		q := strings.ToUpper(m[1])
		if s.Size > 0 {
			sizes[q] += s.Size
			if _, seen := okSize[q]; !seen {
				okSize[q] = true
			}
		} else {
			okSize[q] = false
		}
	}
	for name, sz := range sizes {
		item := HFQuant{Name: name}
		if okSize[name] {
			item.Size = sz
		}
		out.Quantizations = append(out.Quantizations, item)
	}
	sort.Slice(out.Quantizations, func(i, j int) bool {
		if out.Quantizations[i].Name == "Q4_K_M" {
			return true
		}
		if out.Quantizations[j].Name == "Q4_K_M" {
			return false
		}
		if out.Quantizations[i].Size != out.Quantizations[j].Size {
			return out.Quantizations[i].Size > 0 && (out.Quantizations[j].Size == 0 || out.Quantizations[i].Size < out.Quantizations[j].Size)
		}
		return out.Quantizations[i].Name < out.Quantizations[j].Name
	})
	return out, nil
}
