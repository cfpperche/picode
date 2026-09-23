package skills

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Source is where a skill comes from, parsed from what a person typed.
type Source struct {
	Kind  string `json:"kind"` // github | well-known | local
	Input string `json:"input"`
	// github
	Owner string `json:"owner,omitempty"`
	Repo  string `json:"repo,omitempty"`
	Ref   string `json:"ref,omitempty"`
	Sub   string `json:"sub,omitempty"` // a folder inside the repository
	// well-known: the site's origin
	Origin string `json:"origin,omitempty"`
	// local: an absolute folder
	Dir string `json:"dir,omitempty"`
}

// Lock fields as the skills CLI writes them for this source.
func (s Source) lockSource() (source, sourceType, sourceURL string) {
	switch s.Kind {
	case "github":
		return s.Owner + "/" + s.Repo, "github", "https://github.com/" + s.Owner + "/" + s.Repo + ".git"
	case "well-known":
		return strings.TrimPrefix(strings.TrimPrefix(s.Origin, "https://"), "http://"), "well-known", s.Origin + wellKnownPath
	default:
		return s.Dir, "local", ""
	}
}

var (
	ownerRepo = regexp.MustCompile(`^([A-Za-z0-9](?:[A-Za-z0-9-]{0,38})?)/([A-Za-z0-9._-]{1,100})((?:/[A-Za-z0-9._@-]+)*)?(?:#([A-Za-z0-9._/-]{1,200}))?$`)
	// ErrSource is an input PiCode cannot read as a source.
	ErrSource = errors.New("not a skill source")
)

const wellKnownPath = "/.well-known/agent-skills/index.json"

// ParseSource reads "owner/repo[/path][#ref]", a GitHub URL, an https://
// site (its .well-known index), or a local folder.
func ParseSource(input, homeDir string) (Source, error) {
	in := strings.TrimSpace(input)
	src := Source{Input: in}
	switch {
	case in == "":
		return src, fmt.Errorf("%w: empty", ErrSource)
	case strings.HasPrefix(in, "/") || strings.HasPrefix(in, "~/") || strings.HasPrefix(in, "./"):
		dir := in
		if strings.HasPrefix(dir, "~/") {
			dir = filepath.Join(homeDir, dir[2:])
		}
		if !filepath.IsAbs(dir) {
			return src, fmt.Errorf("%w: a local folder must be an absolute path", ErrSource)
		}
		src.Kind, src.Dir = "local", filepath.Clean(dir)
		return src, nil
	case strings.HasPrefix(in, "https://") || strings.HasPrefix(in, "http://"):
		u, err := url.Parse(in)
		if err != nil || u.Host == "" || u.User != nil {
			return src, fmt.Errorf("%w: %s", ErrSource, in)
		}
		if strings.EqualFold(u.Hostname(), "github.com") {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) < 2 {
				return src, fmt.Errorf("%w: a GitHub URL names owner/repo", ErrSource)
			}
			src.Kind, src.Owner, src.Repo = "github", parts[0], strings.TrimSuffix(parts[1], ".git")
			// /tree/<ref>/<path>: the first segment after tree is the ref.
			if len(parts) > 3 && parts[2] == "tree" {
				src.Ref = parts[3]
				src.Sub = strings.Join(parts[4:], "/")
			}
			return src, nil
		}
		if u.Scheme != "https" {
			return src, fmt.Errorf("%w: a site must be https", ErrSource)
		}
		src.Kind, src.Origin = "well-known", "https://"+u.Host
		return src, nil
	}
	m := ownerRepo.FindStringSubmatch(in)
	if m == nil {
		return src, fmt.Errorf("%w: use owner/repo, a GitHub or https:// URL, or a folder path", ErrSource)
	}
	src.Kind, src.Owner, src.Repo = "github", m[1], strings.TrimSuffix(m[2], ".git")
	src.Sub = strings.Trim(m[3], "/")
	src.Ref = m[4]
	return src, nil
}

