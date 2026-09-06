package llama

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// ExecutionProfile describes a PiCode-owned local router without secrets or
// arbitrary command fragments. It is intentionally previewable before a
// lifecycle job is added.
type ExecutionProfile struct {
	Binary     string `json:"binary"`
	ModelsDir  string `json:"modelsDir"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Context    int    `json:"context"`
	GPULayers  int    `json:"gpuLayers"`
	Jinja      bool   `json:"jinja"`
	NoAutoLoad bool   `json:"noAutoLoad"`
}

func (p ExecutionProfile) Validate() error {
	if p.Binary == "" || (filepath.Base(p.Binary) != "llama-server" && filepath.Base(p.Binary) != "llama-server.exe") {
		return fmt.Errorf("binary must be a selected llama-server executable")
	}
	if p.ModelsDir == "" || filepath.IsAbs(p.ModelsDir) == false {
		return fmt.Errorf("models directory must be an absolute path")
	}
	if p.Host != "127.0.0.1" && p.Host != "::1" {
		return fmt.Errorf("owned service must bind to loopback")
	}
	if p.Port < 1024 || p.Port > 65535 || p.Context < 512 || p.Context > 131072 || p.GPULayers < 0 {
		return fmt.Errorf("execution settings are outside the supported range")
	}
	return nil
}

func (p ExecutionProfile) Args() ([]string, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	a := []string{p.Binary, "--models-dir", p.ModelsDir, "--host", p.Host, "--port", strconv.Itoa(p.Port), "--ctx-size", strconv.Itoa(p.Context)}
	if p.Jinja {
		a = append(a, "--jinja")
	}
	if p.NoAutoLoad {
		a = append(a, "--no-models-autoload")
	}
	if p.GPULayers > 0 {
		a = append(a, "--n-gpu-layers", strconv.Itoa(p.GPULayers))
	}
	return a, nil
}
