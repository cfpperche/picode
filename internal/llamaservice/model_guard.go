package llamaservice

import (
	"errors"
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
			err = s.save()
		}
	}
	return j, err
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
