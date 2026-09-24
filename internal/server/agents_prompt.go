package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/store"
)

// maxStartPrompt bounds a create's first prompt: it travels as one launch
// argument, and the drafts that use it are a paragraph, not a document.
const maxStartPrompt = 4000

// startPromptLaunch composes the launch of an agent created with a first
// prompt (Instructions → Draft with an agent, docs/architecture/
// cli-instructions.md). Pi without launch overrides is a managed agent and
// takes the prompt as its first queued task, so the overrides stay nil and
// the caller enqueues it. Every other CLI starts with the prompt through
// the same PromptArgs the brief handoff uses (ADR-0088), after the CLI's
// own configured arguments, so a user's flags survive. A resumed session
// already pins its arguments; the prompt is for the first launch only.
func startPromptLaunch(deps Deps, cliID, prompt string, ov *clilaunch.Overrides) (*clilaunch.Overrides, int, error) {
	if len(prompt) > maxStartPrompt {
		return nil, http.StatusBadRequest, errors.New("The first prompt is too long.")
	}
	id := strings.TrimSpace(cliID)
	if id == "" {
		id = store.CLIPi
	}
	cli, ok := clilaunch.Find(id)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("Unknown CLI.")
	}
	if cli.ID == store.CLIPi && ov == nil {
		return nil, 0, nil
	}
	if ov != nil && ov.Args != nil {
		return nil, http.StatusBadRequest, errors.New("A first prompt and launch arguments cannot be combined.")
	}
	p, ok := clisession.PrompterFor(cli.ID)
	if !ok {
		return nil, http.StatusConflict, errors.New(cli.Name + " can't start with a prompt yet.")
	}
	c, err := cliConfig(deps, cli.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	args := append(append([]string{}, c.Args...), p.PromptArgs(prompt, "")...)
	out := clilaunch.Overrides{}
	if ov != nil {
		out = *ov
	}
	out.Args = &args
	return &out, 0, nil
}
