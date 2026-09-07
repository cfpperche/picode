package llama

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// ExecutionProfile describes a PiCode-owned local router without secrets or
// arbitrary command fragments. It is intentionally previewable before a
// lifecycle job is added.
type ExecutionProfile struct {
	Binary       string `json:"binary"`
	BinarySHA256 string `json:"binarySHA256"`
	Threads      int    `json:"threads"`
	ModelsDir    string `json:"modelsDir"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Context      int    `json:"context"`
	GPULayers    int    `json:"gpuLayers"`
	Jinja        bool   `json:"jinja"`
	NoAutoLoad   bool   `json:"noAutoLoad"`
}

func (p ExecutionProfile) Validate() error {
	if !filepath.IsAbs(p.Binary) || (filepath.Base(p.Binary) != "llama-server" && filepath.Base(p.Binary) != "llama-server.exe") {
		return fmt.Errorf("binary must be a selected llama-server executable")
	}
	digest, err := hex.DecodeString(p.BinarySHA256)
	if err != nil || len(digest) != sha256.Size {
		return fmt.Errorf("binary SHA-256 must contain 64 hexadecimal characters")
	}
	if p.ModelsDir == "" || filepath.IsAbs(p.ModelsDir) == false {
		return fmt.Errorf("models directory must be an absolute path")
	}
	if p.Host != "127.0.0.1" && p.Host != "::1" {
		return fmt.Errorf("owned service must bind to loopback")
	}
	if p.Port < 1024 || p.Port > 65535 || p.Context < 512 || p.Context > 131072 || p.GPULayers < 0 || p.Threads < 1 || p.Threads > 4 {
		return fmt.Errorf("execution settings are outside the supported range")
	}
	return nil
}

// VerifyBinary checks the current Linux/WSL file against the profile's pin.
// The caller must obtain the pin from a trusted installation manifest. This
// does not establish provenance or prevent replacement after verification;
// a future launcher must revalidate under its lifecycle lock before execution.
func (p ExecutionProfile) VerifyBinary() error {
	if err := p.Validate(); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(p.Binary)
	if err != nil {
		return fmt.Errorf("resolve binary: %w", err)
	}
	if resolved != filepath.Clean(p.Binary) {
		return fmt.Errorf("binary path must not contain symlinks")
	}
	info, err := os.Lstat(p.Binary)
	if err != nil {
		return fmt.Errorf("inspect binary: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("binary must be a regular executable file")
	}
	f, err := os.Open(p.Binary)
	if err != nil {
		return fmt.Errorf("open binary: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened binary: %w", err)
	}
	if !os.SameFile(info, opened) {
		return fmt.Errorf("binary changed during verification")
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash binary: %w", err)
	}
	want, _ := hex.DecodeString(p.BinarySHA256)
	if hex.EncodeToString(h.Sum(nil)) != hex.EncodeToString(want) {
		return fmt.Errorf("binary SHA-256 does not match profile")
	}
	return nil
}

func (p ExecutionProfile) Args() ([]string, error) {
	if err := p.VerifyBinary(); err != nil {
		return nil, err
	}
	a := []string{p.Binary, "--models-dir", p.ModelsDir, "--host", p.Host, "--port", strconv.Itoa(p.Port), "--ctx-size", strconv.Itoa(p.Context), "--threads", strconv.Itoa(p.Threads), "--threads-batch", strconv.Itoa(p.Threads), "--n-gpu-layers", strconv.Itoa(p.GPULayers)}
	if p.Jinja {
		a = append(a, "--jinja")
	} else {
		a = append(a, "--no-jinja")
	}
	if p.NoAutoLoad {
		a = append(a, "--no-models-autoload")
	} else {
		a = append(a, "--models-autoload")
	}
	return a, nil
}
