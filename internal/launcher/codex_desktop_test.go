package launcher

import (
	"strings"
	"testing"

	"github.com/jzlxiaohei/agenttape/internal/source/hook"
	"github.com/jzlxiaohei/agenttape/internal/source/httpcap"
)

func TestMergeCodexDesktopConfig(t *testing.T) {
	sess := &httpcap.Session{ID: "sess1", Token: "tok1"}
	original := `model_provider = "openai"
model = "gpt-5.5"

[profiles.work]
model_provider = "azure"

[mcp_servers.fs]
command = "fs-server"
`
	got := MergeCodexDesktopConfig(original, "http://127.0.0.1:8787", sess, hook.DefaultCodexEvents())

	// Our top-level provider must come before any table header, and the original's
	// top-level model_provider must be gone (else duplicate-key TOML error).
	idxProvider := strings.Index(got, `model_provider = "agenttape"`)
	idxFirstTable := strings.Index(got, "\n[")
	if idxProvider < 0 || idxFirstTable < 0 || idxProvider > idxFirstTable {
		t.Fatalf("top-level model_provider not before first table:\n%s", got)
	}
	if strings.Contains(got, `model_provider = "openai"`) {
		t.Error("original top-level model_provider was not stripped")
	}
	// Profile-scoped model_provider must survive.
	if !strings.Contains(got, `model_provider = "azure"`) {
		t.Error("profile-scoped model_provider must be preserved")
	}
	// User content and our provider/hook tables must all be present.
	for _, want := range []string{
		CodexMarker,
		`model = "gpt-5.5"`,
		"[mcp_servers.fs]",
		"[model_providers.agenttape]",
		`base_url = "http://127.0.0.1:8787/s/tok1"`,
		"[[hooks.SessionStart]]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("merged config missing %q", want)
		}
	}
	if !HasCodexMarker(got) {
		t.Error("HasCodexMarker should detect the injection")
	}
}

func TestMergeCodexDesktopConfigNoHooks(t *testing.T) {
	sess := &httpcap.Session{ID: "s", Token: "t"}
	got := MergeCodexDesktopConfig("", "http://h", sess, nil)
	if strings.Contains(got, "[[hooks.") {
		t.Error("hooks must be omitted when no events are given")
	}
	if !strings.Contains(got, "[model_providers.agenttape]") {
		t.Error("provider routing must always be present")
	}
}
