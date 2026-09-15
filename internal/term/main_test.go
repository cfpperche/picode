package term

import (
	"os"
	"testing"

	"github.com/cfpperche/picode/internal/tmuxtest"
)

// Bridge tests start real tmux sessions; the private namespace keeps them
// off the user's server (see internal/tmuxtest).
func TestMain(m *testing.M) { os.Exit(tmuxtest.Main(m)) }
