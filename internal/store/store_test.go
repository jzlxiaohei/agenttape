package store_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"agenttape/internal/event"
	"agenttape/internal/normalize/providers"
	"agenttape/internal/sink"
	"agenttape/internal/store"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return string(b)
}

func httpRecord(t *testing.T, sessionID string) sink.Record {
	id := "evt-http"
	ev := event.New(id, event.KindHTTPExchange, event.SourceRef{
		Kind: event.SourceCapture, Adapter: "httpcap", Client: "codex_cli",
	})
	ev.Correlation.SessionID = sessionID
	ev.Timing = event.Timing{StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	req := event.NewRawArtifact(id+":req", event.RoleRequestBody, []byte(fixture(t, "openai_responses.request.json")))
	resp := event.NewRawArtifact(id+":resp", event.RoleResponseBody, []byte(fixture(t, "openai_responses.response.sse")))
	ev.RawArtifacts = []event.RawArtifact{req, resp}
	ev.Capture = &event.CaptureEvent{
		Protocol: "http", Method: "POST",
		Target:   "https://chatgpt.com/backend-api/codex/responses",
		Request:  event.HTTPMessage{BodyArtifactID: req.ID},
		Response: event.HTTPMessage{StatusCode: 200, BodyArtifactID: resp.ID},
	}
	env, err := providers.Registry().Normalize(&ev)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	return sink.Record{Event: &ev, Normalized: env}
}

func TestStoreWriteEndToEnd(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	// HTTP completion (openai-responses → tool_call + reasoning signals).
	if err := st.Write(httpRecord(t, "sess-http")); err != nil {
		t.Fatalf("write http: %v", err)
	}
	// Hook event from a different session → subagent tag, decoupled pipeline.
	hookEv := event.NewHookEvent("evt-hook", "claude_code", "SubagentStop", []byte(`{"agent":"explore"}`))
	hookEv.Correlation.SessionID = "sess-hook"
	if err := st.Write(sink.Record{Event: &hookEv}); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	// Sessions + events persisted, kept distinct.
	sessions, err := st.ListSessions()
	if err != nil || len(sessions) != 2 {
		t.Fatalf("ListSessions = %v (%d), want 2", err, len(sessions))
	}

	// http_exchanges row carries denormalized provider/model/tokens.
	var provider, model string
	var total int64
	var normJSON string
	row := st.DB().QueryRow(`SELECT provider, model, total_tokens, normalized_json FROM http_exchanges WHERE event_id='evt-http'`)
	if err := row.Scan(&provider, &model, &total, &normJSON); err != nil {
		t.Fatalf("scan http_exchanges: %v", err)
	}
	if provider != "openai-responses" || model != "gpt-5.5" || total == 0 {
		t.Errorf("http row = %s/%s total=%d", provider, model, total)
	}
	// block.Raw stripped from persisted JSON.
	if strings.Contains(normJSON, `"raw":`) {
		t.Errorf("normalized_json still contains block.Raw")
	}

	// Raw bytes retained on disk (user-facing feature).
	for _, suffix := range []string{"request.json", "response.txt"} {
		p := filepath.Join(dir, "raw", "sess-http", "evt-http."+suffix)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("raw file missing: %s", p)
		}
	}

	// Tags persisted: structural (tool_call/reasoning) + hook-derived subagent.
	tags, err := st.TagCounts()
	if err != nil {
		t.Fatalf("tag counts: %v", err)
	}
	for _, want := range []string{"tool_call", "reasoning", "subagent"} {
		if tags[want] == 0 {
			t.Errorf("missing tag %q (got %v)", want, tags)
		}
	}

	// FTS works: a tool name from the request is searchable.
	ids, err := st.Search("exec_command")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(ids) == 0 || ids[0] != "evt-http" {
		t.Errorf("FTS search returned %v, want evt-http", ids)
	}

	// Sections persisted for the http event.
	var nSections int
	st.DB().QueryRow(`SELECT COUNT(*) FROM sections WHERE event_id='evt-http'`).Scan(&nSections)
	if nSections != 3 {
		t.Errorf("sections = %d, want 3", nSections)
	}

	// ListEvents returns the http event for its session.
	events, err := st.ListEvents("sess-http")
	if err != nil || len(events) != 1 {
		t.Fatalf("ListEvents = %v (%d), want 1", err, len(events))
	}
	if events[0].ID != "evt-http" || events[0].Provider != "openai-responses" || !events[0].IsCompletion {
		t.Errorf("event summary wrong: %+v", events[0])
	}

	// GetEvent assembles detail with normalized payload, tags, raw files.
	detail, err := st.GetEvent("evt-http")
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if detail.Provider != "openai-responses" || detail.Model != "gpt-5.5" || !detail.IsCompletion {
		t.Errorf("detail meta wrong: %+v", detail)
	}
	if len(detail.Normalized) == 0 {
		t.Errorf("detail missing normalized payload")
	}
	if len(detail.Tags) < 2 || len(detail.RawFiles) != 2 {
		t.Errorf("detail tags=%d rawfiles=%d", len(detail.Tags), len(detail.RawFiles))
	}

	// RawFilePath resolves to a real file on disk.
	p, err := st.RawFilePath("evt-http", "request_body")
	if err != nil {
		t.Fatalf("RawFilePath: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("raw file path not on disk: %s", p)
	}

	// Missing event → ErrNoRows.
	if _, err := st.GetEvent("nope"); err != store.ErrNoRows {
		t.Errorf("GetEvent(missing) = %v, want ErrNoRows", err)
	}

	// SearchEvents: FTS hit with snippet.
	res, err := st.SearchEvents(store.SearchFilters{Query: "exec_command"})
	if err != nil || len(res) == 0 {
		t.Fatalf("SearchEvents = %v (%d), want >=1", err, len(res))
	}
	if res[0].EventID != "evt-http" || res[0].Client != "codex_cli" || res[0].Snippet == "" {
		t.Errorf("search result wrong: %+v", res[0])
	}
	// Filter that excludes the only match → empty.
	if r, _ := st.SearchEvents(store.SearchFilters{Query: "exec_command", Client: "claude_code"}); len(r) != 0 {
		t.Errorf("client filter should exclude, got %d", len(r))
	}
	// Browse (no query) with provider filter still finds it.
	if r, _ := st.SearchEvents(store.SearchFilters{Provider: "openai-responses"}); len(r) == 0 {
		t.Errorf("provider browse should find the event")
	}

	// Facets expose distinct providers/clients/tags.
	providers, clients, tags2, err := st.Facets()
	if err != nil {
		t.Fatalf("Facets: %v", err)
	}
	if !slices.Contains(providers, "openai-responses") || !slices.Contains(clients, "codex_cli") || !slices.Contains(tags2, "tool_call") {
		t.Errorf("facets missing values: %v %v %v", providers, clients, tags2)
	}
}