// Limits for what a source may bring in. A repository tarball carries more
// than the skill it holds; one installed skill obeys the digest limits.
var (
	MaxDownload  int64 = 50 << 20
	MaxExtracted int64 = 200 << 20
	MaxFiles           = 20000
)

// Fetcher downloads sources. Its client refuses private addresses unless a
// test swaps it (httptest serves on loopback).
type Fetcher struct {
	Client   *http.Client
	Codeload string // https://codeload.github.com
}

func NewFetcher() *Fetcher {
	return &Fetcher{Client: publicClient(), Codeload: "https://codeload.github.com"}
}

func privateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// publicClient is the SSRF guard: a skill source is a public site, so a name
// that resolves to a private address is refused at dial time, which also
// covers redirects.
func publicClient() *http.Client {
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("%s has no addresses", host)
		}
		for _, ia := range ips {
			if privateIP(ia.IP) {
				return nil, fmt.Errorf("%s resolves to a private address", host)
			}
		}
		var d net.Dialer
		return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return errors.New("redirect left https")
			}
			return nil
		},
	}
}

func (f *Fetcher) get(ctx context.Context, u string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "picode (https://github.com/cfpperche/picode)")
	res, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s: not found", u)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", u, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s is larger than %d MiB", u, limit>>20)
	}
	return body, nil
}

// Fetch materialises a source into dir (created empty by the caller) and
// returns notes about what was skipped.
func (f *Fetcher) Fetch(ctx context.Context, src Source, dir string) ([]string, error) {
	switch src.Kind {
	case "github":
		ref := src.Ref
		if ref == "" {
			ref = "HEAD"
		}
		u := strings.TrimRight(f.Codeload, "/") + "/" + src.Owner + "/" + src.Repo + "/tar.gz/" + url.PathEscape(ref)
		body, err := f.get(ctx, u, MaxDownload)
		if err != nil {
			return nil, err
		}
		return extractTarGz(body, dir, true)
	case "well-known":
		return f.fetchWellKnown(ctx, src, dir)
	case "local":
		return copyTree(src.Dir, dir)
	}
	return nil, ErrSource
}

type wellKnownIndex struct {
	Schema string `json:"$schema"`
	Skills []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
		URL         string `json:"url"`
		Digest      string `json:"digest"`
	} `json:"skills"`
}

// fetchWellKnown reads the discovery index (the agentskills.io discovery
// RFC, schema 0.2.0) and every skill it lists, each checked against its
// digest: a mismatch refuses the whole source.
func (f *Fetcher) fetchWellKnown(ctx context.Context, src Source, dir string) ([]string, error) {
	base := src.Origin + wellKnownPath
	body, err := f.get(ctx, base, 1<<20)
	if err != nil {
		return nil, err
	}
	var idx wellKnownIndex
	if err := json.Unmarshal(body, &idx); err != nil {
		return nil, fmt.Errorf("%s is not a skills index: %v", base, err)
	}
	if len(idx.Skills) == 0 {
		return nil, fmt.Errorf("%s lists no skills", base)
	}
	if len(idx.Skills) > 100 {
		idx.Skills = idx.Skills[:100]
	}
	baseURL, _ := url.Parse(base)
	var notes []string
	for _, s := range idx.Skills {
		if !validName(s.Name) {
			notes = append(notes, "skipped an entry with an invalid name: "+s.Name)
			continue
		}
		ref, err := url.Parse(s.URL)
		if err != nil {
			return nil, fmt.Errorf("%s: bad url", s.Name)
		}
		u := baseURL.ResolveReference(ref).String()
		data, err := f.get(ctx, u, MaxDownload)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		want := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s.Digest)), "sha256:")
		if want == "" || hex.EncodeToString(sum[:]) != want {
			return nil, fmt.Errorf("%s: the download does not match the digest in the index — corrupted or tampered content", s.Name)
		}
		target := filepath.Join(dir, s.Name)
		switch s.Type {
		case "skill-md":
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(target, "SKILL.md"), data, 0o644); err != nil {
				return nil, err
			}
		case "archive":
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
			var n []string
			if strings.HasSuffix(strings.ToLower(ref.Path), ".zip") {
				n, err = extractZip(data, target)
			} else {
				n, err = extractTarGz(data, target, false)
			}
			if err != nil {
				return nil, fmt.Errorf("%s: %w", s.Name, err)
			}
			notes = append(notes, n...)
		default:
			notes = append(notes, "skipped "+s.Name+": unknown type "+s.Type)
		}
	}
	return notes, nil
}

