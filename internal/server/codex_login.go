package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/clisettings"
)

// Codex's logins through PiCode's GUI (ADR-0191). Codex ships the API its own
// apps sign in with: `codex app-server` speaks JSON-RPC over stdio, and
// `account/login/start` takes exactly the doors its /login offers — ChatGPT in
// the browser (Codex serves the localhost callback itself), a device code, an
// API key, Amazon Bedrock by API key or by access keys (measured on 0.156.0).
// Codex writes its own auth.json (and, for Bedrock, config.toml), so PiCode
// never guesses a file shape: it drives the door and reports what came back.

// codexLoginTypes are the login/start types PiCode offers, with the fields
// each one carries. Anything else is refused before Codex is started.
var codexLoginTypes = map[string][]string{
	"chatgpt":                 nil,
	"chatgptDeviceCode":       nil,
	"apiKey":                  {"apiKey"},
	"amazonBedrock":           {"apiKey", "region"},
	"amazonBedrockAccessKeys": {"accessKeyId", "secretAccessKey", "region"},
}

// codexLoginWindow bounds a browser or device-code sign-in: past it the Codex
// process is stopped and the login reported as expired.
const codexLoginWindow = 10 * time.Minute

type codexLoginState struct {
	mu        sync.Mutex
	running   bool
	kind      string
	done      bool
	err       string
	cancel    context.CancelFunc
	startedAt time.Time
}

var codexLogin codexLoginState

// codexBinary locates the codex executable (tests put a fake on PATH).
func codexBinary() (string, error) {
	return exec.LookPath("codex")
}

// codexHome is Codex's own home: $CODEX_HOME, else ~/.codex.
func codexHome() string {
	if h := strings.TrimSpace(os.Getenv("CODEX_HOME")); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex")
}

// codexRPC is one app-server conversation.
type codexRPC struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan map[string]any
	closed chan struct{}
}

func startCodexRPC(ctx context.Context) (*codexRPC, error) {
	bin, err := codexBinary()
	if err != nil {
		return nil, errors.New("Codex is not installed on this machine.")
	}
	cmd := exec.CommandContext(ctx, bin, "app-server")
	cmd.Env = os.Environ()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r := &codexRPC{cmd: cmd, stdin: stdin, lines: make(chan map[string]any, 32), closed: make(chan struct{})}
	go func() {
		defer close(r.closed)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1<<16), 1<<22)
		for sc.Scan() {
			var msg map[string]any
			if json.Unmarshal(sc.Bytes(), &msg) == nil {
				select {
				case r.lines <- msg:
				default:
				}
			}
		}
	}()
	return r, nil
}

func (r *codexRPC) send(msg map[string]any) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = r.stdin.Write(append(raw, '\n'))
	return err
}

// await returns the response to id, or an error when the process ends or the
// deadline passes first.
func (r *codexRPC) await(id float64, timeout time.Duration) (map[string]any, error) {
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-r.lines:
			if v, ok := msg["id"].(float64); ok && v == id {
				if e, ok := msg["error"].(map[string]any); ok {
					m, _ := e["message"].(string)
					return nil, errors.New(codexErrorText(m))
				}
				res, _ := msg["result"].(map[string]any)
				return res, nil
			}
		case <-r.closed:
			return nil, errors.New("Codex stopped before it answered.")
		case <-deadline:
			return nil, errors.New("Codex did not answer in time.")
		}
	}
}

func (r *codexRPC) stop() {
	_ = r.stdin.Close()
	select {
	case <-r.closed:
	case <-time.After(3 * time.Second):
		if r.cmd.Process != nil {
			_ = r.cmd.Process.Kill()
		}
	}
	_ = r.cmd.Wait()
}

// codexErrorText keeps Codex's own message, minus protocol noise.
func codexErrorText(m string) string {
	m = strings.TrimSpace(m)
	if m == "" {
		return "Codex refused the sign-in."
	}
	return m
}

