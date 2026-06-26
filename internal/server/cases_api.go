package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jzlxiaohei/agenttape/internal/source"
	"github.com/jzlxiaohei/agenttape/internal/store"
)

// handleListCases returns the replay library.
func (s *Server) handleListCases(st *store.Store, w http.ResponseWriter, _ *http.Request) {
	cases, err := st.ListCases()
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, cases)
}

// handleAddCase saves a captured event as a reusable case.
func (s *Server) handleAddCase(st *store.Store, w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid add-case request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var req struct {
		EventID  string          `json:"event_id"`
		Name     string          `json:"name"`
		Tags     string          `json:"tags"`
		Provider string          `json:"provider"`
		Method   string          `json:"method"`
		Target   string          `json:"target"`
		Endpoint string          `json:"endpoint"`
		Body     json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		http.Error(w, "invalid add-case request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.EventID == "" {
		bodyRaw := req.Body
		if len(bytes.TrimSpace(bodyRaw)) == 0 && isBareManualCasePayload(payload) {
			bodyRaw = payload
		}
		body, err := manualCaseBody(bodyRaw)
		if err != nil {
			http.Error(w, "invalid manual case body: "+err.Error(), http.StatusBadRequest)
			return
		}
		s.handleCreateManualCase(st, w, req.Name, req.Tags, req.Provider, req.Method, req.Target, req.Endpoint, body)
		return
	}
	detail, err := st.GetEvent(req.EventID)
	if err == store.ErrNoRows {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}
	if err != nil {
		httpError(w, err)
		return
	}
	path, err := st.RawFilePath(req.EventID, "request_body")
	if err != nil {
		httpError(w, err)
		return
	}
	body, err := os.ReadFile(path)
	if err != nil {
		httpError(w, err)
		return
	}
	name := req.Name
	if name == "" {
		name = detail.Model + " " + detail.Method
	}
	c := store.ReplayCase{
		ID:       source.RandomID(),
		Name:     name,
		Tags:     req.Tags,
		Provider: detail.Provider,
		Method:   detail.Method,
		Target:   detail.Target,
		Endpoint: store.EndpointForTarget(detail.Provider, detail.Target, sourceUpstream(st, detail.SessionID)),
		Body:     string(body),
		Source:   "captured",
	}
	if err := st.AddCase(c); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, c)
}

func isBareManualCasePayload(payload []byte) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(payload, &obj); err != nil {
		return false
	}
	for _, key := range []string{"event_id", "name", "tags", "provider", "method", "target", "endpoint", "body"} {
		if _, ok := obj[key]; ok {
			return false
		}
	}
	return len(obj) > 0
}

func (s *Server) handleCreateManualCase(st *store.Store, w http.ResponseWriter, name, tags, provider, method, target, endpoint, body string) {
	body = strings.TrimSpace(body)
	if body == "" || !json.Valid([]byte(body)) {
		http.Error(w, "invalid manual case request (valid JSON body required)", http.StatusBadRequest)
		return
	}
	provider, endpoint = normalizeManualCaseRoute(provider, endpoint, body)
	if method == "" {
		method = "POST"
	}
	if name == "" {
		name = defaultManualCaseName(body, method)
	}
	c := store.ReplayCase{
		ID:       source.RandomID(),
		Name:     name,
		Tags:     tags,
		Provider: provider,
		Method:   method,
		Target:   target,
		Endpoint: endpoint,
		Body:     body,
		Source:   "manual",
	}
	if err := st.AddCase(c); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, c)
}

func manualCaseBody(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", nil
	}
	var body string
	if err := json.Unmarshal(raw, &body); err == nil {
		return body, nil
	}
	return string(raw), nil
}

