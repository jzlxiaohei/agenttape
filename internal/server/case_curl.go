package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"tracelab/internal/store"
)

// curlResp is a copy-pasteable curl for a replay case, in one of two flavors.
type curlResp struct {
	Curl     string `json:"curl"`
	HasAuth  bool   `json:"has_auth"` // true only for direct mode (carries auth headers)
	Revealed bool   `json:"revealed"` // direct mode: whether secrets are shown vs masked
	// CredentialKind tells the UI which warning to show: "key" = an API key the
	// user typed (theirs to copy), "subscription" = a captured login session
	// token/cookie (more surprising to expose). Empty for proxy mode (no secret).
	CredentialKind string `json:"credential_kind,omitempty"`
}

// handleCaseCurl builds a curl for a case bound to a live session, in two modes:
//
//   - proxy (default): hits this server's per-session proxy (/s/<token>/…). The
//     proxy injects auth on forward, so the command carries NO secret — safe to
//     show and copy freely. Only works while the session is alive in this process.
//   - direct: hits upstream directly with the session's real auth headers — the
//     Postman-style portable command. Secrets are masked unless reveal=true, since
//     the real risk is accidental leakage (paste into an issue, screen-share), not
//     local snooping on 127.0.0.1.
func (s *Server) handleCaseCurl(st *store.Store, w http.ResponseWriter, r *http.Request) {
	c, err := st.GetCase(r.PathValue("id"))
	if err == store.ErrNoRows {
		http.Error(w, "case not found", http.StatusNotFound)
		return
	}
	if err != nil {
		httpError(w, err)
		return
	}
	var req struct {
		SessionID string  `json:"session_id"`
		Mode      string  `json:"mode"` // proxy | direct
		Reveal    bool    `json:"reveal"`
		Body      *string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.SessionID == "" {
		http.Error(w, "session_id required (it supplies routing and credentials)", http.StatusBadRequest)
		return
	}
	sess := s.Sessions.Get(req.SessionID)
	if sess == nil {
		http.Error(w, "no live launch session for that id; launch a session first", http.StatusConflict)
		return
	}
	body := c.Body
	if req.Body != nil {
		body = *req.Body
	}
	method := c.Method
	if method == "" {
		method = "POST"
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = store.EndpointForTarget(c.Provider, c.Target, sess.Upstream)
	}

	if req.Mode == "direct" {
		auth := s.Sessions.AuthFor(req.SessionID)
		if auth == nil {
			http.Error(w, replayCredentialConflictMessage(s, req.SessionID), http.StatusConflict)
			return
		}
		kind := "subscription"
		if s.Sessions.InjectFor(req.SessionID) != nil {
			kind = "key"
		}
		target := strings.TrimRight(sess.Upstream, "/") + "/" + strings.TrimPrefix(endpoint, "/")
		writeJSON(w, curlResp{
			Curl:           buildDirectCurl(method, target, body, auth, req.Reveal),
			HasAuth:        true,
			Revealed:       req.Reveal,
			CredentialKind: kind,
		})
		return
	}

	proxyTarget := "http://" + r.Host + "/s/" + sess.Token + "/" + strings.TrimPrefix(endpoint, "/")
	ct := "application/json"
	if auth := s.Sessions.AuthFor(req.SessionID); auth != nil && auth.Get("Content-Type") != "" {
		ct = auth.Get("Content-Type")
	}
	writeJSON(w, curlResp{Curl: buildProxyCurl(method, proxyTarget, ct, body)})
}

// buildProxyCurl renders the through-proxy command (no secret).
func buildProxyCurl(method, target, contentType, body string) string {
	var b strings.Builder
	b.WriteString("curl -X " + method + " " + shellQuote(target))
	b.WriteString(" \\\n  -H " + shellQuote("content-type: "+contentType))
	if body != "" {
		b.WriteString(" \\\n  --data-raw " + shellQuote(body))
	}
	return b.String()
}

// buildDirectCurl renders the upstream command with the session's real auth
// headers, masking sensitive values unless reveal is set.
func buildDirectCurl(method, target, body string, auth http.Header, reveal bool) string {
	var b strings.Builder
	b.WriteString("curl -X " + method + " " + shellQuote(target))

	keys := make([]string, 0, len(auth))
	for k := range auth {
		if skipCurlHeader(k) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range auth[k] {
			if !reveal && isSensitiveHeader(k) {
				v = maskHeaderValue(v)
			}
			b.WriteString(" \\\n  -H " + shellQuote(k+": "+v))
		}
	}
	if body != "" {
		b.WriteString(" \\\n  --data-raw " + shellQuote(body))
	}
	return b.String()
}

// skipCurlHeader drops hop-by-hop / client-owned headers that must not be pinned
// into a portable command (curl recomputes them).
func skipCurlHeader(k string) bool {
	switch strings.ToLower(k) {
	case "content-length", "host", "accept-encoding", "connection", "transfer-encoding":
		return true
	}
	return false
}

// isSensitiveHeader reports whether a header carries a credential we mask by
// default. Deliberately broad: explicit auth headers plus anything that looks
// like a key/token/cookie/secret, so an unfamiliar provider's auth still hides.
func isSensitiveHeader(k string) bool {
	k = strings.ToLower(k)
	switch k {
	case "authorization", "cookie", "set-cookie", "x-api-key", "api-key",
		"openai-api-key", "x-goog-api-key":
		return true
	}
	return strings.Contains(k, "auth") || strings.Contains(k, "key") ||
		strings.Contains(k, "token") || strings.Contains(k, "cookie") ||
		strings.Contains(k, "secret")
}

// maskHeaderValue hides the secret while keeping a recognizable auth scheme so the
// masked command still reads as valid (e.g. "Bearer ****").
func maskHeaderValue(v string) string {
	if scheme, rest, ok := strings.Cut(v, " "); ok && rest != "" {
		switch scheme {
		case "Bearer", "Basic", "Token":
			return scheme + " ****"
		}
	}
	return "****"
}
