package server

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	webappTitleTag     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	webappLinkIcon     = regexp.MustCompile(`(?is)<link[^>]+rel=["']?[^"'> ]*icon[^"'> ]*["']?[^>]*>`)
	webappLinkManifest = regexp.MustCompile(`(?is)<link[^>]+rel=["']?[^"'> ]*manifest[^"'> ]*["']?[^>]*>`)
)

func webappTitle(body []byte) string {
	m := webappTitleTag.FindSubmatch(body)
	if m == nil {
		return ""
	}
	title := strings.Join(strings.Fields(html.UnescapeString(string(m[1]))), " ")
	if len([]rune(title)) > webappTitleCap {
		title = strings.TrimSpace(string([]rune(title)[:webappTitleCap])) + "…"
	}
	return title
}

func webappDeriveName(u *url.URL, title string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	host = strings.TrimPrefix(host, "www.")
	name := strings.Join(strings.Fields(html.UnescapeString(strings.Split(host, ":")[0])), " ")
	if len([]rune(name)) > webappTitleCap {
		name = strings.TrimSpace(string([]rune(name)[:webappTitleCap]))
	}
	return name
}

func webappAttr(tag, name string) string {
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(name) + `\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	m := re.FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	for _, g := range m[2:] {
		if g != "" {
			return g
		}
	}
	return ""
}

func webappFindIconURL(base *url.URL, body []byte) (string, bool) {
	if base == nil {
		return "", false
	}
	best := ""
	bestScore := -1
	for _, tag := range webappLinkIcon.FindAllSubmatch(body, 16) {
		tagHTML := string(tag[0])
		href := html.UnescapeString(webappAttr(tagHTML, "href"))
		if href == "" || strings.HasPrefix(strings.TrimSpace(href), "data:") {
			continue
		}
		ref, err := url.Parse(strings.TrimSpace(href))
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(ref)
		if resolved == nil || (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" {
			continue
		}
		if !isSameWebappOrigin(base, resolved) {
			continue
		}
		score := 0
		if strings.Contains(resolved.Path, "apple-touch-icon") {
			score += 3
		}
		if t := webappAttr(tagHTML, "type"); strings.Contains(t, "svg") || strings.Contains(t, "png") {
			score += 2
		}
		if s := webappAttr(tagHTML, "sizes"); strings.Contains(s, "x") {
			score += 1
		}
		if score > bestScore {
			best, bestScore = resolved.String(), score
		}
	}
	if best != "" {
		return best, true
	}
	fallback := *base
	fallback.Path = "/favicon.ico"
	fallback.RawQuery = ""
	fallback.Fragment = ""
	return fallback.String(), true
}

func isSameWebappOrigin(a, b *url.URL) bool {
	return a.Scheme == b.Scheme && a.Host == b.Host
}

func webappFetchIcon(ctx context.Context, client *http.Client, raw string) ([]byte, string, error) {
	res, err := webappFetch(ctx, client, raw, webappIconCap, false)
	if err != nil {
		return nil, "", err
	}
	if res.status < 200 || res.status >= 300 {
		return nil, "", errors.New("icon request was not successful")
	}
	ct := strings.TrimSpace(res.headers.Get("Content-Type"))
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil || mt == "" {
		mt = ""
	}
	ct = strings.ToLower(mt)
	data := res.body
	if len(data) == 0 {
		return nil, "", errors.New("empty icon body")
	}
	if !webappIconMimeOK(ct) {
		if ct != "" {
			return nil, "", errors.New("icon is not an image")
		}
		ct = http.DetectContentType(data)
		if !webappIconMimeOK(ct) || ct == "application/octet-stream" {
			return nil, "", errors.New("icon is not an image")
		}
	}
	if ct == "application/octet-stream" {
		return nil, "", errors.New("icon is not an image")
	}
	return data, ct, nil
}

func webappIconMimeOK(mt string) bool {
	switch mt {
	case "", "image/png", "image/jpeg", "image/gif", "image/webp", "image/svg+xml", "image/x-icon", "image/vnd.microsoft.icon", "image/bmp":
		return true
	}
	return false
}

type webappManifestIcon struct {
	Src   string `json:"src"`
	Sizes string `json:"sizes"`
	Type  string `json:"type"`
}

type webappManifest struct {
	Name      string               `json:"name"`
	ShortName string               `json:"short_name"`
	Icons     []webappManifestIcon `json:"icons"`
}

const (
	webappManifestCap    = 64 << 10
	webappManifestMaxLen = 200
)

func webappManifestLinkURL(base *url.URL, body []byte) (string, bool) {
	if base == nil {
		return "", false
	}
	for _, tag := range webappLinkManifest.FindAllSubmatch(body, 4) {
		tagHTML := string(tag[0])
		href := html.UnescapeString(webappAttr(tagHTML, "href"))
		if href == "" {
			continue
		}
		ref, err := url.Parse(strings.TrimSpace(href))
		if err != nil || ref.Scheme == "data" {
			continue
		}
		resolved := base.ResolveReference(ref)
		if resolved == nil || (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" {
			continue
		}
		return resolved.String(), true
	}
	return "", false
}

func webappManifestName(m webappManifest) string {
	name := strings.TrimSpace(m.Name)
	if name == "" {
		name = strings.TrimSpace(m.ShortName)
	}
	name = strings.Join(strings.Fields(html.UnescapeString(name)), " ")
	if len([]rune(name)) > webappManifestMaxLen {
		name = strings.TrimSpace(string([]rune(name)[:webappManifestMaxLen]))
	}
	return name
}

func webappManifestIconURL(m webappManifest, base *url.URL) string {
	best := ""
	bestScore := -1
	for _, icon := range m.Icons {
		src := strings.TrimSpace(icon.Src)
		if src == "" || strings.HasPrefix(src, "data:") {
			continue
		}
		ref, err := url.Parse(src)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(ref)
		if resolved == nil || (resolved.Scheme != "http" && resolved.Scheme != "https") || resolved.Host == "" {
			continue
		}
		if !isSameWebappOrigin(base, resolved) {
			continue
		}
		score := 0
		if s := strings.TrimSpace(icon.Sizes); s != "" && s != "any" {
			if w, _, ok := strings.Cut(s, "x"); ok {
				if n, err := strconv.Atoi(strings.TrimSpace(w)); err == nil && n >= 192 {
					score += 2
				}
			}
		}
		if t := strings.ToLower(strings.TrimSpace(icon.Type)); strings.Contains(t, "png") || strings.Contains(t, "svg") {
			score++
		}
		if score > bestScore {
			best, bestScore = resolved.String(), score
		}
	}
	return best
}

func webappFetchManifest(ctx context.Context, client *http.Client, raw string) (webappManifest, bool) {
	var m webappManifest
	res, err := webappFetch(ctx, client, raw, webappManifestCap, false)
	if err != nil {
		return m, false
	}
	if res.status >= 400 {
		return m, false
	}
	if err := json.Unmarshal(res.body, &m); err != nil {
		return webappManifest{}, false
	}
	return m, true
}

// webappLookupMetadata derives the tile identity from the fetched page:
// a same-origin PWA manifest wins (name, then icon resolved against the
// manifest URL), the <title> is the fallback. An empty name means the
// caller derives one from the host.
func webappLookupMetadata(ctx context.Context, client *http.Client, page webappFetchResult) (string, string) {
	if raw, ok := webappManifestLinkURL(page.finalURL, page.body); ok {
		if m, ok := webappFetchManifest(ctx, client, raw); ok {
			if name := webappManifestName(m); name != "" {
				base, err := url.Parse(raw)
				if err != nil {
					return name, ""
				}
				return name, webappManifestIconURL(m, base)
			}
		}
	}
	return webappTitle(page.body), ""
}
