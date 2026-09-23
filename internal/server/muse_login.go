package server

import (
	"net/http"
	"os/exec"
	"strings"

	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/credentials"
)

// Muse's sign-in through PiCode's GUI (ADR-0195). `muse login` is Meta's
// device-code flow and runs without a terminal (measured on 1.3.0): it prints
// the page and the code and waits, then writes Muse's own auth.json. Muse
// keeps one credential — a Meta account login or a Meta API key — so a key is
// the vault row made live by Use, which files the login it replaces first.

var museLogin = deviceLogin{name: "Muse"}

func handleMuseLoginStart(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bin, err := exec.LookPath("muse")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Muse is not installed on this machine.")
			return
		}
		museLogin.start(w, bin, []string{"login"}, []string{"BROWSER=true", "NO_COLOR=1"}, func() {
			fileCLILogin("muse", "meta-ai")
		})
	}
}

// fileCLILogin files the login a CLI holds in its own file into the vault,
// when there is one (the vault keeps one row per credential).
func fileCLILogin(cli, provider string) {
	if login, ok := clicreds.DetectProvider(cli, provider); ok {
		// A store that names its account only as the identity (Muse's
		// user_email) still gets that name on the row, not "Account N".
		label := login.Label
		if label == "" && strings.Contains(login.Identity, "@") {
			label = login.Identity
		}
		_, _ = credentials.Default().Import(login.Provider, login.Cred, label, "imported:"+cli, login.Identity)
	}
}
