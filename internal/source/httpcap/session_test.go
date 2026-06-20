package httpcap

import "testing"

// TestSessionsAreDistinct ensures concurrent sessions get unique tokens so
// their traffic never crosses (next.md 3.1).
func TestSessionsAreDistinct(t *testing.T) {
	s := NewSessions()
	a := s.Register("claude_code", "https://api.anthropic.com")
	b := s.Register("codex_cli", "https://api.openai.com/v1")
	if a.Token == b.Token || a.ID == b.ID {
		t.Fatalf("sessions not distinct: %+v %+v", a, b)
	}
	if s.Lookup(a.Token).Client != "claude_code" || s.Lookup(b.Token).Client != "codex_cli" {
		t.Errorf("lookup returned wrong session")
	}
}
