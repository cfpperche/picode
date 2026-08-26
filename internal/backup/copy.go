package backup

import (
	"io"
	"os"
	"path/filepath"
)

func putFile(src, dst, prev string, mode os.FileMode, files *[]FileEnt) error {
	sum, size, err := hashFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	linked := false
	if prev != "" {
		if ps, err := os.Stat(prev); err == nil && ps.Size() == size {
			if psum, _, err := hashFile(prev); err == nil && psum == sum {
				if err := os.Link(prev, dst); err == nil {
					linked = true
				}
			}
		}
	}
	if !linked {
		if err := copyRegular(src, dst, mode); err != nil {
			return err
		}
	}
	rel := dst
	*files = append(*files, FileEnt{Path: rel, Size: size, SHA: sum, Link: linked})
	return nil
}

func copyRegular(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func walkCopy(srcDir, dstDir, prevDir string, files *[]FileEnt) error {
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		prev := ""
		if prevDir != "" {
			prev = filepath.Join(prevDir, rel)
		}
		mode := info.Mode().Perm()
		return putFile(path, filepath.Join(dstDir, rel), prev, mode, files)
	})
}
