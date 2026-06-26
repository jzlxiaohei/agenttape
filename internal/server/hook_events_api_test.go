package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jzlxiaohei/agenttape/internal/normalize/providers"
	"github.com/jzlxiaohei/agenttape/internal/server"
	"github.com/jzlxiaohei/agenttape/internal/store"
)

func hookEventsServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	srv := server.New(&memSink{}, providers.Registry())
	srv.EnableAPI(st)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func listHookEvents(t *testing.T, base string) []store.HookEventDef {
	t.Helper()
	resp, err := http.Get(base + "/api/hook-events")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	var defs []store.HookEventDef
	if err := json.NewDecoder(resp.Body).Decode(&defs); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return defs
}

func sendHook(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func TestHookEventsAPI(t *testing.T) {
	ts := hookEventsServer(t)

	// Seeded on store open: defaults present, all enabled, source=seed.
	defs := listHookEvents(t, ts.URL)
	if len(defs) == 0 {
		t.Fatal("expected seeded hook events")
	}
	for _, d := range defs {
		if d.Source != "seed" || !d.Enabled {
			t.Fatalf("seed default unexpected: %+v", d)
		}
	}

	// Add a user event.
	if resp := sendHook(t, "POST", ts.URL+"/api/hook-events",
		map[string]string{"client": "claude_code", "event": "BrandNewEvent"}); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("add status = %d", resp.StatusCode)
	}
	if findHookSource(listHookEvents(t, ts.URL), "claude_code", "BrandNewEvent") != "user" {
		t.Fatal("added event should be source=user")
	}

	// Disable a seed default (mute without delete).
	if resp := sendHook(t, "PATCH", ts.URL+"/api/hook-events",
		map[string]any{"client": "claude_code", "event": "SessionStart", "enabled": false}); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("patch status = %d", resp.StatusCode)
	}
	if hookEnabled(listHookEvents(t, ts.URL), "claude_code", "SessionStart") {
		t.Fatal("SessionStart should be disabled")
	}

	// Deleting a seed default is refused (disable instead).
	if resp := sendHook(t, "DELETE", ts.URL+"/api/hook-events",
		map[string]string{"client": "claude_code", "event": "SessionStart"}); resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete seed status = %d, want 403; body=%s", resp.StatusCode, b)
	}

	// Deleting a user event works.
	if resp := sendHook(t, "DELETE", ts.URL+"/api/hook-events",
		map[string]string{"client": "claude_code", "event": "BrandNewEvent"}); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete user status = %d", resp.StatusCode)
	}
	if findHookSource(listHookEvents(t, ts.URL), "claude_code", "BrandNewEvent") != "" {
		t.Fatal("deleted user event still present")
	}

	// Unknown client is rejected.
	if resp := sendHook(t, "POST", ts.URL+"/api/hook-events",
		map[string]string{"client": "bogus", "event": "X"}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown client status = %d, want 400", resp.StatusCode)
	}
}

func findHookSource(defs []store.HookEventDef, client, event string) string {
	for _, d := range defs {
		if d.Client == client && d.Event == event {
			return d.Source
		}
	}
	return ""
}

func hookEnabled(defs []store.HookEventDef, client, event string) bool {
	for _, d := range defs {
		if d.Client == client && d.Event == event {
			return d.Enabled
		}
	}
	return false
}