func normalizeManualCaseRoute(provider, endpoint, body string) (string, string) {
	if provider == "" {
		provider = guessManualProvider(body)
	}
	if endpoint == "" {
		if provider == "anthropic" {
			endpoint = "/v1/messages"
		} else {
			endpoint = "/responses"
		}
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	return provider, endpoint
}

func guessManualProvider(body string) string {
	var probe map[string]json.RawMessage
	if json.Unmarshal([]byte(body), &probe) != nil {
		return "openai-responses"
	}
	if _, ok := probe["messages"]; ok {
		if _, hasMax := probe["max_tokens"]; hasMax {
			return "anthropic"
		}
	}
	return "openai-responses"
}

func defaultManualCaseName(body, method string) string {
	var probe struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal([]byte(body), &probe)
	if probe.Model != "" {
		return probe.Model + " " + method
	}
	return "Manual " + method
}

// handleSnapshotCase saves an edited body as a NEW case (a snapshot) derived from
// an existing one. The original is never mutated — captured/seed cases stay as the
// baseline while edits accumulate as separate, independently re-runnable snapshots.
func (s *Server) handleSnapshotCase(st *store.Store, w http.ResponseWriter, r *http.Request) {
	base, err := st.GetCase(r.PathValue("id"))
	if err == store.ErrNoRows {
		http.Error(w, "case not found", http.StatusNotFound)
		return
	}
	if err != nil {
		httpError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		http.Error(w, "invalid snapshot request (body required)", http.StatusBadRequest)
		return
	}
	name := req.Name
	if name == "" {
		name = base.Name + " (snapshot)"
	}
	c := store.ReplayCase{
		ID:       source.RandomID(),
		Name:     name,
		Tags:     base.Tags,
		Provider: base.Provider,
		Method:   base.Method,
		Target:   base.Target,
		Endpoint: base.Endpoint,
		Body:     req.Body,
		Source:   "snapshot",
	}
	if err := st.AddCase(c); err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, c)
}

// handleDeleteCase removes a case. Built-in seed cases are deletable too: the
// library is the user's to curate, and seeding is gated on a content digest (see
// store.seedCases), so a deleted built-in stays gone across restarts — until the
// embedded seed set changes (a rebuild), which reinstalls the full built-in set.
func (s *Server) handleDeleteCase(st *store.Store, w http.ResponseWriter, r *http.Request) {
	c, err := st.GetCase(r.PathValue("id"))
	if err == store.ErrNoRows {
		http.Error(w, "case not found", http.StatusNotFound)
		return
	}
	if err != nil {
		httpError(w, err)
		return
	}
	// Built-in (seed) cases are not user-deletable. Refreshing the seed set is done
	// out-of-band via SQL (see REPLAY_LIB §3), not this API.
	if c.Source == "seed" {
		http.Error(w, "built-in cases cannot be deleted", http.StatusForbidden)
		return
	}
	if err := st.DeleteCase(c.ID); err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleOverwriteCase saves an edited body back onto the SAME case (in-place),
// unlike snapshot which forks a new one. Irreversible — the original body is
// replaced — so the UI gates it behind a confirm. Works for any case, built-in
// included, now that the library is user-owned.
func (s *Server) handleOverwriteCase(st *store.Store, w http.ResponseWriter, r *http.Request) {
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
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Body == "" {
		http.Error(w, "invalid overwrite request (body required)", http.StatusBadRequest)
		return
	}
	c.Body = req.Body
	if err := st.AddCase(*c); err != nil { // INSERT OR REPLACE on the same id
		httpError(w, err)
		return
	}
	writeJSON(w, c)
}

// handleRunCase runs a case against a chosen session (which supplies credentials),
// optionally with an edited body. Real billed call; result is not persisted.
func (s *Server) handleRunCase(st *store.Store, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := st.GetCase(id)
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
		Body      *string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.SessionID == "" {
		http.Error(w, "session_id required (it supplies the credentials)", http.StatusBadRequest)
		return
	}
	sess := s.Sessions.Get(req.SessionID)
	if sess == nil {
		http.Error(w, "no live launch session for that id; launch a session first", http.StatusConflict)
		return
	}
	auth := s.Sessions.AuthFor(req.SessionID)
	if auth == nil {
		http.Error(w, replayCredentialConflictMessage(s, req.SessionID), http.StatusConflict)
		return
	}
	body := c.Body
	if req.Body != nil {
		body = *req.Body
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = store.EndpointForTarget(c.Provider, c.Target, sess.Upstream)
	}
	out, err := s.executeCaseThroughSession(r.Context(), "http://"+r.Host, sess, c.Method, endpoint, []byte(body), auth)
	if err != nil {
		http.Error(w, "run failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, out)
}

func sourceUpstream(st *store.Store, sessionID string) string {
	ss, err := st.GetSession(sessionID)
	if err != nil {
		return ""
	}
	return ss.Upstream
}
