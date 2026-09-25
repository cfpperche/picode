package llamaservice

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func releasePin(version string) (string, error) {
	pins := map[string]map[string]string{
		"b10809": {"amd64": "5e34434ddc6d03cd1584f403201aff0d4bd1a5793a72ff7e286532dfd1e4b941", "arm64": "f2b7333971e1b7b42e9268bfdbfa30f5f56e2897156084d2251385df94aec358"},
		"b10826": {"amd64": "c708a8d84853c86ad3400428b5d6ef2b504e297d98ccdcf8362195c6638b7fd9", "arm64": "9947a1d80a82c4e5c89df2f68f8d09ddef184c768ced7f68b4d32f8e5526332a"},
	}
	pin := pins[version][runtime.GOARCH]
	if pin == "" {
		return "", errors.New("Choose a verified release for this platform.")
	}
	return pin, nil
}
func hashRegular(path string) (string, int64, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", 0, err
	}
	if resolved != filepath.Clean(path) {
		return "", 0, errors.New("Symbolic links are not eligible.")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, errors.New("Expected a regular file.")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	if !os.SameFile(info, opened) {
		return "", 0, errors.New("File changed during verification.")
	}
	h := sha256.New()
	n, err := io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), n, err
}
func verifyRelease(r *Release) error {
	if r == nil || len(r.Files) == 0 {
		return errors.New("Install a verified release first.")
	}
	for name, pin := range r.Files {
		if filepath.Base(name) != name {
			return errors.New("Invalid release manifest.")
		}
		// Never from memory: this guards every start of the binary.
		h, _, err := hashRegular(filepath.Join(r.Dir, name))
		if err != nil || h != pin {
			return errors.New("Installed files changed. Reinstall a verified release before starting.")
		}
	}
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		return err
	}
	if len(entries) != len(r.Files) {
		return errors.New("Unexpected files in the installed release.")
	}
	return nil
}
func (s *Service) install(version string) (*Release, error) {
	pin, err := releasePin(version)
	if err != nil {
		return nil, err
	}
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	name := "llama-" + version + "-bin-ubuntu-" + arch + ".tar.gz"
	path := filepath.Join(s.root, "cache", name)
	// A cached archive is reusable only if it matches the embedded official pin.
	h, _, e := hashRegular(path)
	if e != nil || h != pin {
		ctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
		defer cancel()
		req, e := http.NewRequestWithContext(ctx, "GET", "https://github.com/ggml-org/llama.cpp/releases/download/"+version+"/"+name, nil)
		if e != nil {
			return nil, e
		}
		client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 3 * time.Minute}
		defer client.CloseIdleConnections()
		res, e := client.Do(req)
		if e != nil {
			return nil, errors.New("Could not download the release. Check the connection and retry.")
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return nil, errors.New("The release download is unavailable.")
		}
		f, e := os.CreateTemp(filepath.Join(s.root, "cache"), ".install-")
		if e != nil {
			return nil, e
		}
		temp := f.Name()
		defer os.Remove(temp)
		progress := &downloadProgress{service: s, total: res.ContentLength}
		_, e = io.Copy(f, io.TeeReader(io.LimitReader(res.Body, 128<<20), progress))
		closeErr := f.Close()
		if e != nil {
			return nil, e
		}
		if closeErr != nil {
			return nil, closeErr
		}
		got, _, e := hashRegular(temp)
		if e != nil || got != pin {
			return nil, errors.New("Release checksum verification failed. The current version was retained.")
		}
		// Never replace an unowned file, even when its filename matches a release.
		if _, e = os.Lstat(path); !os.IsNotExist(e) {
			return nil, errors.New("An unexpected archive occupies the download path; it was retained.")
		}
		if e = os.Rename(temp, path); e != nil {
			return nil, e
		}
	}
	s.mu.Lock()
	s.doc.Archives[name] = pin
	err = s.save()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(s.root, "release-"+version+"-")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(dir)
		}
	}()
	files, err := extractRelease(path, dir)
	if err != nil {
		return nil, err
	}
	release := &Release{Version: version, Dir: dir, Files: files}
	if err = verifyRelease(release); err != nil {
		return nil, err
	}
	// Hashes authenticate the archive; version and required flags establish compatibility.
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()
	output, err := inspectExecutable(ctx, dir, "--version")
	if err != nil || !strings.Contains(string(output), strings.TrimPrefix(version, "b")) {
		return nil, errors.New("The downloaded executable is incompatible with this host or version.")
	}
	output, err = inspectExecutable(ctx, dir, "--help")
	if err != nil {
		return nil, errors.New("Could not inspect the downloaded server.")
	}
	for _, flag := range []string{"--models-dir", "--no-models-autoload", "--no-jinja", "--threads-batch", "--n-gpu-layers"} {
		if !strings.Contains(string(output), flag) {
			return nil, fmt.Errorf("The downloaded server does not support %s.", flag)
		}
	}
	keep = true
	return release, nil
}
func extractRelease(archive, dir string) (map[string]string, error) {
	f, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	links := map[string]string{}
	seen := map[string]bool{}
	total := int64(0)
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		name := filepath.Base(h.Name)
		if name == "." || name == ".." || strings.Contains(h.Name, "../") || filepath.IsAbs(h.Name) || seen[name] {
			return nil, errors.New("Invalid archive entry.")
		}
		seen[name] = true
		if h.Typeflag == tar.TypeSymlink {
			if filepath.Base(h.Linkname) != h.Linkname || h.Linkname == ".." {
				return nil, errors.New("Unsafe archive link.")
			}
			links[name] = h.Linkname
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size < 0 || h.Size > 256<<20 {
			return nil, errors.New("Unsupported archive entry.")
		}
		total += h.Size
		if total > 512<<20 {
			return nil, errors.New("Release exceeds the extraction limit.")
		}
		out, e := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if e != nil {
			return nil, e
		}
		_, e = io.CopyN(out, tr, h.Size)
		ce := out.Close()
		if e != nil {
			return nil, e
		}
		if ce != nil {
			return nil, ce
		}
	}
	// Materialize internal library links as regular pinned files.
	for len(links) > 0 {
		progress := false
		for name, target := range links {
			if _, pending := links[target]; pending {
				continue
			}
			data, e := os.ReadFile(filepath.Join(dir, target))
			if e != nil {
				return nil, errors.New("Archive link has no regular target.")
			}
			if e = os.WriteFile(filepath.Join(dir, name), data, 0700); e != nil {
				return nil, e
			}
			delete(links, name)
			progress = true
		}
		if !progress {
			return nil, errors.New("Cyclic archive links.")
		}
	}
	files := map[string]string{}
	for name := range seen {
		h, _, e := hashRegular(filepath.Join(dir, name))
		if e != nil {
			return nil, e
		}
		files[name] = h
	}
	if files["llama-server"] == "" {
		return nil, errors.New("Release has no llama-server executable.")
	}
	return files, nil
}
