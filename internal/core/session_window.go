package core

import "sync"

// sessionExchange is one user text + final assistant reply for EP-014 sliding session memory.
type sessionExchange struct {
	user, assistant string
}

// sessionKeyBuf holds exchanges for one session key with its own mutex (REQ-14.005).
type sessionKeyBuf struct {
	mu        sync.Mutex
	exchanges []sessionExchange
}

// sessionWindowStore is an in-memory sliding window per session key (REQ-14.004, REQ-14.005).
type sessionWindowStore struct {
	mu   sync.Mutex
	keys map[string]*sessionKeyBuf
}

func newSessionWindowStore() *sessionWindowStore {
	return &sessionWindowStore{keys: make(map[string]*sessionKeyBuf)}
}

// snapshot returns a copy of stored exchanges oldest-to-newest for key, or nil if none.
func (s *sessionWindowStore) snapshot(key string) []sessionExchange {
	if s == nil || key == "" {
		return nil
	}
	s.mu.Lock()
	kb := s.keys[key]
	s.mu.Unlock()
	if kb == nil {
		return nil
	}
	kb.mu.Lock()
	defer kb.mu.Unlock()
	out := make([]sessionExchange, len(kb.exchanges))
	copy(out, kb.exchanges)
	return out
}

// appendExchange adds one exchange and drops oldest entries when len exceeds max.
func (s *sessionWindowStore) appendExchange(key, user, assistant string, max int) {
	if s == nil || key == "" || max < 1 {
		return
	}
	s.mu.Lock()
	kb := s.keys[key]
	if kb == nil {
		kb = &sessionKeyBuf{}
		s.keys[key] = kb
	}
	s.mu.Unlock()

	kb.mu.Lock()
	defer kb.mu.Unlock()
	kb.exchanges = append(kb.exchanges, sessionExchange{user: user, assistant: assistant})
	for len(kb.exchanges) > max {
		kb.exchanges = kb.exchanges[1:]
	}
}
