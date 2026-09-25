// Package llamaservice manages only an explicitly created local CPU service.
package llamaservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/llama"
	"github.com/cfpperche/picode/internal/store"
)

type Config struct {
	Port    int  `json:"port"`
	Context int  `json:"context"`
	Threads int  `json:"threads"`
	Jinja   bool `json:"jinja"`
}

func (c Config) validate() error {
	if c.Port < 1024 || c.Port > 65535 || c.Context < 512 || c.Context > 131072 || c.Threads < 1 || c.Threads > 4 {
		return errors.New("Use port 1024–65535, context 512–131072 and 1–4 CPU threads.")
	}
	return nil
}

type Release struct {
	Version string            `json:"version"`
	Dir     string            `json:"dir"`
	Files   map[string]string `json:"files"`
}
type Job struct {
	ID      string    `json:"id"`
	Action  string    `json:"action"`
	State   string    `json:"state"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
	Done    int64     `json:"done"`
	Total   int64     `json:"total"`
}
type Document struct {
	Created  bool     `json:"created"`
	Config   Config   `json:"config"`
	Applied  Config   `json:"applied"`
	Current  *Release `json:"current"`
	Previous *Release `json:"previous"`
	Jobs     []Job    `json:"jobs"`
	// Archives are individually pinned installer artifacts, never discovered files.
	Archives  map[string]string     `json:"archives"`
	Models    map[string]OwnedModel `json:"models"`
	Downloads map[string][]string   `json:"downloads"`
	Releases  map[string]Release    `json:"releases,omitempty"`
	Creation  *Creation             `json:"creation,omitempty"`
}
type Snapshot struct {
	Document
	Revision        int      `json:"revision"`
	Running         bool     `json:"running"`
	Supported       bool     `json:"supported"`
	Host            string   `json:"host"`
	URL             string   `json:"url"`
	ModelsDir       string   `json:"modelsDir"`
	RestartRequired bool     `json:"restartRequired"`
	Busy            bool     `json:"busy"`
	Versions        []string `json:"versions"`
	Command         []string `json:"command"`
}
type Request struct {
	Action    string   `json:"action"`
	Version   string   `json:"version"`
	Files     []string `json:"files"`
	Revision  int      `json:"revision"`
	Interrupt bool     `json:"interrupt"`
}
type Preview struct {
	Token       string    `json:"token"`
	Request     Request   `json:"request"`
	Args        []string  `json:"args"`
	Consumers   []string  `json:"consumers"`
	Bytes       int64     `json:"bytes"`
	Expires     time.Time `json:"expires"`
	Fingerprint string    `json:"-"`
}
type Service struct {
	mu           sync.Mutex
	resolveJobs  func() // SetJobResolver; run by Preview before its guard
	st           *store.Store
	root         string
	doc          Document
	persisted    json.RawMessage
	rev          int
	process      *exec.Cmd
	lease        *os.File
	done         chan struct{}
	busy, closed bool
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	previews     map[string]Preview
	// consumers is conservative: configured llama agents and unknown terminal users.
	consumers func() ([]string, error)
}

func New(st *store.Store, dataDir string, consumers func() ([]string, error)) (*Service, error) {
	root, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{st: st, root: filepath.Join(root, "llama-owned"), ctx: ctx, cancel: cancel, previews: map[string]Preview{}, consumers: consumers}
	raw, rev, err := st.LlamaService()
	if err != nil {
		cancel()
		return nil, err
	}
	if err = json.Unmarshal(raw, &s.doc); err != nil {
		cancel()
		return nil, err
	}
	s.rev = rev
	s.persisted = raw
	if !s.doc.Created && s.doc.Creation != nil {
		// Files left by an interrupted setup are removed only with its proof.
		// Unexpected contents remain available for inspection and block retry.
		if s.rollbackCreation(s.doc.Creation) == nil {
			s.doc.Creation = nil
			if err = s.save(); err != nil {
				cancel()
				return nil, err
			}
		}
	}
	s.sweepInterrupted()
	changed := false
	for i := range s.doc.Jobs {
		if s.doc.Jobs[i].State == "running" || s.doc.Jobs[i].State == "queued" {
			s.doc.Jobs[i].State = "interrupted"
			s.doc.Jobs[i].Message = "PiCode restarted. Review the service before retrying."
			changed = true
		}
	}
	// A download's starting file list outlives its job only by the leak that
	// ObserveDownload now closes; drop what earlier runs left (a job that is
	// over, or no longer in the store). A store error keeps the entry.
	for id := range s.doc.Downloads {
		j, e := st.LlamaJob(id)
		if errors.Is(e, sql.ErrNoRows) || (e == nil && !j.Active()) {
			delete(s.doc.Downloads, id)
			changed = true
		}
	}
	if changed {
		err = s.save()
		if err != nil {
			cancel()
			return nil, err
		}
	}
	return s, nil
}
func (s *Service) save() error {
	raw, err := json.Marshal(s.doc)
	if err != nil {
		return err
	}
	rev, err := s.st.SaveLlamaService(raw, s.rev)
	if err == nil {
		s.rev = rev
		s.persisted = raw
	} else {
		var previous Document
		if json.Unmarshal(s.persisted, &previous) == nil {
			s.doc = previous
		}
	}
	return err
}
func (s *Service) snapshot() Snapshot {
	host, _ := os.Hostname()
	versions := []string{"b10809", "b10826"}
	var command []string
	if s.doc.Current != nil {
		command, _ = s.profile(s.doc.Current, s.doc.Config).Args()
	}
	port := s.doc.Config.Port
	if s.process != nil {
		port = s.doc.Applied.Port
	}
	v := Snapshot{Document: s.doc, Revision: s.rev, Running: s.process != nil, Supported: runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"), Host: host, URL: "http://127.0.0.1:" + strconv.Itoa(port), ModelsDir: filepath.Join(s.root, "models"), RestartRequired: s.process != nil && s.doc.Config != s.doc.Applied, Busy: s.busy, Versions: versions, Command: command}
	raw, _ := json.Marshal(v)
	var copy Snapshot
	_ = json.Unmarshal(raw, &copy)
	return copy
}
func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot()
}
func (s *Service) Configure(c Config, revision int) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := c.validate(); err != nil {
		return Snapshot{}, err
	}
	if s.closed || s.busy || revision != s.rev {
		return Snapshot{}, store.ErrLlamaConflict
	}
	if !s.snapshot().Supported {
		return Snapshot{}, errors.New("Local services require Linux or WSL on x64 or ARM64.")
	}
	if !s.doc.Created {
		if err := s.createProfile(c); err != nil {
			return Snapshot{}, err
		}
		return s.snapshot(), nil
	}
	s.doc.Config = c
	if err := s.save(); err != nil {
		return Snapshot{}, err
	}
	return s.snapshot(), nil
}
func (s *Service) profile(r *Release, c Config) llama.ExecutionProfile {
	return llama.ExecutionProfile{Binary: filepath.Join(r.Dir, "llama-server"), BinarySHA256: r.Files["llama-server"], ModelsDir: filepath.Join(s.root, "models"), Host: "127.0.0.1", Port: c.Port, Context: c.Context, Threads: c.Threads, Jinja: c.Jinja, NoAutoLoad: true}
}
func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func token() string { var b [24]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }
func (s *Service) preview(req Request) (Preview, error) {
	p := Preview{Request: req, Args: []string{}, Consumers: []string{}}
	if s.closed || s.busy || req.Revision != s.rev {
		return p, store.ErrLlamaConflict
	}
	if !s.doc.Created {
		return p, errors.New("Create a local service first.")
	}
	if !s.snapshot().Supported {
		return p, errors.New("Local services require Linux or WSL.")
	}
	switch req.Action {
	case "install", "update":
		if _, err := releasePin(req.Version); err != nil {
			return p, err
		}
		if req.Action == "install" && s.doc.Current != nil {
			return p, errors.New("The service is already installed.")
		}
		if req.Action == "update" && s.doc.Current == nil {
			return p, errors.New("Install the service first.")
		}
	case "start", "stop", "restart":
		if s.doc.Current == nil {
			return p, errors.New("Install the service first.")
		}
		if req.Action == "start" && s.process != nil {
			return p, errors.New("The service is already running.")
		}
	case "rollback":
		if s.doc.Previous == nil {
			return p, errors.New("No previous version is available.")
		}
	case "cleanup":
		if s.process != nil {
			return p, errors.New("Stop the local service before cleaning its cache.")
		}
		if len(req.Files) == 0 {
			return p, errors.New("Select at least one eligible cache file.")
		}
		if len(req.Files) > 20 {
			return p, errors.New("Select up to 20 cache files per action.")
		}
	default:
		return p, errors.New("Unknown service action.")
	}
	jobs, err := s.st.LlamaJobs()
	if err != nil {
		return p, err
	}
	for _, j := range jobs {
		if j.Active() && s.ownsEndpoint(j.Endpoint) {
			return p, errors.New("Finish or reconcile model operations before changing the service.")
		}
	}
	if s.process != nil && (req.Action == "stop" || req.Action == "restart" || req.Action == "update" || req.Action == "rollback") {
		if s.consumers != nil {
			p.Consumers, err = s.consumers()
			if err != nil {
				return p, err
			}
		}
		// Any network client can use a loopback server; always review interruption.
		p.Consumers = append(p.Consumers, "Other applications connected to this local server")
	}
	r := s.doc.Current
	if req.Action == "rollback" {
		r = s.doc.Previous
	}
	if r != nil && (req.Action == "start" || req.Action == "restart" || req.Action == "rollback") {
		if err = verifyRelease(r); err != nil {
			return p, err
		}
		p.Args, err = s.profile(r, s.doc.Config).Args()
		if err != nil {
			return p, err
		}
	}
	disk := map[string]string{}
	if req.Action == "cleanup" {
		seen := map[string]bool{}
		for _, name := range req.Files {
			if seen[name] {
				return p, errors.New("Duplicate cache selection.")
			}
			seen[name] = true
			h, size, e := s.cleanupFingerprint(name)
			if e != nil {
				return p, e
			}
			disk[name] = h
			p.Bytes += size
		}
	}
	p.Fingerprint = fingerprint(struct {
		Req       Request
		Consumers []string
		Disk      map[string]string
		Running   bool
	}{req, p.Consumers, disk, s.process != nil})
	return p, nil
}
func (s *Service) Preview(req Request) (Preview, error) {
	s.mu.Lock()
	resolve := s.resolveJobs
	var warm []string
	if req.Action == "cleanup" {
		warm = s.cleanupPaths(req.Files)
	}
	s.mu.Unlock()
	if resolve != nil {
		resolve() // settle jobs on a stopped owned endpoint before the guard reads them
	}
	warmHashes(warm) // the long part, outside the lock (hash_memo.go)
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.preview(req)
	if err != nil {
		return p, err
	}
	for k, v := range s.previews {
		if time.Now().After(v.Expires) {
			delete(s.previews, k)
		}
	}
	if len(s.previews) >= 32 {
		return p, errors.New("Too many pending reviews. Wait five minutes and retry.")
	}
	p.Token = token()
	p.Expires = time.Now().Add(5 * time.Minute)
	s.previews[p.Token] = p
	return p, nil
}
func (s *Service) Execute(key string, interrupt bool) (Job, error) {
	s.mu.Lock()
	if p, ok := s.previews[key]; ok && p.Request.Action == "cleanup" {
		warm := s.cleanupPaths(p.Request.Files)
		s.mu.Unlock()
		warmHashes(warm)
		s.mu.Lock()
	}
	defer s.mu.Unlock()
	for _, j := range s.doc.Jobs {
		if j.ID == key {
			return j, nil
		}
	}
	p, ok := s.previews[key]
	if !ok || time.Now().After(p.Expires) {
		return Job{}, errors.New("This review expired. Preview the action again.")
	}
	next, err := s.preview(p.Request)
	if err != nil {
		return Job{}, err
	}
	if next.Fingerprint != p.Fingerprint {
		return Job{}, errors.New("The service changed. Review the action again.")
	}
	if len(next.Consumers) > 0 && !interrupt {
		return Job{}, errors.New("Confirm interruption of connected applications.")
	}
	j := Job{ID: key, Action: p.Request.Action, State: "queued", Message: "Waiting to start.", At: time.Now().UTC()}
	s.doc.Jobs = append([]Job{j}, s.doc.Jobs...)
	if len(s.doc.Jobs) > 50 {
		s.doc.Jobs = s.doc.Jobs[:50]
	}
	if err = s.save(); err != nil {
		return Job{}, err
	}
	delete(s.previews, key)
	s.busy = true
	s.wg.Add(1)
	go s.run(p.Request)
	return j, nil
}
func (s *Service) run(req Request) {
	defer s.wg.Done()
	s.mu.Lock()
	s.doc.Jobs[0].State = "running"
	s.doc.Jobs[0].Message = "Applying the reviewed action."
	if err := s.save(); err != nil {
		s.busy = false
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	var installed *Release
	var err error
	if req.Action == "install" || req.Action == "update" {
		installed, err = s.install(req.Version)
	}
	if req.Action == "cleanup" {
		// busy already refuses competing changes; hash before the lock so
		// the recheck under it answers from memory.
		s.mu.Lock()
		warm := s.cleanupPaths(req.Files)
		s.mu.Unlock()
		warmHashes(warm)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.rememberRelease(s.doc.Current)
		s.rememberRelease(s.doc.Previous)
		s.rememberRelease(installed)
		switch req.Action {
		case "install":
			s.doc.Current = installed
		case "update", "rollback":
			target := installed
			if req.Action == "rollback" {
				target = s.doc.Previous
			}
			old := s.doc.Current
			oldApplied := s.doc.Applied
			wasRunning := s.process != nil
			if wasRunning {
				err = s.stopProcess()
			}
			if err == nil {
				s.doc.Current = target
				if wasRunning {
					err = s.startProcess()
				}
			}
			if err != nil {
				s.doc.Current = old
				if wasRunning && s.process == nil {
					desired := s.doc.Config
					s.doc.Config = oldApplied
					restore := s.startProcess()
					s.doc.Config = desired
					if restore != nil {
						err = errors.New("Update failed and the previous version could not restart. Review the service.")
					} else {
						err = errors.New("Update failed. The previous version was restored.")
					}
				}
			} else {
				s.doc.Previous = old
			}
		case "start":
			err = s.startProcess()
		case "stop":
			err = s.stopProcess()
		case "restart":
			err = s.stopProcess()
			if err == nil {
				err = s.startProcess()
			}
		case "cleanup":
			// Recheck every selected file before removing any; the worker owns the lock.
			for _, name := range req.Files {
				_, _, e := s.cleanupFingerprint(name)
				if e != nil {
					err = errors.New("Cache changed after review; nothing was removed.")
					break
				}
			}
			if err == nil {
				for _, name := range req.Files {
					if strings.HasPrefix(name, "release-") {
						if err = s.removeRelease(name); err != nil {
							break
						}
						continue
					}
					path, _, targetErr := s.cacheTarget(name)
					if targetErr != nil {
						err = targetErr
						break
					}
					if model, ok := s.doc.Models[name]; ok && model.Snapshot != "" {
						if err = os.Remove(filepath.Join(s.root, model.Snapshot)); err != nil {
							break
						}
					}
					if err = os.Remove(path); err != nil {
						break
					}
					delete(s.doc.Archives, name)
					delete(s.doc.Models, name)
				}
			}
		}
	}
	j := &s.doc.Jobs[0]
	j.State = "succeeded"
	j.Message = "Completed."
	if err != nil {
		j.State = "failed"
		j.Message = err.Error()
	}
	if s.closed {
		j.State = "interrupted"
		j.Message = "PiCode stopped. Review the service before retrying."
	}
	s.busy = false
	if e := s.save(); e != nil {
		s.closed = true
		s.cancel()
		_ = s.stopProcess()
	}
}
func (s *Service) startProcess() error {
	if s.process != nil {
		return errors.New("The service is already running.")
	}
	if err := verifyRelease(s.doc.Current); err != nil {
		return err
	}
	args, err := s.profile(s.doc.Current, s.doc.Config).Args()
	if err != nil {
		return err
	}
	address := "127.0.0.1:" + strconv.Itoa(s.doc.Config.Port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		return errors.New("The selected port is in use. Choose another port.")
	}
	_ = l.Close()
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, append([]string{"--internal-llama-supervisor"}, args...)...)
	cmd.Dir = s.doc.Current.Dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + s.root, "LD_LIBRARY_PATH=" + s.doc.Current.Dir, "LLAMA_CACHE=" + filepath.Join(s.root, "models")}
	reader, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.Stdin = reader
	// The supervisor reports its router's PID (the router's process group)
	// here, so a supervisor killed outright does not strand the router's
	// per-model servers (ADR-0090 amendment 2026-09-25).
	groupR, groupW, err := os.Pipe()
	if err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return err
	}
	cmd.ExtraFiles = []*os.File{groupW}
	cmd.Env = append(cmd.Env, routerFDEnv+"=3")
	if err = cmd.Start(); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		_ = groupR.Close()
		_ = groupW.Close()
		return errors.New("Could not start the verified executable on this host.")
	}
	_ = reader.Close()
	_ = groupW.Close()
	group := readRouterGroup(groupR)
	s.lease = writer
	s.process = cmd
	s.done = make(chan struct{})
	done := s.done
	go func() {
		_ = cmd.Wait()
		// A supervisor that exited normally already killed the group; one
		// killed outright left the router's children behind.
		go clearOrphanedGroup(group)
		close(done)
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.process == cmd {
			_ = s.lease.Close()
			s.lease = nil
			s.process = nil
			_ = s.save()
		}
	}()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			_ = s.lease.Close()
			s.lease = nil
			s.process = nil
			return errors.New("The local server exited before becoming ready.")
		case <-s.ctx.Done():
			_ = s.stopProcess()
			return s.ctx.Err()
		default:
		}
		res, e := client.Get("http://" + address + "/health")
		if e == nil {
			_ = res.Body.Close()
			if res.StatusCode == 200 {
				s.doc.Applied = s.doc.Config
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = s.stopProcess()
	return errors.New("The local server did not become ready in time.")
}
func (s *Service) stopProcess() error {
	if s.process == nil {
		return nil
	}
	if s.lease != nil {
		_ = s.lease.Close()
		s.lease = nil
	}
	select {
	case <-s.done:
		s.process = nil
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("Could not confirm that the owned process stopped.")
	}
}
func (s *Service) Close() {
	s.cancel()
	s.mu.Lock()
	s.closed = true
	_ = s.stopProcess()
	s.mu.Unlock()
	s.wg.Wait()
}

type CacheFile struct {
	Name     string `json:"name"`
	Bytes    int64  `json:"bytes"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
	Label    string `json:"label,omitempty"`
}

