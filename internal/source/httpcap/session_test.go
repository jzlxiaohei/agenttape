package httpcap

import "testing"

// memPersister is an in-memory SessionPersister for tests. It records only the
// non-secret SessionRecord, mirroring the real store contract.
type memPersister struct{ rows map[string]SessionRecord }

func newMemPersister() *memPersister { return &memPersister{rows: map[string]SessionRecord{}} }

func (m *memPersister) SaveSession(r SessionRecord) error { m.rows[r.ID] = r; return nil }
func (m *memPersister) DeleteSession(id string) error     { delete(m.rows, id); return nil }
func (m *memPersister) AllSessions() ([]SessionRecord, error) {
	out := make([]SessionRecord, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, r)
	}
	return out, nil
}

// TestSessionsAreDistinct ensures concurrent sessions get unique tokens so
// their traffic never crosses (next.md 3.1).
func TestSessionsAreDistinct(t *testing.T) {
	s := NewSessions()
	a := s.Register("claude_code", "https://api.anthropic.com", "anthropic", "subscription")
	b := s.Register("codex_cli", "https://api.openai.com/v1", "openai-responses", "key")
	if a.Token == b.Token || a.ID == b.ID {
		t.Fatalf("sessions not distinct: %+v %+v", a, b)
	}
	if s.Lookup(a.Token).Client != "claude_code" || s.Lookup(b.Token).Client != "codex_cli" {
		t.Errorf("lookup returned wrong session")
	}
}

// TestReattachAfterRestart simulates a agenttape restart: a new registry bound to the
// same persister must restore the token route (so a live agent keeps working), and a
// key-mode session must report NeedsKey until its key is re-supplied — proving no
// secret was persisted.
func TestReattachAfterRestart(t *testing.T) {
	p := newMemPersister()

	// First process: a subscription cc session and a key-mode codex session.
	s1 := NewSessions()
	s1.BindPersister(p)
	sub := s1.Register("claude_code", "https://api.anthropic.com", "anthropic", "subscription")
	key := s1.Register("codex_cli", "https://api.openai.com/v1", "openai-responses", "key")
	s1.RememberInject(key.ID, map[string][]string{"Authorization": {"Bearer real-secret"}})

	// Restart: a fresh registry rehydrates from the same persister.
	s2 := NewSessions()
	s2.BindPersister(p)

	if got := s2.Lookup(sub.Token); got == nil || got.Client != "claude_code" {
		t.Fatal("subscription session token did not survive restart")
	}
	if got := s2.Lookup(key.Token); got == nil || got.Mode != "key" {
		t.Fatal("key session token did not survive restart")
	}
	// The real key was NOT persisted, so the key-mode session needs it re-supplied.
	if !s2.NeedsKey(key.ID) {
		t.Error("restored key-mode session should report NeedsKey (no secret persisted)")
	}
	if s2.InjectFor(key.ID) != nil {
		t.Error("no injected key should survive a restart")
	}
	// Subscription session never needs a key.
	if s2.NeedsKey(sub.ID) {
		t.Error("subscription session must not report NeedsKey")
	}
	// Re-supplying the key clears NeedsKey.
	s2.RememberInject(key.ID, map[string][]string{"Authorization": {"Bearer again"}})
	if s2.NeedsKey(key.ID) {
		t.Error("NeedsKey should clear after the key is re-supplied")
	}

	// Removing forgets it from persistence too.
	s2.Remove(sub.ID)
	if _, err := p.AllSessions(); err != nil {
		t.Fatal(err)
	}
	s3 := NewSessions()
	s3.BindPersister(p)
	if s3.Lookup(sub.Token) != nil {
		t.Error("removed session should not come back on the next restart")
	}
}