// openCodexRPC starts app-server and completes its handshake. Bedrock's
// login types sit behind the experimental API capability (measured).
func openCodexRPC(ctx context.Context) (*codexRPC, error) {
	r, err := startCodexRPC(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.send(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{
		"clientInfo":   map[string]any{"name": "picode", "version": "1"},
		"capabilities": map[string]any{"experimentalApi": true},
	}}); err != nil {
		r.stop()
		return nil, err
	}
	if _, err := r.await(1, 20*time.Second); err != nil {
		r.stop()
		return nil, err
	}
	_ = r.send(map[string]any{"method": "initialized"})
	return r, nil
}

// codexLoginParams checks and shapes the request body into login/start params.
func codexLoginParams(body map[string]string) (map[string]any, error) {
	kind := strings.TrimSpace(body["type"])
	fields, ok := codexLoginTypes[kind]
	if !ok {
		return nil, errors.New("Unknown sign-in method.")
	}
	params := map[string]any{"type": kind}
	for _, f := range fields {
		v := strings.TrimSpace(body[f])
		if v == "" {
			return nil, errors.New(codexFieldMessage(f))
		}
		if f == "region" && !claudeRegionRe.MatchString(v) {
			return nil, errors.New("An AWS region is required, like us-east-1.")
		}
		params[f] = v
	}
	if kind == "amazonBedrockAccessKeys" {
		if v := strings.TrimSpace(body["sessionToken"]); v != "" {
			params["sessionToken"] = v
		}
	}
	return params, nil
}

func codexFieldMessage(f string) string {
	switch f {
	case "apiKey":
		return "An API key is required."
	case "region":
		return "An AWS region is required, like us-east-1."
	case "accessKeyId", "secretAccessKey":
		return "An access key ID and its secret are required."
	}
	return "A field is missing."
}

// finishCodexLogin runs after Codex reports a successful sign-in. A login
// that is not Bedrock ends Bedrock's turn: Codex leaves
// `model_provider = "amazon-bedrock"` in config.toml when auth.json moves to
// another mode (measured on 0.156.0), which would point the next session at
// Bedrock with no Bedrock credentials.
func finishCodexLogin(kind string) error {
	if strings.HasPrefix(kind, "amazonBedrock") {
		return nil
	}
	return clearCodexBedrockProvider()
}

// clearCodexBedrockProvider removes config.toml's model_provider when, and
// only when, it names Codex's built-in Bedrock provider.
func clearCodexBedrockProvider() error {
	home := codexHome()
	if home == "" {
		return nil
	}
	path := filepath.Join(home, "config.toml")
	doc, err := clisettings.OpenDoc(path, clisettings.FormatTOML)
	if err != nil || !doc.Exists() {
		return nil
	}
	v, ok := doc.Scalar("model_provider")
	if s, _ := v.(string); !ok || s != "amazon-bedrock" {
		return nil
	}
	rev := doc.Revision()
	if err := doc.Remove("model_provider"); err != nil {
		return err
	}
	return doc.Save(rev)
}

// codexPlatformView is the Bedrock setup Codex is on, read from its files —
// never the key.
func codexPlatformView() (claudePlatformView, bool) {
	home := codexHome()
	if home == "" {
		return claudePlatformView{}, false
	}
	raw, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		return claudePlatformView{}, false
	}
	var auth struct {
		Mode   string `json:"auth_mode"`
		APIKey *struct {
			Region string `json:"region"`
		} `json:"bedrock_api_key"`
		AccessKeys *struct {
			Region string `json:"region"`
		} `json:"bedrock_access_keys"`
	}
	if json.Unmarshal(raw, &auth) != nil {
		return claudePlatformView{}, false
	}
	switch auth.Mode {
	case "bedrockApiKey":
		v := claudePlatformView{Kind: "bedrock", Name: "Amazon Bedrock", Auth: "bearer"}
		if auth.APIKey != nil {
			v.Region = auth.APIKey.Region
		}
		return v, true
	case "bedrockAccessKeys":
		v := claudePlatformView{Kind: "bedrock", Name: "Amazon Bedrock", Auth: "accessKey"}
		if auth.AccessKeys != nil {
			v.Region = auth.AccessKeys.Region
		}
		return v, true
	}
	return claudePlatformView{}, false
}

