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
	// headers holds the latest real request headers per session id, kept ONLY in
	// process memory (never persisted). Replay reuses them to re-send to upstream
	// with the original auth; they vanish when the process exits — credentials are
	// never written to disk.
	headers map[string]http.Header
}

// NewSessions builds an empty registry.
func NewSessions() *Sessions {
	return &Sessions{byToken: map[string]*Session{}, headers: map[string]http.Header{}}
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
	s.mu.Unlock()
	return sess
}

// Lookup returns the session for a token, or nil.
func (s *Sessions) Lookup(token string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byToken[token]
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
