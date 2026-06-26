package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agenttape/internal/normalize/providers"
	"agenttape/internal/server"
	"agenttape/internal/store"
)

type caseCurlResp struct {
	Curl           string `json:"curl"`
	HasAuth        bool   `json:"has_auth"`
	Revealed       bool   `json:"revealed"`
	CredentialKind string `json:"credential_kind"`
}

func postCurl(t *testing.T, base, sessionID, mode string, reveal bool) caseCurlResp {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"session_id": sessionID, "mode": mode, "reveal": reveal})
	resp, err := http.Post(base+"/api/cases/seed:codex-pure-text/curl", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("curl req: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("curl status=%d body=%s", resp.StatusCode, b)
	}
	var out caseCurlResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode curl: %v", err)
	}
	return out
}

func TestCaseCurlProxyVsDirect(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	srv := server.New(&memSink{}, providers.Registry())
	srv.EnableAPI(st)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	sess := registerSessionInfo(t, ts.URL, "codex_cli", up.URL)

	// Warm up through the proxy WITH a real auth header so the session captures
	// credentials (the same path AuthFor reads for direct curls).
	const secret = "supersecret-token-123"
	req, _ := http.NewRequest("POST", ts.URL+"/s/"+sess.Token+"/responses",
		strings.NewReader(`{"model":"gpt-5.5","input":"hi","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("warmup: %v", err)
	}
	resp.Body.Close()

	// Proxy mode: routes through /s/<token>/, carries NO secret.
	proxy := postCurl(t, ts.URL, sess.SessionID, "proxy", false)
	if !strings.Contains(proxy.Curl, "/s/"+sess.Token+"/responses") {
		t.Errorf("proxy curl missing session route: %s", proxy.Curl)
	}
	if strings.Contains(proxy.Curl, secret) {
		t.Errorf("proxy curl must not contain the secret: %s", proxy.Curl)
	}
	if proxy.HasAuth {
		t.Error("proxy curl should report has_auth=false")
	}

	// Direct mode, masked: hits upstream with auth, but the secret is hidden.
	masked := postCurl(t, ts.URL, sess.SessionID, "direct", false)
	if !strings.Contains(masked.Curl, up.URL+"/responses") {
		t.Errorf("direct curl missing upstream target: %s", masked.Curl)
	}
	if strings.Contains(masked.Curl, secret) {
		t.Errorf("masked curl leaked the secret: %s", masked.Curl)
	}
	if !strings.Contains(masked.Curl, "Bearer ****") {
		t.Errorf("masked curl should show 'Bearer ****': %s", masked.Curl)
	}
	if !masked.HasAuth || masked.Revealed {
		t.Errorf("masked curl flags wrong: %+v", masked)
	}
	if masked.CredentialKind != "subscription" {
		t.Errorf("captured-header session should be 'subscription', got %q", masked.CredentialKind)
	}

	// Direct mode, revealed: the real secret is present.
	revealed := postCurl(t, ts.URL, sess.SessionID, "direct", true)
	if !strings.Contains(revealed.Curl, secret) {
		t.Errorf("revealed curl should contain the secret: %s", revealed.Curl)
	}
	if !revealed.Revealed {
		t.Error("revealed curl should report revealed=true")
	}
}
