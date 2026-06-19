package httpcap

import (
	"slices"
	"strings"
	"testing"
)

// TestLaunchClaudeCode_NonInvasive verifies cc is redirected purely via an env
// var on the child process — the parent env and global config are untouched
// (next.md 1.1).
func TestLaunchClaudeCode_NonInvasive(t *testing.T) {
	sess := &Session{ID: "sess9", Token: "tok123"}
	cmd := LaunchClaudeCode("http://127.0.0.1:8787", sess, "--help")

	want := "ANTHROPIC_BASE_URL=http://127.0.0.1:8787/s/tok123"
	if !slices.Contains(cmd.Env, want) {
		t.Errorf("missing %q in child env", want)
	}
	if cmd.Args[0] != "claude" {
		t.Errorf("unexpected command: %v", cmd.Args)
	}
	// Hooks are injected via --settings (a JSON string), never a global file.
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "--settings") {
		t.Fatalf("expected --settings injection, got %v", cmd.Args)
	}
	if !strings.Contains(joined, "/_hook") || !strings.Contains(joined, "session=sess9") {
		t.Errorf("hook settings missing endpoint/session: %s", joined)
	}
	// Lifecycle events we care about are present.
	for _, ev := range []string{"PreToolUse", "PostToolUse", "SubagentStop", "PreCompact", "PostCompact", "Notification", "PermissionRequest", "TaskCreated"} {
		if !strings.Contains(joined, "event="+ev) {
			t.Errorf("missing hook for %s", ev)
		}
	}
}

// TestLaunchCodex_UsesSingleRunOverrides verifies codex is pointed at the proxy
// with `-c` overrides only — no ~/.codex/config.toml mutation (next.md 1.1).
func TestLaunchCodex_UsesSingleRunOverrides(t *testing.T) {
	sess := &Session{Token: "tokABC"}
	cmd := LaunchCodex("http://127.0.0.1:8787", sess, "exec", "hello")

	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{
		`model_provider="tracelab"`,
		`model_providers.tracelab.base_url="http://127.0.0.1:8787/s/tokABC"`,
		`requires_openai_auth=true`,
		"exec hello",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("codex args missing %q\n got: %s", want, joined)
		}
	}
}

// TestSessionsAreDistinct ensures concurrent sessions get unique tokens so their
// traffic never crosses (next.md 3.1).
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
