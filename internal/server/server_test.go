package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"tracelab/internal/normalize/providers"
	"tracelab/internal/server"
	"tracelab/internal/sink"
	"tracelab/internal/store"
)

type memSink struct {
	mu   sync.Mutex
	recs []sink.Record
}

func (m *memSink) Write(r sink.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recs = append(m.recs, r)
	return nil
}
func (m *memSink) Close() error { return nil }
func (m *memSink) records() []sink.Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]sink.Record(nil), m.recs...)
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return string(b)
}

// fakeUpstream serves the captured response fixtures for both wire formats.
func fakeUpstream(t *testing.T) *httptest.Server {
	anthropic := fixture(t, "anthropic_messages.response.sse")
	responses := fixture(t, "openai_responses.response.sse")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/v1/messages"):
			io.WriteString(w, anthropic)
		case strings.Contains(r.URL.Path, "/responses"):
			io.WriteString(w, responses)
		default:
			http.NotFound(w, r)
		}
	}))
}

func registerSession(t *testing.T, base, client, upstream string) string {
	t.Helper()
	resp, err := http.Post(base+"/_register", "application/json",
		strings.NewReader(`{"client":"`+client+`","upstream":"`+upstream+`"}`))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	// crude token extraction to avoid a struct just for the test
	const key = `"token":"`
	i := strings.Index(string(b), key)
	if i < 0 {
		t.Fatalf("no token in %s", b)
	}
	rest := string(b)[i+len(key):]
	return rest[:strings.IndexByte(rest, '"')]
}

type registeredSession struct {
	SessionID string `json:"session_id"`
	Token     string `json:"token"`
}

func registerSessionInfo(t *testing.T, base, client, upstream string) registeredSession {
	t.Helper()
	resp, err := http.Post(base+"/_register", "application/json",
		strings.NewReader(`{"client":"`+client+`","upstream":"`+upstream+`"}`))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	var out registeredSession
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if out.SessionID == "" || out.Token == "" {
		t.Fatalf("bad register response: %+v", out)
	}
	return out
}

func proxyPOST(t *testing.T, base, token, path, body string) string {
	t.Helper()
	resp, err := http.Post(base+"/s/"+token+"/"+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("proxy POST: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

// TestEndToEnd_ConcurrentSessionsStayIsolated drives both a cc (anthropic) and a
// codex (openai-responses) session through one running server and asserts each
// exchange is captured, normalized by the right provider, and kept on its own
// session — the foundation for terminal-style multi-session (next.md 3.1).
func TestEndToEnd_ConcurrentSessionsStayIsolated(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	ms := &memSink{}
	srv := server.New(ms, providers.Registry())
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ccToken := registerSession(t, ts.URL, "claude_code", up.URL)
	codexToken := registerSession(t, ts.URL, "codex_cli", up.URL)

	ccResp := proxyPOST(t, ts.URL, ccToken, "v1/messages", fixture(t, "anthropic_messages.request.json"))
	if !strings.Contains(ccResp, "message_start") {
		t.Errorf("cc client did not receive upstream stream")
	}
	codexResp := proxyPOST(t, ts.URL, codexToken, "responses", fixture(t, "openai_responses.request.json"))
	if !strings.Contains(codexResp, "response.completed") {
		t.Errorf("codex client did not receive upstream stream")
	}

	recs := ms.records()
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}

	byProvider := map[string]sink.Record{}
	sessions := map[string]bool{}
	for _, r := range recs {
		if r.Normalized == nil {
			t.Fatalf("record not normalized: %s", r.NormalizeError)
		}
		byProvider[r.Normalized.Provider.Name] = r
		sessions[r.Event.Correlation.SessionID] = true
	}
	if _, ok := byProvider["anthropic"]; !ok {
		t.Errorf("missing anthropic record")
	}
	if _, ok := byProvider["openai-responses"]; !ok {
		t.Errorf("missing openai-responses record")
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 distinct sessions, got %d", len(sessions))
	}

	// Sensitive request headers must be redacted on capture.
	for _, r := range recs {
		if vals, ok := r.Event.Capture.Request.Headers["Authorization"]; ok {
			if len(vals) != 1 || vals[0] != "[redacted]" {
				t.Errorf("authorization not redacted: %v", vals)
			}
		}
	}
}

func TestCaseRunUsesLaunchSessionProxy(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ms := &memSink{}
	srv := server.New(ms, providers.Registry())
	srv.EnableAPI(st)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	sess := registerSessionInfo(t, ts.URL, "codex_cli", up.URL)
	if resp := proxyPOST(t, ts.URL, sess.Token, "responses", fixture(t, "openai_responses.request.json")); !strings.Contains(resp, "response.completed") {
		t.Fatalf("warmup proxy response missing completion: %s", resp)
	}

	body, _ := json.Marshal(map[string]string{"session_id": sess.SessionID})
	resp, err := http.Post(ts.URL+"/api/cases/seed:codex-pure-text/run", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("run case: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("run case status=%d body=%s", resp.StatusCode, b)
	}
	if !strings.Contains(string(b), `"status":200`) || !strings.Contains(string(b), `"openai-responses"`) {
		t.Fatalf("run case did not normalize fake upstream response: %s", b)
	}

	recs := ms.records()
	if len(recs) != 2 {
		t.Fatalf("expected warmup + case run captures, got %d", len(recs))
	}
	last := recs[len(recs)-1]
	if last.Event.Correlation.SessionID != sess.SessionID {
		t.Fatalf("case run captured on session %q, want %q", last.Event.Correlation.SessionID, sess.SessionID)
	}
	if last.Event.Capture == nil || last.Event.Capture.Target != up.URL+"/responses" {
		t.Fatalf("case run target = %#v, want %s/responses", last.Event.Capture, up.URL)
	}
}