func (s *Service) Cache() []CacheFile {
	s.mu.Lock()
	var warm []string
	for name := range s.doc.Archives {
		warm = append(warm, filepath.Join(s.root, "cache", name))
	}
	s.mu.Unlock()
	warmHashes(warm)
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []CacheFile{}
	for name, pin := range s.doc.Archives {
		h, size, err := hashKnown(filepath.Join(s.root, "cache", name))
		eligible := err == nil && h == pin && !s.busy && s.process == nil
		reason := "Verified installer download; the installed copy is retained."
		if !eligible {
			reason = "Stop the service and refresh; changed files are preserved."
		}
		out = append(out, CacheFile{Name: name, Bytes: size, Eligible: eligible, Reason: reason})
	}
	// Untracked model files remain ineligible, even inside the service directory.
	_ = filepath.WalkDir(filepath.Join(s.root, "models"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Join(s.root, "models"), path)
		info, e := d.Info()
		if e == nil {
			name := "models/" + rel
			eligible := false
			reason := "Model reference or ownership is unverified; retained."
			if model, known := s.doc.Models[name]; known {
				_, _, targetErr := s.cacheTarget(name)
				// Inventory is metadata-only for large models. Preview and execution
				// still verify SHA-256 before allowing deletion.
				eligible = targetErr == nil && info.Mode().IsRegular() && info.Size() == model.Size && info.ModTime().UnixNano() == model.Modified && !s.busy && s.process == nil
				if eligible {
					reason = "Downloaded by this service; no configured agent references it."
				} else if targetErr != nil {
					reason = targetErr.Error()
				} else {
					reason = "Stop the service and refresh; changed files are preserved."
				}
			}
			out = append(out, CacheFile{Name: name, Bytes: info.Size(), Eligible: eligible, Reason: reason})
		}
		return nil
	})
	out = append(out, s.installationFiles()...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (s *Service) Diagnostics() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	// An allowlist, not regex masking of arbitrary logs/configuration.
	version := ""
	if s.doc.Current != nil {
		version = s.doc.Current.Version
	}
	jobs := []map[string]string{}
	for _, j := range s.doc.Jobs {
		jobs = append(jobs, map[string]string{"action": j.Action, "state": j.State})
	}
	return map[string]any{"platform": runtime.GOOS + "/" + runtime.GOARCH, "version": version, "running": s.process != nil, "busy": s.busy, "config": s.doc.Config, "jobs": jobs}
}
