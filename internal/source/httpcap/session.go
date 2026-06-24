// Package httpcap is the HTTP capture adapter: a reverse proxy that records the
// request/response of each coding-agent ↔ LLM exchange and emits a
// event.SourceEvent. It carries no provider semantics — it only records raw
// HTTP facts and applies transport decoding.
package httpcap

import (
	"log"
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
	Provider string `json:"provider"` // wire/normalize id, e.g. "anthropic" / "openai-responses"
	Mode     string `json:"mode"`     // "subscription" | "key" — drives re-attach after a restart
}

// SessionRecord is the NON-SECRET subset of a session that may be persisted so a
// still-running agent can be re-attached to the proxy after a tracelab restart.
// It deliberately carries NO credentials: token is just a routing handle, and the
// real key (key mode) is never written — it must be re-supplied after a restart.
type SessionRecord struct {
	ID       string
	Token    string
	Client   string
	Upstream string
	Provider string
	Mode     string
}

// SessionPersister stores live-session routing facts across restarts. Implementations
// MUST persist only the non-secret SessionRecord fields — never headers or keys.
type SessionPersister interface {
	SaveSession(SessionRecord) error
	DeleteSession(id string) error
	AllSessions() ([]SessionRecord, error)
}

func (s *Session) record() SessionRecord {
	return SessionRecord{ID: s.ID, Token: s.Token, Client: s.Client, Upstream: s.Upstream, Provider: s.Provider, Mode: s.Mode}
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
	// persist stores only the non-secret SessionRecord so a live agent can be
	// re-attached after a restart. nil = no persistence (JSONL/debug mode).
	persist SessionPersister
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

// BindPersister attaches a persister and rehydrates sessions saved by an earlier
// run, so an agent that is still running keeps routing through the proxy after a
// tracelab restart. Only the non-secret routing facts come back; in-memory auth
// (captured headers / injected keys) does NOT — a restored key-mode session reports
// NeedsKey()==true until its key is re-supplied. Safe to call once at startup.
func (s *Sessions) BindPersister(p SessionPersister) {
	if p == nil {
		return
	}
	recs, err := p.AllSessions()
	s.mu.Lock()
	s.persist = p
	for _, rec := range recs {
		sess := &Session{ID: rec.ID, Token: rec.Token, Client: rec.Client, Upstream: rec.Upstream, Provider: rec.Provider, Mode: rec.Mode}
		s.byToken[sess.Token] = sess
		s.byID[sess.ID] = sess
	}
	s.mu.Unlock()
	if err != nil {
		// Non-fatal: persistence is a convenience. Worst case the agent must relaunch.
		log.Printf("tracelab: rehydrate live sessions: %v", err)
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

// Register creates and stores a session. provider is the wire/normalize id and
// mode is "subscription" | "key"; both are persisted (non-secret) so the session
// can be re-attached after a restart. The injected key for a key-mode session is
// NOT part of this — it is held separately in memory via RememberInject.
func (s *Sessions) Register(client, upstream, provider, mode string) *Session {
	sess := &Session{
		ID:       source.RandomID(),
		Token:    source.RandomID(),
		Client:   client,
		Upstream: strings.TrimRight(upstream, "/"),
		Provider: provider,
		Mode:     mode,
	}
	s.mu.Lock()
	s.byToken[sess.Token] = sess
	s.byID[sess.ID] = sess
	p := s.persist
	s.mu.Unlock()
	if p != nil {
		if err := p.SaveSession(sess.record()); err != nil {
			log.Printf("tracelab: persist live session %s: %v", sess.ID, err)
		}
	}
	return sess
}

// NeedsKey reports whether a session is in key mode but has no injected key in
// memory — the state a key-mode session lands in after a restart, where the
// non-secret routing was restored but the real key (never persisted) was not.
// The proxy would otherwise forward the agent's placeholder and get a 401.
func (s *Sessions) NeedsKey(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess := s.byID[id]
	return sess != nil && sess.Mode == "key" && s.inject[id] == nil
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
	p := s.persist
	if p != nil {
		if err := p.DeleteSession(id); err != nil {
			log.Printf("tracelab: forget persisted live session %s: %v", id, err)
		}
	}
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