// safeRel refuses a path that is absolute or leaves its root.
func safeRel(name string) (string, error) {
	n := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if n == "." || n == "" {
		return "", nil
	}
	if path.IsAbs(n) || n == ".." || strings.HasPrefix(n, "../") || strings.Contains(n, "/../") || filepath.VolumeName(n) != "" {
		return "", fmt.Errorf("the archive has a path outside its folder: %s", name)
	}
	return n, nil
}

type budget struct {
	files int
	bytes int64
}

func (b *budget) take(size int64) error {
	b.files++
	b.bytes += size
	if b.files > MaxFiles {
		return fmt.Errorf("the source has more than %d files", MaxFiles)
	}
	if b.bytes > MaxExtracted {
		return fmt.Errorf("the source unpacks to more than %d MiB", MaxExtracted>>20)
	}
	return nil
}

func writeLimited(dst string, r io.Reader, b *budget, size int64) error {
	if err := b.take(size); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(r, size+1))
	cerr := out.Close()
	if err != nil {
		return err
	}
	if n > size {
		return errors.New("an archive entry is larger than it declares")
	}
	return cerr
}

// extractTarGz unpacks regular files and folders only: links and devices are
// skipped (never materialised) and named in the notes.
func extractTarGz(data []byte, dir string, stripFirst bool) ([]string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("not a .tar.gz archive: %v", err)
	}
	tr := tar.NewReader(gz)
	var b budget
	skipped := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("broken archive: %v", err)
		}
		name := h.Name
		if stripFirst {
			if i := strings.Index(name, "/"); i >= 0 {
				name = name[i+1:]
			} else {
				continue
			}
		}
		rel, err := safeRel(name)
		if err != nil {
			return nil, err
		}
		if rel == "" {
			continue
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(rel)), 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			if err := writeLimited(filepath.Join(dir, filepath.FromSlash(rel)), tr, &b, h.Size); err != nil {
				return nil, err
			}
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
		default:
			skipped++
		}
	}
	if skipped > 0 {
		return []string{fmt.Sprintf("skipped %d links or special files in the source", skipped)}, nil
	}
	return nil, nil
}

func extractZip(data []byte, dir string) ([]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("not a .zip archive: %v", err)
	}
	var b budget
	skipped := 0
	for _, f := range zr.File {
		rel, err := safeRel(f.Name)
		if err != nil {
			return nil, err
		}
		if rel == "" {
			continue
		}
		mode := f.Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(rel)), 0o755); err != nil {
				return nil, err
			}
		case mode.IsRegular():
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			err = writeLimited(filepath.Join(dir, filepath.FromSlash(rel)), rc, &b, int64(f.UncompressedSize64))
			rc.Close()
			if err != nil {
				return nil, err
			}
		default:
			skipped++
		}
	}
	if skipped > 0 {
		return []string{fmt.Sprintf("skipped %d links or special files in the source", skipped)}, nil
	}
	return nil, nil
}

// copyTree copies regular files and folders from src into dst; links are
// skipped, .git and node_modules are not copied.
func copyTree(src, dst string) ([]string, error) {
	st, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", src, err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", src)
	}
	var b budget
	skipped := 0
	err = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if !d.Type().IsRegular() {
			skipped++
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		return writeLimited(filepath.Join(dst, rel), in, &b, info.Size())
	})
	if err != nil {
		return nil, err
	}
	if skipped > 0 {
		return []string{fmt.Sprintf("skipped %d links or special files in the source", skipped)}, nil
	}
	return nil, nil
}
