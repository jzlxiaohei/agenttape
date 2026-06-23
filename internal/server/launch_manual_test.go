package server

import (
	"strings"
	"testing"
)

// buildManualEnvCommand / manualCommand live in package server, so this is a
// white-box test (package server, not server_test).

func TestBuildManualEnvCommand_CC(t_ *testing.T) {
	cmd := buildManualEnvCommand("cc", "subscription", "http://127.0.0.1:8787", "tok123", "--resume")
	for _, want := range []string{
		"export ANTHROPIC_BASE_URL='http://127.0.0.1:8787/s/tok123'",
		"claude --resume",
	} {
		if !strings.Contains(cmd, want) {
			t_.Errorf("cc cmd missing %q\n got: %s", want, cmd)
		}
	}
	if strings.Contains(cmd, "ANTHROPIC_API_KEY") {
		t_.Errorf("subscription cmd should not mention the key env: %s", cmd)
	}
}

func TestBuildManualEnvCommand_CCKeyMode(t_ *testing.T) {
	cmd := buildManualEnvCommand("cc", "key", "http://h", "tok", "")
	if !strings.Contains(cmd, "export ANTHROPIC_API_KEY='<YOUR_KEY>'") {
		t_.Errorf("key mode should prompt for the key env: %s", cmd)
	}
}

func TestBuildManualEnvCommand_Codex(t_ *testing.T) {
	cmd := buildManualEnvCommand("codex", "subscription", "http://h", "tok", "resume")
	for _, want := range []string{
		`-c 'model_provider="tracelab"'`,
		`-c 'model_providers.tracelab.base_url="http://h/s/tok"'`,
		`-c 'model_providers.tracelab.wire_api="responses"'`,
		"resume", // trailing client arg
	} {
		if !strings.Contains(cmd, want) {
			t_.Errorf("codex cmd missing %q\n got: %s", want, cmd)
		}
	}
}

func TestManualCommand_ForwardsArgs(t_ *testing.T) {
	cmd := manualCommand("/bin/tracelab", "/work", "http://h", "cc", "subscription", "--resume")
	if !strings.Contains(cmd, "launch -kind cc") || !strings.Contains(cmd, " -- --resume") {
		t_.Errorf("full-capture cmd should forward args after --: %s", cmd)
	}
}