func handleCodexLoginStart(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "Invalid sign-in request.")
			return
		}
		params, err := codexLoginParams(body)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		kind := params["type"].(string)

		codexLogin.mu.Lock()
		if codexLogin.running {
			codexLogin.mu.Unlock()
			writeErr(w, http.StatusConflict, "A Codex sign-in is already in progress. Finish or cancel it first.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), codexLoginWindow)
		codexLogin.running, codexLogin.kind, codexLogin.done, codexLogin.err = true, kind, false, ""
		codexLogin.cancel, codexLogin.startedAt = cancel, time.Now()
		codexLogin.mu.Unlock()

		finish := func(errText string) {
			if errText == "" {
				if err := finishCodexLogin(kind); err != nil {
					errText = err.Error()
				}
			}
			codexLogin.mu.Lock()
			codexLogin.running, codexLogin.done, codexLogin.err = false, true, errText
			codexLogin.mu.Unlock()
			cancel()
		}

		rpc, err := openCodexRPC(ctx)
		if err != nil {
			finish(err.Error())
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if err := rpc.send(map[string]any{"id": 2, "method": "account/login/start", "params": params}); err != nil {
			rpc.stop()
			finish(err.Error())
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		res, err := rpc.await(2, 30*time.Second)
		if err != nil {
			rpc.stop()
			finish(err.Error())
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		// A key or Bedrock login is done when Codex answers: it has written
		// its files. A browser or device-code login finishes later, on
		// Codex's account/login/completed notification.
		if kind != "chatgpt" && kind != "chatgptDeviceCode" {
			rpc.stop()
			finish("")
			codexLogin.mu.Lock()
			errText := codexLogin.err
			codexLogin.mu.Unlock()
			if errText != "" {
				writeErr(w, http.StatusInternalServerError, errText)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"type": kind, "done": true})
			return
		}
		loginID, _ := res["loginId"].(string)
		go func() {
			defer rpc.stop()
			for {
				select {
				case msg := <-rpc.lines:
					if msg["method"] != "account/login/completed" {
						continue
					}
					p, _ := msg["params"].(map[string]any)
					if id, _ := p["loginId"].(string); loginID != "" && id != "" && id != loginID {
						continue
					}
					if ok, _ := p["success"].(bool); ok {
						finish("")
					} else {
						e, _ := p["error"].(string)
						finish(codexErrorText(e))
					}
					return
				case <-rpc.closed:
					finish("Codex stopped before the sign-in finished.")
					return
				case <-ctx.Done():
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						finish("The sign-in expired. Start it again.")
					} else {
						finish("The sign-in was cancelled.")
					}
					return
				}
			}
		}()
		out := map[string]any{"type": kind, "done": false}
		for _, k := range []string{"authUrl", "verificationUrl", "userCode"} {
			if v, ok := res[k].(string); ok && v != "" {
				out[k] = v
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCodexLoginStatus(w http.ResponseWriter, r *http.Request) {
	codexLogin.mu.Lock()
	defer codexLogin.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"pending": codexLogin.running,
		"done":    codexLogin.done,
		"error":   codexLogin.err,
		"type":    codexLogin.kind,
	})
}

func handleCodexLoginCancel(w http.ResponseWriter, r *http.Request) {
	codexLogin.mu.Lock()
	cancel := codexLogin.cancel
	running := codexLogin.running
	codexLogin.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleCodexPlatformDelete stops Codex using Bedrock: Codex's own logout
// removes the Bedrock credentials, and the model_provider line goes with them.
func handleCodexPlatformDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := codexPlatformView(); !ok {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		rpc, err := openCodexRPC(ctx)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		defer rpc.stop()
		if err := rpc.send(map[string]any{"id": 2, "method": "account/logout"}); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if _, err := rpc.await(2, 20*time.Second); err != nil {
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		if err := clearCodexBedrockProvider(); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
