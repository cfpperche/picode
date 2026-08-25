package pipkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const npmSearch = "https://registry.npmjs.org/-/v1/search"

// Hit is one npm package tagged pi-package (gallery discovery).
type Hit struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
	Publisher   string `json:"publisher,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Downloads   int    `json:"downloads,omitempty"`
	Updated     string `json:"updated,omitempty"`
}

// GalleryPage is GET /api/packages/gallery.
type GalleryPage struct {
	Query   string `json:"query"`
	Hits    []Hit  `json:"hits"`
	Source  string `json:"source"`
	Gallery string `json:"gallery"`
}

var galleryHTTP = &http.Client{Timeout: 12 * time.Second}

// SearchGallery queries the public npm registry for keyword pi-package.
func SearchGallery(ctx context.Context, q string) (GalleryPage, error) {
	return searchGallery(ctx, galleryHTTP, npmSearch, q)
}

func searchGallery(ctx context.Context, client *http.Client, endpoint, q string) (GalleryPage, error) {
	page := GalleryPage{
		Query:   strings.TrimSpace(q),
		Hits:    []Hit{},
		Source:  "npm keywords:pi-package",
		Gallery: Gallery,
	}
	text := "keywords:pi-package"
	if page.Query != "" {
		text += " " + page.Query
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return page, err
	}
	qs := u.Query()
	qs.Set("text", text)
	qs.Set("size", "20")
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return page, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "picode (https://github.com/cfpperche/picode)")

	res, err := client.Do(req)
	if err != nil {
		return page, fmt.Errorf("gallery: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return page, err
	}
	if res.StatusCode != http.StatusOK {
		return page, fmt.Errorf("gallery: npm %s", res.Status)
	}
	hits, err := parseNpmSearch(body)
	if err != nil {
		return page, err
	}
	page.Hits = hits
	return page, nil
}

func parseNpmSearch(body []byte) ([]Hit, error) {
	var raw struct {
		Objects []struct {
			Downloads struct {
				Monthly int `json:"monthly"`
			} `json:"downloads"`
			Package struct {
				Name        string   `json:"name"`
				Version     string   `json:"version"`
				Description string   `json:"description"`
				Date        string   `json:"date"`
				Keywords    []string `json:"keywords"`
				Publisher   struct {
					Username string `json:"username"`
				} `json:"publisher"`
			} `json:"package"`
		} `json:"objects"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("gallery: parse: %w", err)
	}
	out := make([]Hit, 0, len(raw.Objects))
	for _, o := range raw.Objects {
		name := strings.TrimSpace(o.Package.Name)
		if name == "" {
			continue
		}
		out = append(out, Hit{
			Name:        name,
			Version:     o.Package.Version,
			Description: strings.TrimSpace(o.Package.Description),
			Source:      "npm:" + name,
			Publisher:   o.Package.Publisher.Username,
			Kind:        kindFromKeywords(o.Package.Keywords),
			Downloads:   o.Downloads.Monthly,
			Updated:     o.Package.Date,
		})
	}
	return out, nil
}

func kindFromKeywords(kws []string) string {
	joined := strings.ToLower(strings.Join(kws, " "))
	switch {
	case strings.Contains(joined, "extension"):
		return "extension"
	case strings.Contains(joined, "skill"):
		return "skill"
	case strings.Contains(joined, "theme"):
		return "theme"
	default:
		return "package"
	}
}
