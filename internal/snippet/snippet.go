// Package snippet runs conversation source blocks in an agent's working directory.
package snippet

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxCode = 100_000
	MaxOut  = 32_768
	Timeout = 15 * time.Second
)

// Spec is one runnable language.
type Spec struct {
	Ext  string
	Bin  []string // first found on PATH
	Args []string // before the file path
}

var langs = map[string]Spec{
	"bash":       {Ext: ".sh", Bin: []string{"bash"}, Args: nil},
	"sh":         {Ext: ".sh", Bin: []string{"bash", "sh"}, Args: nil},
	"shell":      {Ext: ".sh", Bin: []string{"bash", "sh"}, Args: nil},
	"python":     {Ext: ".py", Bin: []string{"python3", "python"}, Args: nil},
	"py":         {Ext: ".py", Bin: []string{"python3", "python"}, Args: nil},
	"javascript": {Ext: ".js", Bin: []string{"node"}, Args: nil},
	"js":         {Ext: ".js", Bin: []string{"node"}, Args: nil},
	"go":         {Ext: ".go", Bin: []string{"go"}, Args: []string{"run"}},
	"golang":     {Ext: ".go", Bin: []string{"go"}, Args: []string{"run"}},
}

func NormalizeLang(lang string) string {
	return strings.ToLower(strings.TrimSpace(lang))
}

// Runnable is true when PiCode knows how to run this fence tag.
func Runnable(lang string) bool {
	_, ok := langs[NormalizeLang(lang)]
	return ok
}

// Result is one run. No secrets.
type Result struct {
	OK       bool   `json:"ok"`
	Exit     int    `json:"exit"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timedOut"`
	Bin      string `json:"bin"`
}

func lookBin(names []string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

func prepGo(code string) string {
	if strings.Contains(code, "package ") {
		return code
	}
	return "package main\n\n" + code
}

// Run writes the snippet to cwd and executes it.
func Run(cwd, lang, code string) (Result, error) {
	lang = NormalizeLang(lang)
	spec, ok := langs[lang]
	if !ok {
		return Result{}, fmt.Errorf("cannot run %s here", lang)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return Result{}, fmt.Errorf("empty snippet")
	}
	if len(code) > MaxCode {
		return Result{}, fmt.Errorf("snippet is too long")
	}
	bin := lookBin(spec.Bin)
	if bin == "" {
		return Result{}, fmt.Errorf("%s is not on PATH", spec.Bin[0])
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if lang == "go" || lang == "golang" {
		code = prepGo(code)
	}
	f, err := os.CreateTemp(cwd, "picode-run-*"+spec.Ext)
	if err != nil {
		return Result{}, err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString(code); err != nil {
		_ = f.Close()
		return Result{}, err
	}
	if err := f.Close(); err != nil {
		return Result{}, err
	}
	_ = os.Chmod(path, 0o700)

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	args := append(append([]string{}, spec.Args...), path)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	res := Result{Bin: filepath.Base(bin), Stdout: clip(stdout.String()), Stderr: clip(stderr.String())}
	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.Exit = -1
		return res, nil
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.Exit = ee.ExitCode()
		} else {
			res.Exit = 1
			if res.Stderr == "" {
				res.Stderr = err.Error()
			}
		}
		return res, nil
	}
	res.OK = true
	return res, nil
}

func clip(s string) string {
	if len(s) <= MaxOut {
		return s
	}
	return s[:MaxOut] + "\n… truncated"
}
