package llamaservice

import (
	"errors"
	"log"
	"net"
	"net/url"
	"strconv"

	"github.com/cfpperche/picode/internal/store"
)

// WithModelOperation shares the lifecycle lock through durable reservation.
// A model request cannot slip between preview revalidation and job acceptance.
func (s *Service) WithModelOperation(endpoint string, start func() (store.LlamaJob, error)) (store.LlamaJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc.Created && s.ownsEndpoint(endpoint) && s.busy {
		return store.LlamaJob{}, errors.New("Wait for the local service action to finish.")
	}
	var baseline []string
	var err error
	owned := s.doc.Created && s.process != nil && s.isAppliedEndpoint(endpoint)
	if owned {
		baseline, err = s.modelFiles()
		if err != nil {
			return store.LlamaJob{}, err
		}
	}
	j, err := start()
	if err == nil && owned && j.Operation == "download" {
		if s.doc.Downloads == nil {
			s.doc.Downloads = map[string][]string{}
		}
		if _, exists := s.doc.Downloads[j.ID]; !exists {
			s.doc.Downloads[j.ID] = baseline
			// The job is created and running: a failed save of the ownership
			// baseline must not report the operation as failed (it answered
			// 500 while the download went on). The file just stays unowned.
			if saveErr := s.save(); saveErr != nil {
				log.Printf("llama service: could not record the download baseline for %s: %v", j.ID, saveErr)
			}
		}
	}
	return j, err
}

// Stopped reports whether endpoint is PiCode's own service and it is not
// running — no process, and no action (a start) under way. Model jobs there
// cannot be in flight (ADR-0083/0090 amendments, 2026-09-23).
func (s *Service) Stopped(endpoint string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doc.Created && s.process == nil && !s.busy && s.ownsEndpoint(endpoint)
}

// SetJobResolver installs what Preview runs first to settle model jobs on a
// stopped owned endpoint (llamajob.InterruptStopped). It runs outside s.mu:
// the resolver asks Stopped, which takes it.
func (s *Service) SetJobResolver(fn func()) {
	s.mu.Lock()
	s.resolveJobs = fn
	s.mu.Unlock()
}

// Ownership is stricter than interruption guards: aliases and unsaved ports
// may name another server, so only our exact advertised endpoint earns a ledger.
func (s *Service) isAppliedEndpoint(endpoint string) bool {
	return endpoint == "http://127.0.0.1:"+strconv.Itoa(s.doc.Applied.Port)
}

func (s *Service) ownsEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return false
	}
	return u.Port() == strconv.Itoa(s.doc.Config.Port) || u.Port() == strconv.Itoa(s.doc.Applied.Port)
}
