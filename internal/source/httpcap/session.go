// Package httpcap is the HTTP capture adapter: a reverse proxy that records the
// request/response of each coding-agent ↔ LLM exchange and emits a
// event.SourceEvent. It carries no provider semantics — it only records raw
// HTTP facts and applies transport decoding.
package httpcap

import (
	"net/http"
	"strings"
	"sync"

	"tracelab/internal/source"
)

// Session maps a launcher-issued token to the upstream it should be proxied to.
// One running proxy serves many sessions concurrently (cc and codex at once),
// kept apart purely by token — this is what makes the terminal-style multi-tab
// model possible downstream (next.md 3.1).
type Session struct {
	ID       string `json:"id"`
	Token    string `json:"token"`
	Client   string `json:"client"`   // e.g. "claude_code", "codex_cli"
	Upstream string `json:"upstream"` // e.g. "https://api.anthropic.com"
}

// SessionBaseURL is the per-session proxy entrypoint a client should be pointed
// at: <proxyBase>/s/<token>.
func SessionBaseURL(proxyBase string, sess *Session) string {
	return strings.TrimRight(proxyBase, "/") + "/s/" + sess.Token
}

// Sessions is a concurrency-safe registry of active capture sessions.
type Sessions struct {
	mu      sync.RWMutex
	byToken map[string]*Session
	byID    map[string]*Session
	// headers holds the latest real request headers per session id, captured from
	// proxied traffic. inject holds auth headers the proxy should ADD on forward
	// (API-key launch mode, so the agent never sees the real key). Both live ONLY
	// in process memory — credentials are never written to disk and die on exit.
	headers map[string]http.Header
	inject  map[string]http.Header
}

// NewSessions builds an empty registry.
func NewSessions() *Sessions {
	return &Sessions{
		byToken: map[string]*Session{},
		byID:    map[string]*Session{},
		headers: map[string]http.Header{},
		inject:  map[string]http.Header{},
	}
}

// RememberHeaders stores (in memory only) the latest request headers for a
// session so replay can reuse the original auth.
func (s *Sessions) RememberHeaders(sessionID string, h http.Header) {
	s.mu.Lock()
	s.headers[sessionID] = h.Clone()
	s.mu.Unlock()
}

// Headers returns the in-memory request headers for a session, or nil if none
// were captured in this process (e.g. the session predates a restart).
func (s *Sessions) Headers(sessionID string) http.Header {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.headers[sessionID]
}

// Register creates and stores a session for the given client and upstream.
func (s *Sessions) Register(client, upstream string) *Session {
	sess := &Session{
		ID:       source.RandomID(),
		Token:    source.RandomID(),
		Client:   client,
		Upstream: strings.TrimRight(upstream, "/"),
	}
	s.mu.Lock()
	s.byToken[sess.Token] = sess
	s.byID[sess.ID] = sess
	s.mu.Unlock()
	return sess
}

// Lookup returns the session for a token, or nil.
func (s *Sessions) Lookup(token string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byToken[token]
}

// Get returns the session for an id, or nil.
func (s *Sessions) Get(sessionID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID[sessionID]
}

// RememberInject stores auth headers the proxy will inject on forward for this
// session (API-key launch mode). Memory only.
func (s *Sessions) RememberInject(sessionID string, h http.Header) {
	s.mu.Lock()
	s.inject[sessionID] = h.Clone()
	s.mu.Unlock()
}

// InjectFor returns the proxy-inject auth headers for a session, or nil.
func (s *Sessions) InjectFor(sessionID string) http.Header {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inject[sessionID]
}

// AuthFor returns the best auth headers for replaying a session's requests:
// the proxy-injected key if set (API-key mode), else the captured request
// headers (subscription/captured mode). nil if neither is in memory.
func (s *Sessions) AuthFor(sessionID string) http.Header {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if h := s.inject[sessionID]; h != nil {
		return h
	}
	return s.headers[sessionID]
}

// Remove forgets a live session: it drops the token→upstream mapping and any
// in-memory auth/headers for it. This does NOT kill the coding-agent process
// (tracelab doesn't own it) — it only revokes the proxy session, so the agent's
// next proxied request fails and replay can no longer use it. Returns false if
// no such session was registered in this process.
func (s *Sessions) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.byID[id]
	if sess == nil {
		return false
	}
	delete(s.byToken, sess.Token)
	delete(s.byID, id)
	delete(s.headers, id)
	delete(s.inject, id)
	return true
}

// List returns a snapshot of all sessions.
func (s *Sessions) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0, len(s.byToken))
	for _, v := range s.byToken {
		out = append(out, v)
	}
	return out
}
