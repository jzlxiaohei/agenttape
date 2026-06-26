package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jzlxiaohei/agenttape/internal/store"
)

// handleRenameSession sets a user-chosen name (label) for a captured session.
// Body: {"title": "..."}.
func (s *Server) handleRenameSession(st *store.Store, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := st.GetSession(id); err == store.ErrNoRows {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	} else if err != nil {
		httpError(w, err)
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := st.SetSessionLabel(id, strings.TrimSpace(req.Title)); err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteSession permanently removes a captured session and all its data
// (events, detail rows, tags, FTS entries, and the raw bytes on disk). The
// in-memory session (if one is still live for this id) is intentionally left
// alone: it dies with the process, and dropping it could break an active capture
// mid-flight. A still-live session that makes another request after deletion will
// simply re-appear — which is honest.
func (s *Server) handleDeleteSession(st *store.Store, w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := st.GetSession(id); err == store.ErrNoRows {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	} else if err != nil {
		httpError(w, err)
		return
	}
	if err := st.DeleteSession(id); err != nil {
		httpError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
