package gitgraph

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// Blob serves one file's raw contents at one revision — the visual preview
// behind a binary row in a git diff card (screenshots, renders, recordings).
// The text diff cannot show these, and "Binary file — no text diff." is where
// useful answers go to die.
//
// Errors are sentinels so the HTTP layer can map them without string
// matching: ErrNoRepo, ErrBadHash and ErrBadPath are caller mistakes;
// ErrNoBlob and ErrTooBig describe the asset.
var (
	ErrNoRepo  = errors.New("not a git repository")
	ErrBadHash = errors.New("bad revision")
	ErrBadPath = errors.New("bad path")
	ErrNoBlob  = errors.New("no such file at that revision")
	ErrTooBig  = errors.New("file too large to preview")
)

// MaxBlobBytes caps one previewed asset. Same ceiling as the working-tree
// /blob endpoint: a 4K render can be tens of megabytes, and a preview that
// streams a movie into a diff card helps nobody.
var MaxBlobBytes = 32 << 20

// Blob returns the contents of path as it exists in revision hash. hash is a
// full object name (isHash) or the literal "HEAD"; path is repo-relative.
// The bytes come back exactly as committed — no newline trimming, unlike the
// text-oriented git() helper, or every PNG would gain corruption for free.
func Blob(dir, hash, path string) ([]byte, error) {
	if Key(dir) == "" {
		return nil, ErrNoRepo
	}
	if !validBlobHash(hash) {
		return nil, ErrBadHash
	}
	if hash == "HEAD" {
		// Resolve before use: rev-parse answers for an empty repository
		// with nothing hash-like, which then fails isHash below.
		hash = git(dir, "rev-parse", "HEAD")
	}
	if !isHash(hash) {
		return nil, ErrBadHash
	}
	if !validBlobPath(path) {
		return nil, ErrBadPath
	}
	// Size first, fetch second: a cap discovered after reading 2 GB into
	// memory is not a cap. cat-file -s answers in one spawn.
	if n := git(dir, "cat-file", "-s", hash+":"+path); n != "" {
		if size, err := strconv.Atoi(n); err == nil && size > MaxBlobBytes {
			return nil, ErrTooBig
		}
	}
	b, err := gitBytes(dir, "cat-file", "blob", hash+":"+path)
	if err != nil {
		return nil, ErrNoBlob
	}
	return b, nil
}

// validBlobHash accepts a full object name or the HEAD alias — the two
// shapes the graph's previews legitimately ask for.
func validBlobHash(h string) bool {
	return h == "HEAD" || isHash(h)
}

// validBlobPath refuses the shapes that should never reach a git command
// line or leave the repository: empty, flag-looking, absolute, or climbing
// out with a ".." element. A subdirectory path is fine — git resolves it.
func validBlobPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "-") || strings.HasPrefix(p, "/") {
		return false
	}
	if strings.ContainsAny(p, "\x00") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

// gitBytes runs git and returns raw stdout, byte-exact: a blob that ends in
// 0x0A must come back with it, so nothing here trims. Exit errors mean "no
// such blob": a missing path, a directory where a file was expected, a
// repository without that object — the caller's answer is the same either way.
func gitBytes(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
