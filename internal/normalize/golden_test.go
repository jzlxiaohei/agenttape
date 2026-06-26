package normalize_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jzlxiaohei/agenttape/internal/event"
	"github.com/jzlxiaohei/agenttape/internal/normalize"
	"github.com/jzlxiaohei/agenttape/internal/normalize/providers"
)

// buildCaptureEvent constructs a SourceEvent from raw request/response bodies,
// mirroring what the httpcap adapter produces. Tests use real captured bodies
// from testdata so the normalizers are exercised against actual traffic.
func buildCaptureEvent(id, target, reqBody, respBody string) *event.SourceEvent {
	reqArt := event.NewRawArtifact(id+":req", event.RoleRequestBody, []byte(reqBody))
	respArt := event.NewRawArtifact(id+":resp", event.RoleResponseBody, []byte(respBody))
	ev := event.New(id, event.KindHTTPExchange, event.SourceRef{Kind: event.SourceCapture, Adapter: "httpcap"})
	ev.Capture = &event.CaptureEvent{
		Protocol: "http",
		Method:   "POST",
		Target:   target,
		Request:  event.HTTPMessage{BodyArtifactID: reqArt.ID},
		Response: event.HTTPMessage{BodyArtifactID: respArt.ID, StatusCode: 200},
	}
	ev.RawArtifacts = []event.RawArtifact{reqArt, respArt}
	return &ev
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func hasSignal(env *normalize.NormalizedEnvelope, tag string) bool {
	for _, s := range env.Signals {
		if s.Tag == tag {
			return true
		}
	}
	return false
}

func sectionsOK(t *testing.T, env *normalize.NormalizedEnvelope) {
	t.Helper()
	want := map[string]bool{"system": false, "tools": false, "messages": false}
	for _, s := range env.Request.Sections {
		want[s.Name] = true
		if s.Name == "messages" && s.ApproxTokens <= 0 {
			t.Errorf("messages section has non-positive approx tokens: %d", s.ApproxTokens)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing section stat %q", name)
		}
	}
}

func TestNormalizeAnthropicMessages(t *testing.T) {
	ev := buildCaptureEvent("a1", "https://api.anthropic.com/v1/messages?beta=true",
		readFixture(t, "anthropic_messages.request.json"),
		readFixture(t, "anthropic_messages.response.sse"))

	env, err := providers.Registry().Normalize(ev)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if env.Provider.Name != "anthropic" {
		t.Fatalf("provider = %q, want anthropic", env.Provider.Name)
	}
	if env.Provider.Model != "claude-opus-4-8" {
		t.Errorf("model = %q, want claude-opus-4-8", env.Provider.Model)
	}
	if env.Request == nil || len(env.Request.Messages) == 0 || len(env.Request.System) == 0 {
		t.Fatalf("request not parsed structurally: %+v", env.Request)
	}
	if len(env.Request.Tools) != 29 {
		t.Errorf("tools = %d, want 29", len(env.Request.Tools))
	}
	sectionsOK(t, env)
	if env.Response == nil || env.Response.FinalText == "" {
		t.Fatalf("response final text empty")
	}
	if env.Response.Usage == nil || env.Response.Usage.InputTokens == 0 || env.Response.Usage.OutputTokens == 0 {
		t.Errorf("usage not reconstructed: %+v", env.Response.Usage)
	}
}

func TestNormalizeOpenAIResponses(t *testing.T) {
	ev := buildCaptureEvent("o1", "https://chatgpt.com/backend-api/codex/responses",
		readFixture(t, "openai_responses.request.json"),
		readFixture(t, "openai_responses.response.sse"))

	env, err := providers.Registry().Normalize(ev)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if env.Provider.Name != "openai-responses" {
		t.Fatalf("provider = %q, want openai-responses", env.Provider.Name)
	}
	if env.Provider.Model != "gpt-5.5" {
		t.Errorf("model = %q, want gpt-5.5", env.Provider.Model)
	}
	if env.Request == nil || len(env.Request.Messages) == 0 || len(env.Request.System) == 0 {
		t.Fatalf("request not parsed structurally")
	}
	if len(env.Request.Tools) != 16 {
		t.Errorf("tools = %d, want 16", len(env.Request.Tools))
	}
	sectionsOK(t, env)
	// This conversation contains function_call and reasoning items — structural
	// signals must be derived (facts, not suspected).
	if !hasSignal(env, "tool_call") {
		t.Errorf("expected structural tool_call signal")
	}
	if !hasSignal(env, "reasoning") {
		t.Errorf("expected structural reasoning signal")
	}
	if env.Response == nil || len(env.Response.Output) == 0 {
		t.Fatalf("response output empty")
	}
	if env.Response.Usage == nil || env.Response.Usage.TotalTokens == 0 {
		t.Errorf("usage not parsed: %+v", env.Response.Usage)
	}
}
