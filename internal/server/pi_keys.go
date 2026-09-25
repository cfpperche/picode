package server

import (
	"os"
	"runtime"

	"github.com/cfpperche/picode/internal/pikeys"
)

// piKeysReport is pi's own key report, which /api/cli-keys?cli=pi wraps. File is this machine's real
// path, so the pane can name what it writes, and Platform picks the default pi
// actually binds here: nine actions carry a Windows or WSL alternate that the
// default column alone would misreport (pikeys.Catalog's Alt).
type piKeysReport struct {
	Actions  []pikeys.Action     `json:"actions"`
	User     map[string][]string `json:"user"`
	File     string              `json:"file"`
	Exists   bool                `json:"exists"`
	Platform string              `json:"platform"`
}

// piKeysPlatform names the host the way the catalog's Alt map does: "wsl"
// before GOOS, because a WSL distro is linux with pi's WSL row.
func piKeysPlatform() string {
	if isWSL() {
		return "wsl"
	}
	return runtime.GOOS
}

func piKeysSnapshot() (piKeysReport, error) {
	user, err := pikeys.LoadUser()
	if err != nil {
		return piKeysReport{}, err
	}
	path := pikeys.File()
	exists := false
	if path != "" {
		_, statErr := os.Stat(path)
		exists = statErr == nil
	}
	return piKeysReport{
		Actions:  pikeys.Catalog,
		User:     user,
		File:     path,
		Exists:   exists,
		Platform: piKeysPlatform(),
	}, nil
}
