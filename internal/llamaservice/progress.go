package llamaservice

import "time"

type downloadProgress struct {
	service     *Service
	done, total int64
	last        time.Time
}

func (p *downloadProgress) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if time.Since(p.last) > 250*time.Millisecond || (p.total > 0 && p.done == p.total) {
		p.last = time.Now()
		s := p.service
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.doc.Jobs) > 0 {
			j := &s.doc.Jobs[0]
			j.Done = p.done
			j.Total = p.total
			j.Message = "Downloading verified release…"
			if err := s.save(); err != nil {
				return 0, err
			}
		}
	}
	return len(b), nil
}
