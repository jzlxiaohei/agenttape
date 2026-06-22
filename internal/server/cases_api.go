package server

import (
	"encoding/json"
	"net/http"
	"os"

	"tracelab/internal/source"
	"tracelab/internal/store"
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
	var req struct {
		EventID string `json:"event_id"`
		Name    string `json:"name"`
		Tags    string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EventID == "" {
		http.Error(w, "invalid add-case request", http.StatusBadRequest)
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
		Body:     string(body),
		Source:   "captured",
	}
	if err := st.AddCase(c); err != nil {
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
	auth := s.Sessions.AuthFor(req.SessionID)
	if auth == nil {
		http.Error(w, "no in-memory credentials for that session; launch a session first", http.StatusConflict)
		return
	}
	body := c.Body
	if req.Body != nil {
		body = *req.Body
	}
	out, err := s.executeReplay(r.Context(), c.Method, c.Target, []byte(body), auth, req.SessionID)
	if err != nil {
		http.Error(w, "run failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, out)
}
